package cmd

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/DefiantLabs/cosmos-indexer/config"
	"github.com/DefiantLabs/cosmos-indexer/csv"
	csvParsers "github.com/DefiantLabs/cosmos-indexer/csv/parsers"
	dbTypes "github.com/DefiantLabs/cosmos-indexer/db"
	"gorm.io/gorm"

	"github.com/spf13/cobra"
)

var queryConfig config.QueryConfig
var queryDbConnection *gorm.DB
var validParserKeys = csvParsers.GetParserKeys()

func init() {
	config.SetupLogFlags(&queryConfig.Log, queryCmd)
	config.SetupDatabaseFlags(&queryConfig.Database, queryCmd)
	config.SetupQuerySpecificFlags(validParserKeys, &queryConfig, queryCmd)
	rootCmd.AddCommand(queryCmd)
}

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Queries the currently indexed data.",
	Long: `Queries the indexed data according to a configuration. Mainly address based. Apply
	your address to the command and a CSV export with your data for your address will be generated.`,
	PreRunE: setupQuery,
	Run: func(cmd *cobra.Command, args []string) {

		db := queryDbConnection

		// Validate and set dates
		var startDate *time.Time
		var endDate *time.Time
		expectedLayout := "2006-01-02:15:04:05"
		if queryConfig.Base.StartDate != "" {
			parsedDate, _ := time.Parse(expectedLayout, queryConfig.Base.StartDate)
			startDate = &parsedDate
		}
		if queryConfig.Base.EndDate != "" {
			parsedDate, _ := time.Parse(expectedLayout, queryConfig.Base.EndDate)
			endDate = &parsedDate
		}

		csvRows, headers, err := csv.ParseForAddress(queryConfig.Base.Addresses, startDate, endDate, db, queryConfig.Base.Format, queryConfig)
		if err != nil {
			log.Println(queryConfig.Base.Addresses)
			config.Log.Fatal("Error calling parser for address", err)
		}

		buffer, err := csv.ToCsv(csvRows, headers)
		if err != nil {
			config.Log.Fatal("Error generating CSV", err)
		}
		fmt.Println(buffer.String())
	},
}

func setupQuery(cmd *cobra.Command, args []string) error {
	if len(validParserKeys) == 0 {
		return errors.New("error during setup, no CSV parsers found")
	}

	bindFlags(cmd, viperConf)
	err := queryConfig.Validate(validParserKeys)

	if err != nil {
		return err
	}

	// Logger
	logLevel := queryConfig.Log.Level
	logPath := queryConfig.Log.Path
	prettyLogging := queryConfig.Log.Pretty
	config.DoConfigureLogger(logPath, logLevel, prettyLogging)

	db, err := dbTypes.PostgresDbConnect(queryConfig.Database.Host, queryConfig.Database.Port, queryConfig.Database.Database,
		queryConfig.Database.User, queryConfig.Database.Password, strings.ToLower(queryConfig.Database.LogLevel))
	if err != nil {
		config.Log.Fatal("Could not establish connection to the database", err)
	}

	sqldb, _ := db.DB()
	sqldb.SetMaxIdleConns(10)
	sqldb.SetMaxOpenConns(100)
	sqldb.SetConnMaxLifetime(time.Hour)

	// run database migrations at every runtime
	err = dbTypes.MigrateModels(db)
	if err != nil {
		config.Log.Error("Error running DB migrations", err)
		return err
	}

	queryDbConnection = db

	// We should stop relying on the denom cache now that we are running this as a CLI tool only
	dbTypes.CacheDenoms(db)
	dbTypes.CacheIBCDenoms(db)

	return nil
}
