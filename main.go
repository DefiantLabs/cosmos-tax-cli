package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gorm.io/gorm"
)

//setup does pre-run setup configurations.
//	* Loads the application config from config.tml, cli args and parses/merges
//	* Connects to the database and returns the db object
//	* Returns various values used throughout the application
func setup() (string, *gorm.DB, uint64, error) {

	argConfig, err := ParseArgs(os.Stderr, os.Args[1:])

	if err != nil {
		return "", nil, 1, err
	}

	var location string
	if argConfig.ConfigFileLocation != "" {
		location = argConfig.ConfigFileLocation
	} else {
		location = "./config.toml"
	}

	fileConfig, err := GetConfig(location)

	if err != nil {
		fmt.Println("Error opening configuration file", err)
		return "", nil, 1, err
	}

	config := MergeConfigs(fileConfig, argConfig)

	apiHost := config.Api.Host
	startingBlock := config.Base.StartBlock

	//0 is an invalid starting block, set it to 1
	if startingBlock == 0 {
		startingBlock = 1
	}

	db, err := PostgresDbConnect(config.Database.Host, config.Database.Port, config.Database.Database, config.Database.User, config.Database.Password)
	if err != nil {
		fmt.Println("Could not establish connection to the database", err)
		return "", nil, 1, err
	}

	setupAddressRegex("juno(valoper)?1[a-z0-9]{38}")
	//run database migrations at every runtime
	MigrateModels(db)

	return apiHost, db, startingBlock, nil

}

func main() {

	apiHost, db, startingBlock, err := setup()

	if err != nil {
		fmt.Println("Error during application setup, exiting")
		os.Exit(1)
	}

	dbConn, _ := db.DB()

	//is this needed? probably handled by gorm but no idea
	defer dbConn.Close()

	var latestBlock uint64 = 1

	resp, err := GetLatestBlock(apiHost)

	if err != nil {
		fmt.Println("Error getting latest block", err)
		os.Exit(1)
	}

	latestBlock, err = strconv.ParseUint(resp.Block.BlockHeader.Height, 10, 64)

	if err != nil {
		fmt.Println("Error getting latest block", err)
		os.Exit(1)
	}

	fmt.Println("Found latest block", latestBlock)

	highestBlock := GetHighestIndexedBlock(db)

	var startHeight uint64 = startingBlock
	if highestBlock.Height == 0 {
		fmt.Printf("No blocks indexed, starting at block height from the base configuration %d\n", startHeight)
	} else {
		fmt.Println("Found highest indexed block", highestBlock.Height)
		startHeight = highestBlock.Height + 1
	}

	currBlock := startHeight
	for ; ; currBlock++ {

		//need to sleep for a bit to wait for next block to be indexed
		if currBlock == latestBlock {
			for {
				resp, err := GetLatestBlock(apiHost)

				if err != nil {
					fmt.Println("Error getting latest block", err)
					os.Exit(1)
				}

				newLatestBlock, err := strconv.ParseUint(resp.Block.BlockHeader.Height, 10, 64)

				if err != nil {
					fmt.Println("Error getting latest block", err)
					os.Exit(1)
				}

				if currBlock == newLatestBlock {
					time.Sleep(1)
				} else {
					fmt.Printf("New hightest block found %d, restarting indexer\n", newLatestBlock)
					latestBlock = newLatestBlock
					break
				}
			}
		}

		result, err := GetBlockByHeight(apiHost, currBlock)

		if err != nil {
			fmt.Println("Error getting block by height", err)
			os.Exit(1)
		}

		//consider optimizing by using block variable instead of parsing out (dangers?)
		height, err := strconv.ParseUint(result.Block.BlockHeader.Height, 10, 64)
		fmt.Println("Found block with height", result.Block.BlockHeader.Height)

		newBlock := Block{Height: height}

		time.Sleep(time.Second)

		var txsWithAddresses []TxWithAddresses

		if len(result.Block.BlockData.Txs) == 0 {
			fmt.Println("Block has no transactions")
		} else {

			result, err := GetTxsByBlockHeight(apiHost, newBlock.Height)
			if err != nil {
				fmt.Println("Error getting transactions by block height", err)
				os.Exit(1)
			}

			fmt.Printf("Block has %s transcation(s)\n", result.Pagination.Total)

			txsWithAddresses = ProcessTxs(result.Txs, result.TxResponses)

			time.Sleep(time.Second)

		}

		err = IndexNewBlock(db, newBlock, txsWithAddresses)

		if err != nil {
			fmt.Println("Error indexing block", err)
			os.Exit(1)
		}

		fmt.Printf("Finished indexing block %d\n", currBlock)

	}
}
