package main

import (
	"fmt"
	"os"
	"testing"

	configUtils "github.com/DefiantLabs/cosmos-exporter/config"
	"github.com/DefiantLabs/cosmos-exporter/core"
	"github.com/DefiantLabs/cosmos-exporter/csv"
	"github.com/DefiantLabs/cosmos-exporter/db"
	dbUtils "github.com/DefiantLabs/cosmos-exporter/db"

	"gorm.io/gorm"
)

//setup does pre-run setup configurations.
//	* Loads the application config from config.tml, cli args and parses/merges
//	* Connects to the database and returns the db object
//	* Returns various values used throughout the application
func db_setup(addressRegex string, addressPrefix string) (*gorm.DB, error) {
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
	core.SetupAddressRegex(addressRegex)
	core.SetupAddressPrefix(addressPrefix)

	//run database migrations at every runtime
	dbUtils.MigrateModels(db)
	dbUtils.CacheDenoms(db) //Have to cache denoms to get translations from e.g. ujuno to Juno
	return db, nil

}

func TestGetOsmosisRewardIndex(t *testing.T) {
	addressRegex := "osmo(valoper)?1[a-z0-9]{38}"
	addressPrefix := "osmo"
	gorm, _ := db_setup(addressRegex, addressPrefix)
	block, err := db.GetHighestTaxableEventBlock(gorm, "test-1")
	if err != nil {
		t.Fail()
	}

	fmt.Println(block.Height)
}

func TestCsvForAddress(t *testing.T) {
	addressRegex := "juno(valoper)?1[a-z0-9]{38}"
	addressPrefix := "juno"
	gorm, _ := db_setup(addressRegex, addressPrefix)
	//address := "juno1mt72y3jny20456k247tc5gf2dnat76l4ynvqwl"
	//address := "juno130mdu9a0etmeuw52qfxk73pn0ga6gawk4k539x" //strangelove's delegator
	address := "juno1m2hg5t7n8f6kzh8kmh98phenk8a4xp5wyuz34y" //local test key address
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
	addressRegex := "juno(valoper)?1[a-z0-9]{38}"
	addressPrefix := "juno"
	gorm, _ := db_setup(addressRegex, addressPrefix)
	//"juno1txpxafd7q96nkj5jxnt7qnqy4l0rrjyuv6dgte"
	//juno1mt72y3jny20456k247tc5gf2dnat76l4ynvqwl
	taxableEvts, err := db.GetTaxableEvents("juno1txpxafd7q96nkj5jxnt7qnqy4l0rrjyuv6dgte", gorm)
	if err != nil || len(taxableEvts) == 0 {
		t.Fatal("Failed to lookup taxable events")
	}
}
