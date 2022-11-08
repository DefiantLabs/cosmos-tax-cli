package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
	"github.com/DefiantLabs/cosmos-tax-cli-private/csv"
	dbTypes "github.com/DefiantLabs/cosmos-tax-cli-private/db"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var DB *gorm.DB
var GlobalCfg *config.Config

func setup() (*gorm.DB, *config.Config, error) {
	argConfig, err := config.ParseArgs(os.Stderr, os.Args[1:])
	if err != nil {
		return nil, nil, err
	}

	var location string
	if argConfig.ConfigFileLocation != "" {
		location = argConfig.ConfigFileLocation
	} else {
		location = "./config.toml"
	}

	fileConfig, err := config.GetConfig(location)
	if err != nil {
		config.Log.Error("Error opening configuration file.", zap.Error(err))
		return nil, nil, err
	}

	cfg := config.MergeConfigs(fileConfig, argConfig)
	logLevel := cfg.Log.Level
	db, err := dbTypes.PostgresDbConnect(cfg.Database.Host, cfg.Database.Port, cfg.Database.Database, cfg.Database.User, cfg.Database.Password, logLevel)
	if err != nil {
		config.Log.Error("Could not establish connection to the database", zap.Error(err))
		return nil, nil, err
	}

	dbTypes.CacheDenoms(db)

	return db, &cfg, nil
}

func main() {
	db, cfg, err := setup()
	if err != nil {
		log.Fatalf("Error setting up. Err: %v", err)
	}

	DB = db
	GlobalCfg = cfg

	r := gin.Default()
	r.Use(CORSMiddleware())

	r.POST("/events.csv", GetTaxableEventsCSV)
	err = r.Run(":8080")
	if err != nil {
		config.Log.Fatal("Error starting server.", zap.Error(err))
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Probably want to lock CORs down later, will need to know the hostname of the UI server
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
	StartDate *string `json:"startDate"` // can be null
	EndDate   *string `json:"endDate"`   // can be null
	Format    string  `json:"format"`
}

func GetTaxableEventsCSV(c *gin.Context) {
	var requestBody TaxableEventsCSVRequest
	err := c.BindJSON(&requestBody)
	if err != nil {
		// the error returned here has already been pushed to the context... I think.
		c.AbortWithError(500, errors.New("Error processing request body")) // nolint:staticcheck,errcheck
		return
	}

	// We expect ISO 8601 dates in UTC
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

	accountRows, headers, err := csv.ParseForAddress(requestBody.Address, DB, requestBody.Format, *GlobalCfg)
	if err != nil {
		// the error returned here has already been pushed to the context... I think.
		c.AbortWithError(500, errors.New("Error getting rows for address")) // nolint:staticcheck,errcheck
		return
	}

	if len(accountRows) == 0 {
		c.JSON(404, gin.H{"message": "No transactions for given address"})
		return
	}

	buffer := csv.ToCsv(accountRows, headers)
	c.Data(200, "text/csv", buffer.Bytes())
}
