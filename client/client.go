package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli/chainregistry"
	"github.com/DefiantLabs/cosmos-tax-cli/client/docs"
	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/csv"
	csvParsers "github.com/DefiantLabs/cosmos-tax-cli/csv/parsers"
	dbTypes "github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

var (
	DB        *gorm.DB
	ClientCfg *config.ClientConfig
)

var chainRegBlacklist = map[string]bool{
	"_IBC":        true,
	"_memo_keys":  true,
	"_non-cosmos": true,
	"_template":   true,
	".github":     true,
	".git":        true,
	"testnets":    true,
	"thorchain":   true,
	"xion":        true,
}

func loadChainRegistryAssetLists(cfg *config.ClientConfig) (map[string]chainregistry.Asset, error) {
	err := chainregistry.UpdateChainRegistryOnDisk(cfg.ChainRegistryLocation)
	if err != nil {
		return nil, err
	}

	assetMap, err := chainregistry.GetAssetMapOnDisk(cfg.ChainRegistryLocation, chainRegBlacklist)

	return assetMap, err
}

func setup() (*gorm.DB, *config.ClientConfig, int, string, error) {
	argConfig, flagSet, svcPort, err := config.ParseClientArgs(os.Stderr, os.Args[1:])
	if err != nil {
		if strings.Contains(err.Error(), "help requested") {
			config.Log.Info("Please see valid flags above.")
			os.Exit(0)
		} else if strings.Contains(err.Error(), "flag provided but not defined") {
			config.Log.Info("Invalid flag. Please see valid flags above.")
			os.Exit(0)
		}
		config.Log.Panicf("Error parsing args. Err: %v", err)
		return nil, nil, svcPort, argConfig.Client.Model, err
	}

	var location string
	if argConfig.ConfigFileLocation != "" {
		location = argConfig.ConfigFileLocation
	} else {
		location = "./config.toml"
	}

	fileConfig, err := config.GetClientConfig(location)
	if err != nil {
		if !strings.Contains(err.Error(), "no such file or directory") {
			config.Log.Panicf("Error opening configuration file. Err: %v", err)
			return nil, nil, svcPort, argConfig.Client.Model, err
		}
	}

	// merge and validate configs
	cfg := config.MergeClientConfigs(fileConfig, argConfig)
	err = cfg.ValidateClientConfig()
	if err != nil {
		flagSet.PrintDefaults()
		config.Log.Fatalf("Config validation failed. Err: %v", err)
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
		return nil, nil, svcPort, cfg.Client.Model, err
	}

	dbTypes.CacheDenoms(db)
	dbTypes.CacheIBCDenoms(db)

	err = db.AutoMigrate(&dbTypes.Address{}, &AddressUsageCSV{}, &AddressUsageJSON{})
	if err != nil {
		config.Log.Fatalf("Error migrating client models. Err: %v", err)
	}

	return db, &cfg, svcPort, cfg.Client.Model, nil
}

// @title Cosmos Tax CLI
// @version         1.0
// @description     An API to interact with the Cosmos Tax CLI backend.
// @contact.name   Defiant Labs
// @contact.url    https://defiantlabs.net/
// @contact.email  info@defiantlabs.net
// @BasePath  /
// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
func main() {
	db, cfg, svcPort, model, err := setup()
	if err != nil {
		log.Fatalf("Error setting up. Err: %v", err)
	}

	DB = db
	ClientCfg = cfg

	assetMap, err := loadChainRegistryAssetLists(ClientCfg)
	if err != nil {
		config.Log.Fatalf("Error loading chain registry. Err: %v", err)
	}

	chainregistry.CacheAssetMap(assetMap)

	// Have to keep this here so that import of docs subfolder (which contains proper init()) stays
	docs.SwaggerInfo.Title = "Cosmos Tax CLI"

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(ZeroLogMiddleware())

	r.Use(CORSMiddleware())

	r.GET("/gcphealth", Healthcheck)

	if model == "commercial" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
		r.POST("/events.json", GetTaxableEventsJSON)
	}

	r.POST("/events.csv", GetTaxableEventsCSV)
	err = r.Run(fmt.Sprintf(":%v", svcPort))
	if err != nil {
		config.Log.Fatal("Error starting server.", err)
	}
}

// @Router /gcphealth [get]
func Healthcheck(context *gin.Context) {
	context.JSON(200, gin.H{"status": "ok"})
}

