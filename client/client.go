package main

import (
	"errors"
	"fmt"
	"os"

	configHelpers "cosmos-exporter/config"

	"cosmos-exporter/csv"
	dbTypes "cosmos-exporter/db"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var DB *gorm.DB

func setup() (*gorm.DB, error) {

	argConfig, err := configHelpers.ParseArgs(os.Stderr, os.Args[1:])

	if err != nil {
		return nil, err
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
		return nil, err
	}

	config := configHelpers.MergeConfigs(fileConfig, argConfig)
	logLevel := config.Log.Level
	db, err := dbTypes.PostgresDbConnect(config.Database.Host, config.Database.Port, config.Database.Database,
		config.Database.User, config.Database.Password, logLevel)
	if err != nil {
		fmt.Println("Could not establish connection to the database", err)
		return nil, err
	}

	dbTypes.CacheDenoms(db)

	return db, nil

}

func main() {

	db, err := setup()

	DB = db

	if err != nil {
		fmt.Println("Error setting up")
		fmt.Println(err)
		os.Exit(1)
	}

	r := gin.Default()
	r.GET("/events.csv", GetTaxableEventsCSV)
	r.Run(":8080")
}

func GetTaxableEventsCSV(c *gin.Context) {

	params := c.Request.URL.Query()
	address := params["address"][0]

	accountRows, err := csv.ParseForAddress(address, DB)

	if err != nil {
		c.Header("Access-Control-Allow-Origin", "*")
		c.JSON(404, gin.H{
			"message": "No events for the given address",
		})
		c.AbortWithError(404, errors.New("No events for the given address"))
		return
	}

	buffer := csv.ToCsv(accountRows)
	c.Header("Access-Control-Allow-Origin", "*")
	c.Data(200, "text/csv", buffer.Bytes())

}
