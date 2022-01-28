package main

import (
	"fmt"
	"os"

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

	for block := 5700793; block < 5701060; block++ {
		result, _ := GetBlockByHeight(apiHost, block)

		for _, v := range result.Block.BlockData.Txs {
			txhash := GetTxHash(v)

			tx, _ := GetTxByHash(apiHost, txhash)
			fmt.Printf("%+v\n", tx)
		}
	}
}
