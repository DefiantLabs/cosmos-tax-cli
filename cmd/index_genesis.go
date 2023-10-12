package cmd

import (
	"strings"
	"time"

	"github.com/DefiantLabs/cosmos-indexer/config"
	dbTypes "github.com/DefiantLabs/cosmos-indexer/db"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

var (
	indexGenesisConfig       config.IndexGenesisConfig
	indexGenesisDbConnection *gorm.DB
)

func init() {
	config.SetupLogFlags(&indexGenesisConfig.Log, indexGenesisCmd)
	config.SetupDatabaseFlags(&indexGenesisConfig.Database, indexGenesisCmd)
	config.SetupLensFlags(&indexGenesisConfig.Lens, indexGenesisCmd)
	config.SetupIndexGenesisSpecificFlags(&indexGenesisConfig, indexGenesisCmd)
	rootCmd.AddCommand(indexGenesisCmd)
}

var indexGenesisCmd = &cobra.Command{
	Use:   "index-genesis",
	Short: "Indexes the blockchain genesis from either the RPC node or a downloaded genesis file.",
	Long: `Indexes the Cosmos-based blockchain according to the configurations found on the command line
	or in the specified config file. Indexes taxable events into a database for easy querying. It is
	highly recommended to keep this command running as a background service to keep your index up to date.`,
	PreRunE: setupGenesisIndex,
	Run:     indexGenesis,
}

func setupGenesisIndex(cmd *cobra.Command, args []string) error {
	bindFlags(cmd, viperConf)

	err := indexGenesisConfig.Validate()
	if err != nil {
		return err
	}

	// Logger
	logLevel := indexGenesisConfig.Log.Level
	logPath := indexGenesisConfig.Log.Path
	prettyLogging := indexGenesisConfig.Log.Pretty
	config.DoConfigureLogger(logPath, logLevel, prettyLogging)

	db, err := dbTypes.PostgresDbConnect(indexGenesisConfig.Database.Host, indexGenesisConfig.Database.Port, indexGenesisConfig.Database.Database,
		indexGenesisConfig.Database.User, indexGenesisConfig.Database.Password, strings.ToLower(indexGenesisConfig.Database.LogLevel))
	if err != nil {
		config.Log.Fatal("Could not establish connection to the database", err)
	}

	sqldb, _ := db.DB()
	sqldb.SetMaxIdleConns(10)
	sqldb.SetMaxOpenConns(100)
	sqldb.SetConnMaxLifetime(time.Hour)

	err = dbTypes.MigrateModels(db)
	if err != nil {
		config.Log.Error("Error running DB migrations", err)
		return err
	}

	indexGenesisDbConnection = db

	return nil
}

func indexGenesis(cmd *cobra.Command, args []string) {
}
