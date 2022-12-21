package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
	"github.com/DefiantLabs/cosmos-tax-cli-private/csv"
	dbTypes "github.com/DefiantLabs/cosmos-tax-cli-private/db"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var DB *gorm.DB
var GlobalCfg *config.Config

func setup() (*gorm.DB, *config.Config, int, error) {
	argConfig, flagSet, svcPort, err := config.ParseArgs(os.Stderr, os.Args[1:])
	if err != nil {
		if strings.Contains(err.Error(), "help requested") {
			log.Println("Please see valid flags above.")
			os.Exit(0)
		} else if strings.Contains(err.Error(), "flag provided but not defined") {
			log.Println("Invalid flag. Please see valid flags above.")
			os.Exit(0)
		}
		log.Panicf("Error parsing args. Err: %v", err)
		return nil, nil, svcPort, err
	}

	var location string
	if argConfig.ConfigFileLocation != "" {
		location = argConfig.ConfigFileLocation
	} else {
		location = "./config.toml"
	}

	fileConfig, err := config.GetConfig(location)
	if err != nil {
		if !strings.Contains(err.Error(), "no such file or directory") {
			log.Panicf("Error opening configuration file. Err: %v", err)
			return nil, nil, svcPort, err
		}
	}

	// merge and validate configs
	cfg := config.MergeConfigs(fileConfig, argConfig)
	err = cfg.ValidateClientConfig()
	if err != nil {
		flagSet.PrintDefaults()
		log.Fatalf("Config validation failed. Err: %v", err)
	}

	// Configure logger
	logLevel := cfg.Log.Level
	logPath := cfg.Log.Path
	prettyLogging := cfg.Log.Pretty
	config.DoConfigureLogger(logPath, logLevel, prettyLogging)

	// Configure DB
	db, err := dbTypes.PostgresDbConnect(cfg.Database.Host, cfg.Database.Port, cfg.Database.Database, cfg.Database.User, cfg.Database.Password, strings.ToLower(cfg.Database.LogLevel))
	if err != nil {
		config.Log.Error("Could not establish connection to the database", err)
		return nil, nil, svcPort, err
	}

	dbTypes.CacheDenoms(db)

	return db, &cfg, svcPort, nil
}

func main() {
	db, cfg, svcPort, err := setup()
	if err != nil {
		log.Fatalf("Error setting up. Err: %v", err)
	}

	DB = db
	GlobalCfg = cfg

	r := gin.Default()
	r.Use(CORSMiddleware())

	r.POST("/events.csv", GetTaxableEventsCSV)
	err = r.Run(fmt.Sprintf(":%v", svcPort))
	if err != nil {
		config.Log.Fatal("Error starting server.", err)
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
	Addresses string  `json:"addresses"`
	StartDate *string `json:"startDate"` // can be null
	EndDate   *string `json:"endDate"`   // can be null
	Format    string  `json:"format"`
}

var jsTimeFmt = "2006-01-02T15:04:05Z07:00"

func GetTaxableEventsCSV(c *gin.Context) {
	var requestBody TaxableEventsCSVRequest
	err := c.BindJSON(&requestBody)
	if err != nil {
		// the error returned here has already been pushed to the context... I think.
		c.AbortWithError(500, errors.New("Error processing request body")) // nolint:staticcheck,errcheck
		return
	}

	// We expect ISO 8601 dates in UTC
	var startDate *time.Time
	if requestBody.StartDate != nil {
		startTime, err := time.Parse(jsTimeFmt, *requestBody.StartDate)
		if err != nil {
			c.AbortWithError(500, fmt.Errorf("invalid start time. Err %v", err)) // nolint:errcheck
		}
		startDate = &startTime
	}

	var endDate *time.Time
	if requestBody.EndDate != nil {
		endTime, err := time.Parse(jsTimeFmt, *requestBody.EndDate)
		if err != nil {
			c.AbortWithError(500, fmt.Errorf("invalid end time. Err %v", err)) // nolint:errcheck
		}
		endDate = &endTime
	}
	config.Log.Infof("Start: %s End: %s\n", startDate, endDate)

	if requestBody.Addresses == "" {
		c.JSON(422, gin.H{"message": "Address is required"})
		return
	}

	// parse addresses
	var addresses []string
	// strip spaces
	requestBody.Addresses = strings.ReplaceAll(requestBody.Addresses, " ", "")
	// split on commas
	addresses = strings.Split(requestBody.Addresses, ",")

	if requestBody.Format == "" {
		c.JSON(422, gin.H{"message": "Format is required"})
		return
	}

	accountRows, headers, err := csv.ParseForAddress(addresses, startDate, endDate, DB, requestBody.Format, *GlobalCfg)
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
