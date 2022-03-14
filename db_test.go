package main

import (
	configUtils "cosmos-exporter/config"
	"cosmos-exporter/csv"
	"cosmos-exporter/db"
	dbUtils "cosmos-exporter/db"
	"fmt"
	"os"
	"testing"

	"gorm.io/gorm"
)

//setup does pre-run setup configurations.
//	* Loads the application config from config.tml, cli args and parses/merges
//	* Connects to the database and returns the db object
//	* Returns various values used throughout the application
func db_setup() (*gorm.DB, error) {
	config, err := configUtils.GetConfig("./config.toml")

	if err != nil {
		fmt.Println("Error opening configuration file", err)
		return nil, err
	}

	db, err := dbUtils.PostgresDbConnectLogInfo(config.Database.Host, config.Database.Port, config.Database.Database, config.Database.User, config.Database.Password)
	if err != nil {
		fmt.Println("Could not establish connection to the database", err)
		return nil, err
	}

	//TODO: create config values for the prefixes here
	//Could potentially check Node info at startup and pass in ourselves?
	setupAddressRegex("juno(valoper)?1[a-z0-9]{38}")
	setupAddressPrefix("juno")

	//run database migrations at every runtime
	dbUtils.MigrateModels(db)

	return db, nil

}

func TestCsvForAddress(t *testing.T) {
	gorm, _ := db_setup()
	address := "juno1txpxafd7q96nkj5jxnt7qnqy4l0rrjyuv6dgte"
	csvRows, err := csv.ParseForAddress(address, gorm)
	if err != nil || len(csvRows) == 0 {
		t.Fatal("Failed to lookup taxable events")
	}

	buffer := csv.ToCsv(csvRows)
	if len(buffer.Bytes()) == 0 {
		t.Fatal("CSV length should never be 0, there are always headers!")
	}

	err = os.WriteFile("accointing.csv", buffer.Bytes(), 0644)
	if err != nil {
		t.Fatal("Failed to write CSV to disk")
	}
}

func TestLookupTxForAddresses(t *testing.T) {
	gorm, _ := db_setup()
	//"juno1txpxafd7q96nkj5jxnt7qnqy4l0rrjyuv6dgte"
	//juno1mt72y3jny20456k247tc5gf2dnat76l4ynvqwl
	taxableEvts, err := db.GetTaxableEvents("juno1txpxafd7q96nkj5jxnt7qnqy4l0rrjyuv6dgte", gorm)
	if err != nil || len(taxableEvts) == 0 {
		t.Fatal("Failed to lookup taxable events")
	}
}