func GetClientIP(c *gin.Context) string {
	// first check the X-Forwarded-For header
	requester := c.Request.Header.Get("X-Forwarded-For")
	// if empty, check the Real-IP header
	if len(requester) == 0 {
		requester = c.Request.Header.Get("X-Real-IP")
	}
	// if the requester is still empty, use the hard-coded address from the socket
	if len(requester) == 0 {
		requester = c.Request.RemoteAddr
	}

	// if requester is a comma delimited list, take the first one
	// (this happens when proxied via elastic load balancer then again through nginx)
	if strings.Contains(requester, ",") {
		requester = strings.Split(requester, ",")[0]
	}

	return requester
}

// ZeroLogMiddleware sends gin logs to our zerologger
func ZeroLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process Request
		c.Next()

		// Stop timer
		duration := fmt.Sprint(time.Since(start).Milliseconds())

		// create and send log event
		event := config.Log.ZInfo().
			Str("client_ip", GetClientIP(c)).
			Str("duration", duration).
			Str("method", c.Request.Method).
			Str("path", c.Request.RequestURI).
			Str("status", fmt.Sprint(c.Writer.Status())).
			Str("referrer", c.Request.Referer())

		if c.Writer.Status() >= 500 {
			event.Err(c.Errors.Last())
		}

		event.Send()
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

// @Accept json
// @Produce text/csv
// @Param data body TaxableEventsCSVRequest true "The options for the POST body"
// @Router /events.csv [post]
func GetTaxableEventsCSV(c *gin.Context) {
	addresses, format, startDate, endDate, err := ParseTaxableEventsBody(c)
	if err != nil {
		return
	}

	parserKeys := csvParsers.GetParserKeys()
	formatFound := false
	for _, b := range parserKeys {
		if format == b {
			formatFound = true
			break
		}
	}

	if !formatFound {
		c.JSON(422, gin.H{"message": fmt.Sprintf("Unsupported format %s, supported values are %s", format, parserKeys)})
		return
	}

	// We only want to process and return data on addresses we know are valid (in our DB)
	validAddresses, validAddressDBObjects, err := GetValidAddresses(addresses)
	if err != nil {
		config.Log.Errorf("Error getting valid addresses: %v", addresses)
		c.AbortWithError(500, errors.New("error getting rows for address")) // nolint:staticcheck,errcheck
		return
	}

	accountRows, headers, addressRowsCount, err := csv.ParseForAddress(validAddresses, startDate, endDate, DB, format)
	if err != nil {
		// the error returned here has already been pushed to the context... I think.
		config.Log.Errorf("Error getting valid addresses %v: %s", addresses, err)
		fmt.Println(err)
		c.AbortWithError(500, errors.New("error getting rows for address")) // nolint:staticcheck,errcheck
		return
	}

	for _, address := range validAddressDBObjects {
		usage := AddressUsageCSV{
			Address:       address,
			AddressID:     address.ID,
			RowsRetrieved: addressRowsCount[address.Address],
			Timestamp:     time.Now(),
		}

		err = DB.Create(&usage).Error
		if err != nil {
			config.Log.Errorf("Error saving address usage: %v", err)
		}
	}

	if len(accountRows) == 0 {
		c.JSON(404, gin.H{"message": "No transactions for given address"})
		return
	}

	buffer, err := csv.ToCsv(accountRows, headers)
	if err != nil {
		config.Log.Error("Error generating CSV", err)
		c.AbortWithError(500, errors.New("error getting rows for address")) // nolint:staticcheck,errcheck
		return
	}

	c.Data(200, "text/csv", buffer.Bytes())
}

