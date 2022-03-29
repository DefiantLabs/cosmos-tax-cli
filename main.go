package main

import (
	"cosmos-exporter/core"
	"cosmos-exporter/rpc"
	"cosmos-exporter/tasks"
	"fmt"
	"os"
	"time"

	configHelpers "cosmos-exporter/config"
	dbTypes "cosmos-exporter/db"

	"github.com/go-co-op/gocron"
	"github.com/strangelove-ventures/lens/client"
	"gorm.io/gorm"
)

//TODO: Refactor all of this code. Move to config folder, make it work for multiple chains.
//Separate the DB logic, scheduler logic, and blockchain logic into different functions.
//
//setup does pre-run setup configurations.
//	* Loads the application config from config.tml, cli args and parses/merges
//	* Connects to the database and returns the db object
//	* Returns various values used throughout the application
func setup() (*configHelpers.Config, *gorm.DB, *gocron.Scheduler, error) {

	argConfig, err := configHelpers.ParseArgs(os.Stderr, os.Args[1:])

	if err != nil {
		return nil, nil, nil, err
	}

	var location string
	if argConfig.ConfigFileLocation != "" {
		location = argConfig.ConfigFileLocation
	} else {
		location = "./config.toml"
	}

	fileConfig, err := configHelpers.GetConfig(location)

	if err != nil {
		fmt.Println("Error opening configuration file", err)
		return nil, nil, nil, err
	}

	config := configHelpers.MergeConfigs(fileConfig, argConfig)

	//0 is an invalid starting block, set it to 1
	if config.Base.StartBlock == 0 {
		config.Base.StartBlock = 1
	}

	db, err := dbTypes.PostgresDbConnect(config.Database.Host, config.Database.Port, config.Database.Database,
		config.Database.User, config.Database.Password, config.Log.Level)

	sqldb, _ := db.DB()
	sqldb.SetMaxIdleConns(10)
	sqldb.SetMaxOpenConns(100)
	sqldb.SetConnMaxLifetime(time.Hour)

	if err != nil {
		fmt.Println("Could not establish connection to the database", err)
	}

	//TODO: make mapping for all chains, globally initialized
	core.SetupAddressRegex("juno(valoper)?1[a-z0-9]{38}")
	core.SetupAddressPrefix("juno")

	scheduler := gocron.NewScheduler(time.UTC)

	//run database migrations at every runtime
	err = dbTypes.MigrateModels(db)
	if err != nil {
		return nil, nil, nil, err
	}
	return &config, db, scheduler, nil
}

func GetIndexerStartingHeight(configStartHeight int64, cl *client.ChainClient, db *gorm.DB) int64 {
	//Start the indexer at the configured value if one has been set. This starting height will be used
	//instead of searching the database to find the last indexed block.
	if configStartHeight != -1 {
		return configStartHeight
	}

	latestBlock, bErr := rpc.GetLatestBlockHeight(cl)
	if bErr != nil {
		fmt.Println("Error getting blockchain latest height, exiting")
		os.Exit(1)
	}

	fmt.Println("Found latest block", latestBlock)
	highestIndexedBlock := dbTypes.GetHighestIndexedBlock(db)
	if highestIndexedBlock.Height < latestBlock {
		return highestIndexedBlock.Height + 1
	}

	return latestBlock

}

func main() {

	config, db, scheduler, err := setup()

	if err != nil {
		fmt.Println("Error during application setup, exiting")
		os.Exit(1)
	}

	apiHost := config.Lens.Rpc
	dbConn, _ := db.DB()
	defer dbConn.Close()

	//TODO may need to run this task in setup() so that we have a cold start functionality before the indexer starts
	scheduler.Every(6).Hours().Do(tasks.DenomUpsertTask, apiHost, db)
	scheduler.StartAsync()

	cl := configHelpers.GetLensClient(config.Lens)
	configHelpers.SetChainConfig("juno")

	//Depending on the app configuration, wait for the chain to catch up
	chainCatchingUp, qErr := rpc.IsCatchingUp(cl)
	for config.Base.WaitForChain && chainCatchingUp && qErr == nil {
		//Wait between status checks, don't spam the node with requests
		time.Sleep(time.Second * time.Duration(config.Base.WaitForChainDelay))
		chainCatchingUp, qErr = rpc.IsCatchingUp(cl)
	}

	if qErr != nil {
		fmt.Print("Error querying chain status, exiting")
		os.Exit(1)
	}

	latestBlock, bErr := rpc.GetLatestBlockHeight(cl)
	if bErr != nil {
		fmt.Println(bErr)
		os.Exit(1)
	}

	//Start at the last indexed block height (or the block height in the config, if set)
	startHeight := GetIndexerStartingHeight(config.Base.StartBlock, cl, db)
	currBlock := startHeight
	lastBlock := config.Base.EndBlock
	numBlocksTimed := config.Base.BlockTimer
	blocksProcessed := 0
	timeStart := time.Now()

	for ; ; currBlock++ {
		//Just measuring how many blocks/second we can process
		if numBlocksTimed > 0 {
			blocksProcessed++
			if blocksProcessed%int(numBlocksTimed) == 0 {
				totalTime := time.Since(timeStart)
				fmt.Printf("Processing %d blocks (%d-%d) took %f seconds\n", numBlocksTimed, currBlock-numBlocksTimed, currBlock, totalTime.Seconds())
				fmt.Printf("%d total blocks have been processed by this indexer.\n", blocksProcessed)
				timeStart = time.Now()
			}
		}

		//Self throttling in case of hitting public APIs
		if config.Base.Throttling != 0 {
			time.Sleep(time.Second * time.Duration(config.Base.Throttling))
		}

		//Already at the latest block, wait for the next block to be available.
		for currBlock == latestBlock {
			latestBlock, bErr = rpc.GetLatestBlockHeight(cl)
			if bErr != nil {
				fmt.Println(bErr)
				os.Exit(1)
			}
			if config.Base.Throttling != 0 {
				time.Sleep(time.Second * time.Duration(config.Base.Throttling))
			}
		}

		if err != nil {
			fmt.Println("Error getting block by height", err)
			os.Exit(1)
		}

		newBlock := dbTypes.Block{Height: currBlock}
		var txDBWrappers []dbTypes.TxDBWrapper

		//TODO: There is currently no pagination implemented!
		//TODO: Do something smarter than giving up when we encounter an error.
		txsEventResp, err := rpc.GetTxsByBlockHeight(cl, newBlock.Height)
		if err != nil {
			fmt.Println("Error getting transactions by block height", err)
			os.Exit(1)
		}

		txDBWrappers = core.ProcessRpcTxs(txsEventResp)

		//While debugging we'll sometimes want to turn off INSERTS to the DB
		//Note that this does not turn off certain reads or DB connections.
		if config.Base.IndexingEnabled {
			err = dbTypes.IndexNewBlock(db, newBlock, txDBWrappers)
		}

		if err != nil {
			fmt.Printf("Error %s indexing block %d\n", err, currBlock)
			os.Exit(1)
		}

		if lastBlock != -1 && currBlock >= lastBlock {
			fmt.Println("Hit the last block, exiting.")
			break
		}
	}

	//If we error out in the main loop, this will block. Meaning we may not know of an error for 6 hours until last scheduled task stops
	scheduler.Stop()
}
