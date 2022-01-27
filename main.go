package main

import (
	"fmt"
	"os"

	"gorm.io/gorm"
)

//setup does pre-run setup configurations.
//	* Loads the application config from config.tml and parses
//	* Connects to the database and returns the db object
//	* Returns various values used throughout the application
func setup() (string, *gorm.DB, error) {
	config, err := GetConfig("./config.toml")
	if err != nil {
		fmt.Println("Error opening configuration file", err)
		return "", nil, err
	}

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

	apiHost, _, err := setup()

	if err != nil {
		fmt.Println("Error during application setup, exiting")
		os.Exit(1)
	}

	blocks := []int{5700793, 5700794, 5700795, 5700796}

	for _, block := range blocks {
		result, _ := GetBlockByHeight(apiHost, block)

		for _, v := range result.Block.BlockData.Txs {
			txhash := GetTxHash(v)

			tx, _ := GetTxByHash(apiHost, txhash)
			fmt.Printf("%+v\n", tx)
		}
	}
}