// @Accept json
// @Produce json
// @Param data body TaxableEventsCSVRequest true "The options for the POST body"
// @Router /events.json [post]
func GetTaxableEventsJSON(c *gin.Context) {
	addresses, format, startDate, endDate, err := ParseTaxableEventsBody(c)
	if err != nil {
		return
	}

	parserKeys := csvParsers.GetParserKeys()
	formatFound := false
	for _, b := range parserKeys {
		if format == b {
			formatFound = true
			break
		}
	}

	if !formatFound {
		c.JSON(422, gin.H{"message": fmt.Sprintf("Unsupported format \"%s\", supported values are %s", format, parserKeys)})
		return
	}

	// We only want to process and return data on addresses we know are valid (in our DB)
	validAddresses, validAddressDBObjects, err := GetValidAddresses(addresses)
	if err != nil {
		config.Log.Errorf("Error getting valid addresses %v: %s", addresses, err)
		c.AbortWithError(500, errors.New("error getting rows for address")) // nolint:staticcheck,errcheck
		return
	}

	accountRows, _, addressRowsCount, err := csv.ParseForAddress(validAddresses, startDate, endDate, DB, format)
	if err != nil {
		// the error returned here has already been pushed to the context... I think.
		config.Log.Errorf("Error getting rows for addresses: %v", validAddresses)
		c.AbortWithError(500, errors.New("error getting rows for address")) // nolint:staticcheck,errcheck
		return
	}

	for _, address := range validAddressDBObjects {
		usage := AddressUsageJSON{
			Address:       address,
			AddressID:     address.ID,
			RowsRetrieved: addressRowsCount[address.Address],
			Timestamp:     time.Now(),
		}

		err = DB.Create(&usage).Error
		if err != nil {
			config.Log.Errorf("Error saving address usage: %v", err)
		}
	}

	if len(accountRows) == 0 {
		c.JSON(404, gin.H{"message": "No transactions for given address"})
		return
	}

	c.JSON(200, accountRows)
}

func ParseTaxableEventsBody(c *gin.Context) ([]string, string, *time.Time, *time.Time, error) {
	var requestBody TaxableEventsCSVRequest
	err := c.BindJSON(&requestBody)
	if err != nil {
		// the error returned here has already been pushed to the context... I think.
		c.AbortWithError(500, errors.New("error processing request body")) // nolint:staticcheck,errcheck
		return nil, "", nil, nil, err
	}

	// We expect ISO 8601 dates in UTC
	var startDate *time.Time
	if requestBody.StartDate != nil {
		startTime, err := time.Parse(jsTimeFmt, *requestBody.StartDate)
		if err != nil {
			c.AbortWithError(500, fmt.Errorf("invalid start time. Err %v", err)) // nolint:errcheck
			return nil, "", nil, nil, err
		}
		startDate = &startTime
	}

	var endDate *time.Time
	if requestBody.EndDate != nil {
		endTime, err := time.Parse(jsTimeFmt, *requestBody.EndDate)
		if err != nil {
			c.AbortWithError(500, fmt.Errorf("invalid end time. Err %v", err)) // nolint:errcheck
			return nil, "", nil, nil, err
		}
		endDate = &endTime
	}
	config.Log.Infof("Start: %s End: %s\n", startDate, endDate)

	if requestBody.Addresses == "" {
		c.JSON(422, gin.H{"message": "Address is required"})
		return nil, "", nil, nil, err
	}

	// parse addresses
	var addresses []string
	// strip spaces
	requestBody.Addresses = strings.ReplaceAll(requestBody.Addresses, " ", "")
	// split on commas
	addresses = strings.Split(requestBody.Addresses, ",")

	format := requestBody.Format

	if format == "" {
		c.JSON(422, gin.H{"message": "Format is required"})
		return nil, "", nil, nil, errors.New("format is required")
	}

	return addresses, format, startDate, endDate, nil
}

func GetValidAddresses(addresses []string) ([]string, []dbTypes.Address, error) {
	validAddresses, err := dbTypes.GetAddresses(addresses, DB)
	if err != nil {
		config.Log.Errorf("Error getting addresses: %v", addresses)
		return nil, nil, err
	}

	var validAddressesStr []string
	if len(validAddresses) != len(addresses) {
		for _, address := range addresses {
			found := false
			for _, validAddress := range validAddresses {
				if address == validAddress.Address {
					found = true
					break
				}
			}
			if found {
				validAddressesStr = append(validAddressesStr, address)
			} else {
				config.Log.Infof("Invalid address not found in DB, skipping: %v", address)
			}
		}
	} else {
		validAddressesStr = addresses
	}

	return validAddressesStr, validAddresses, nil
}

type AddressUsageCSV struct {
	ID            uint
	Address       dbTypes.Address
	AddressID     uint
	RowsRetrieved uint
	Timestamp     time.Time
}

type AddressUsageJSON struct {
	ID            uint
	Address       dbTypes.Address
	AddressID     uint
	RowsRetrieved uint
	Timestamp     time.Time
}
