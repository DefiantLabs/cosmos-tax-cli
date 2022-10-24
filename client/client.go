package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	configHelpers "github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/csv"

	dbTypes "github.com/DefiantLabs/cosmos-tax-cli/db"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var DB *gorm.DB
var Config *config.Config

func setup() (*gorm.DB, *config.Config, error) {

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
	logLevel := config.Log.Level
	db, err := dbTypes.PostgresDbConnect(config.Database.Host, config.Database.Port, config.Database.Database,
		config.Database.User, config.Database.Password, logLevel)
	if err != nil {
		fmt.Println("Could not establish connection to the database", err)
		return nil, nil, err
	}

	dbTypes.CacheDenoms(db)

	return db, &config, nil

}

func main() {

	db, config, err := setup()

	DB = db
	Config = config

	if err != nil {
		fmt.Println("Error setting up")
		fmt.Println(err)
		os.Exit(1)
	}

	r := gin.Default()
	r.Use(CORSMiddleware())

	r.POST("/events.csv", GetTaxableEventsCSV)
	err = r.Run(":8080")
	if err != nil {
		log.Fatalf("Error starting server. Err: %v", err)
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		//Probably want to lock CORs down later, will need to know the hostname of the UI server
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

type TaxableEventsCSVRequest struct {
	Chain     string  `json:"chain"`
	Address   string  `json:"address"`
	StartDate *string `json:"startDate"` //can be null
	EndDate   *string `json:"endDate"`   //can be null
	Format    string  `json:"format"`
}

func GetTaxableEventsCSV(c *gin.Context) {
	var requestBody TaxableEventsCSVRequest
	err := c.BindJSON(&requestBody)
	if err != nil {
		err = c.AbortWithError(500, errors.New("Error processing request body"))
		log.Printf("Error calling AbortWithError. Err: %v", err)
		return
	}

	//We expect ISO 8601 dates in UTC
	var startDate string
	if requestBody.StartDate != nil {
		startDate = *requestBody.StartDate
	}

	var endDate string
	if requestBody.EndDate != nil {
		endDate = *requestBody.EndDate
	}

	if requestBody.Address == "" {
		c.JSON(422, gin.H{"message": "Address is required"})
		return
	}
	fmt.Printf("Start: %s End: %s\n", startDate, endDate)

	if requestBody.Format == "" {
		c.JSON(422, gin.H{"message": "Format is required"})
		return
	}

	accountRows, headers, err := csv.ParseForAddress(requestBody.Address, DB, requestBody.Format, *Config)
	if err != nil {
		c.AbortWithError(500, errors.New("Error getting rows for address"))
		return
	}

	if len(accountRows) == 0 {
		c.JSON(404, gin.H{"message": "No transactions for given address"})
		return
	}

	buffer := csv.ToCsv(accountRows, headers)
	c.Data(200, "text/csv", buffer.Bytes())
}
