package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/DefiantLabs/cosmos-exporter/core"
	"github.com/DefiantLabs/cosmos-exporter/osmosis"
	"github.com/DefiantLabs/cosmos-exporter/rpc"
	"github.com/DefiantLabs/cosmos-exporter/tasks"

	configHelpers "github.com/DefiantLabs/cosmos-exporter/config"
	indexerTx "github.com/DefiantLabs/cosmos-exporter/cosmos/modules/tx"
	dbTypes "github.com/DefiantLabs/cosmos-exporter/db"

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

//OsmosisGetRewardsStartIndexHeight Not yet implemented. Search the DB and get the last indexed rewards height, plus 1.
//If nothing has been indexed yet, the start height should be 0.
func OsmosisGetRewardsStartIndexHeight() int64 {
	return -1
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
	for (config.Base.WaitForChain || config.Base.ExitWhenCaughtUp) && chainCatchingUp && qErr == nil {
		//Wait between status checks, don't spam the node with requests
		time.Sleep(time.Second * time.Duration(config.Base.WaitForChainDelay))
		chainCatchingUp, qErr = rpc.IsCatchingUp(cl)
	}

	if qErr != nil {
		fmt.Print("Error querying chain status, exiting")
		os.Exit(1)
	}

	//Jobs are just the block height; limit max jobs in the queue, otherwise this queue would contain one
	//item (block height) for every block on the entire blockchain we're indexing. Furthermore, once the queue
	//is close to empty, we will spin up a new thread to fill it up with new jobs.
	blockHeightToProcess := make(chan int64, 10000)

	//This channel represents query job results for the RPC queries to Cosmos Nodes. Every time an RPC query
	//completes, the query result will be sent to this channel (for later processing by a different thread).
	//Realistically, I expect that RPC queries will be slower than our relational DB on the local network.
	//If RPC queries are faster than DB inserts this buffer will fill up.
	//We will periodically check the buffer size to monitor performance so we can optimize later.
	jobResultsChannel := make(chan *indexerTx.GetTxsEventResponseWrapper, 10)
	rpcQueryThreads := 4

	//Spin up a (configurable) number of threads to query RPC endpoints for Transactions.
	for i := 0; i < rpcQueryThreads; i++ {
		go QueryRpc(blockHeightToProcess, jobResultsChannel, cl)
	}

	//Start a thread to process transactions after the RPC querier retrieves them.
	go ProcessTxs(jobResultsChannel, config.Base.BlockTimer, config.Base.IndexingEnabled, db)

	//Start at the last indexed block height (or the block height in the config, if set)
	currBlock := GetIndexerStartingHeight(config.Base.StartBlock, cl, db)
	//Don't index past this block no matter what
	lastBlock := config.Base.EndBlock

	if configHelpers.IsOsmosis(config) {
		rewardsIndexerStartHeight := OsmosisGetRewardsStartIndexHeight()
		latestOsmosisBlock, bErr := rpc.GetLatestBlockHeight(cl)
		if bErr != nil {
			fmt.Println("Error getting blockchain latest height, exiting")
			os.Exit(1)
		}

		rpcClient := osmosis.URIClient{
			Address: cl.Config.RPCAddr,
			Client:  &http.Client{},
		}
		go rpcClient.IndexEpochsBetween(rewardsIndexerStartHeight, latestOsmosisBlock)
	}

	//Add jobs to the queue to be processed
	for {
		//The program is configured to stop running after a set block height.
		//Generally this will only be done while debugging or if a particular block was incorrectly processed.
		if (lastBlock != -1 || config.Base.ExitWhenCaughtUp) && currBlock >= lastBlock {
			fmt.Println("Hit the last block we're allowed to index, exiting.")
			break
		}

		//The job queue is running out of jobs to process, see if the blockchain has produced any new blocks we haven't indexed yet.
		if len(blockHeightToProcess) <= cap(blockHeightToProcess)/4 {
			//fmt.Println("Filling jobs queue")

			//This is the latest block height available on the Node.
			latestBlock, bErr := rpc.GetLatestBlockHeight(cl)
			if bErr != nil {
				fmt.Println(bErr)
				os.Exit(1)
			}

			//Throttling in case of hitting public APIs
			//TODO: track tx/s downloaded from each RPC endpoint and implement throttling limits per endpoint.
			if config.Base.Throttling != 0 {
				time.Sleep(time.Second * time.Duration(config.Base.Throttling))
			}

			//Already at the latest block, wait for the next block to be available.
			for currBlock <= latestBlock && len(blockHeightToProcess) != cap(blockHeightToProcess) {

				if config.Base.Throttling != 0 {
					time.Sleep(time.Second * time.Duration(config.Base.Throttling))
				}

				//Add the new block to the queue
				//fmt.Printf("Added block %d to the queue\n", currBlock)
				blockHeightToProcess <- currBlock
				currBlock++
			}
		}
	}

	//if len(ch) == cap(ch) {

	//If we error out in the main loop, this will block. Meaning we may not know of an error for 6 hours until last scheduled task stops
	scheduler.Stop()
}

func QueryRpc(blockHeightToProcess chan int64, results chan *indexerTx.GetTxsEventResponseWrapper, cl *client.ChainClient) {
	for {
		blockToProcess := <-blockHeightToProcess
		//fmt.Printf("Querying RPC transactions for block %d\n", blockToProcess)
		newBlock := dbTypes.Block{Height: blockToProcess}

		//TODO: There is currently no pagination implemented!
		//TODO: Do something smarter than giving up when we encounter an error.
		txsEventResp, err := rpc.GetTxsByBlockHeight(cl, newBlock.Height)
		if err != nil {
			fmt.Println("Error getting transactions by block height", err)
			os.Exit(1)
		}

		res := &indexerTx.GetTxsEventResponseWrapper{
			CosmosGetTxsEventResponse: txsEventResp,
			Height:                    blockToProcess,
		}
		results <- res
	}
}

func ProcessTxs(results chan *indexerTx.GetTxsEventResponseWrapper, numBlocksTimed int64, indexingEnabled bool, db *gorm.DB) {
	blocksProcessed := 0
	timeStart := time.Now()

	for {
		txToProcess := <-results
		txDBWrappers := core.ProcessRpcTxs(txToProcess.CosmosGetTxsEventResponse)
		newBlock := dbTypes.Block{Height: txToProcess.Height}

		//While debugging we'll sometimes want to turn off INSERTS to the DB
		//Note that this does not turn off certain reads or DB connections.
		if indexingEnabled {
			fmt.Printf("Indexing block %d, threaded.\n", newBlock.Height)
			err := dbTypes.IndexNewBlock(db, newBlock, txDBWrappers)
			if err != nil {
				fmt.Printf("Error %s indexing block %d\n", err, newBlock.Height)
				os.Exit(1)
			}
		}

		//Just measuring how many blocks/second we can process
		if numBlocksTimed > 0 {
			blocksProcessed++
			if blocksProcessed%int(numBlocksTimed) == 0 {
				totalTime := time.Since(timeStart)
				fmt.Printf("Processing %d blocks took %f seconds. %d total blocks have been processed.\n", numBlocksTimed, totalTime.Seconds(), blocksProcessed)
				timeStart = time.Now()
			}
		}

	}
}
