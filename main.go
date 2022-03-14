package main

import (
	"cosmos-exporter/rest"
	"fmt"
	"os"
	"strconv"
	"time"

	configHelpers "cosmos-exporter/config"
	dbTypes "cosmos-exporter/db"

	"gorm.io/gorm"
)

//setup does pre-run setup configurations.
//	* Loads the application config from config.tml, cli args and parses/merges
//	* Connects to the database and returns the db object
//	* Returns various values used throughout the application
func setup() (*configHelpers.Config, *gorm.DB, error) {

	argConfig, err := configHelpers.ParseArgs(os.Stderr, os.Args[1:])

	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
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

	//TODO: create config values for the prefixes here
	//Could potentially check Node info at startup and pass in ourselves?
	setupAddressRegex("juno(valoper)?1[a-z0-9]{38}")
	setupAddressPrefix("juno")

	//run database migrations at every runtime
	dbTypes.MigrateModels(db)
	return &config, db, nil
}

func main() {

	config, db, err := setup()

	if err != nil {
		fmt.Println("Error during application setup, exiting")
		os.Exit(1)
	}

	apiHost := config.Api.Host
	dbConn, _ := db.DB()
	defer dbConn.Close()

	latestBlock := rest.GetLatestBlockHeight(apiHost)
	startHeight := rest.GetBlockStartHeight(config, db)
	currBlock := startHeight

	for ; ; currBlock++ {
		//Self throttling in case of hitting public APIs
		if config.Base.Throttling != 0 {
			time.Sleep(time.Second * time.Duration(config.Base.Throttling))
		}

		//need to sleep for a bit to wait for next block to be indexed
		for currBlock == latestBlock {
			latestBlock = rest.GetLatestBlockHeight(apiHost)
			if config.Base.Throttling != 0 {
				time.Sleep(time.Second * time.Duration(config.Base.Throttling))
			}
		}

		result, err := rest.GetBlockByHeight(apiHost, currBlock)

		if err != nil {
			fmt.Println("Error getting block by height", err)
			os.Exit(1)
		}

		//consider optimizing by using block variable instead of parsing out (dangers?)
		height, _ := strconv.ParseUint(result.Block.BlockHeader.Height, 10, 64)
		newBlock := dbTypes.Block{Height: height}

		var txDBWrappers []dbTypes.TxDBWrapper

		if len(result.Block.BlockData.Txs) == 0 {
			//fmt.Println("Block has no transactions")
		} else {
			result, err := rest.GetTxsByBlockHeight(apiHost, newBlock.Height)
			if err != nil {
				fmt.Println("Error getting transactions by block height", err)
				os.Exit(1)
			}

			fmt.Printf("Block %d has %s transaction(s)\n", height, result.Pagination.Total)
			txDBWrappers = ProcessTxs(result.Txs, result.TxResponses)
		}

		err = dbTypes.IndexNewBlock(db, newBlock, txDBWrappers)

		if err != nil {
			fmt.Printf("Error %s indexing block %d\n", err, height)
			os.Exit(1)
		}
	}
}
