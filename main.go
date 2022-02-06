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
func setup() (string, *gorm.DB, error) {

	argConfig, err := ParseArgs(os.Stderr, os.Args[1:])

	if err != nil {
		return "", nil, err
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
		return "", nil, err
	}

	config := MergeConfigs(fileConfig, argConfig)

	apiHost := config.Api.Host

	db, err := PostgresDbConnect(config.Database.Host, config.Database.Port, config.Database.Database, config.Database.User, config.Database.Password)
	if err != nil {
		fmt.Println("Could not establish connection to the database", err)
		return "", nil, err
	}

	//run database migrations at every runtime
	MigrateModels(db)

	return apiHost, db, nil

}

func main() {

	apiHost, db, err := setup()

	if err != nil {
		fmt.Println("Error during application setup, exiting")
		os.Exit(1)
	}

	dbConn, _ := db.DB()

	defer dbConn.Close()

	highestBlock := GetHighestBlock(db)

	var startHeight uint64 = 1
	if highestBlock.Height == 0 {
		fmt.Println("No blocks indexed, starting at block height 1")
	} else {
		fmt.Println("Found highest indexed block", highestBlock.Height)
		startHeight = highestBlock.Height + 1
	}

	for currBlock := startHeight; currBlock < startHeight+10000; currBlock++ {

		result, err := GetBlockByHeight(apiHost, currBlock)

		if err != nil {
			fmt.Println("Error getting block by height", err)
			os.Exit(1)
		}

		//consider optimizing by using block variable instead of parsing out (dangers?)
		height, err := strconv.ParseUint(result.Block.BlockHeader.Height, 10, 64)
		fmt.Println("Found block with height", result.Block.BlockHeader.Height)

		newBlock := Blocks{Height: height}

		time.Sleep(time.Second)

		var currTxs []SingleTx

		if len(result.Block.BlockData.Txs) == 0 {
			fmt.Println("Block has no transactions")
		} else {

			result, err := GetTxsByBlockHeight(apiHost, newBlock.Height)
			if err != nil {
				fmt.Println("Error getting transactions by block height", err)
				os.Exit(1)
			}

			for i, v := range result.Txs {

				//tx data and tx_response data are split, combine into 1
				var currTx SingleTx

				currTxResponse := result.TxResponses[i]

				currTx.TxResponse = currTxResponse
				currTx.Tx = v

				currTxs = append(currTxs, currTx)
			}

			time.Sleep(time.Second)

		}

		//do one db storage request at end of requests
		db.Create(&newBlock)

		for _, tx := range currTxs {
			IndexNewTx(db, tx, newBlock)
		}

	}
}
