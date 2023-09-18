package cmd

import (
	"os"
	"strings"
	"time"

	"github.com/DefiantLabs/cosmos-indexer/config"
	dbTypes "github.com/DefiantLabs/cosmos-indexer/db"
	"github.com/DefiantLabs/cosmos-indexer/tasks"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

var updateDenomsConfig config.UpdateDenomsConfig
var updateDenomsDbConnection *gorm.DB

func init() {
	config.SetupLogFlags(&updateDenomsConfig.Log, updateDenomsCmd)
	config.SetupDatabaseFlags(&updateDenomsConfig.Database, updateDenomsCmd)
	config.SetupLensFlags(&updateDenomsConfig.Lens, updateDenomsCmd)
	config.SetupUpdateDenomsSpecificFlags(&updateDenomsConfig, updateDenomsCmd)
	rootCmd.AddCommand(updateDenomsCmd)
}

var updateDenomsCmd = &cobra.Command{
	Use:   "update-denoms",
	Short: "Reach out to various assetlist locations to update the database with vetted denom information.",
	Long: `Reaches out to various Cosmos Denom assetlist registries and updates the values found in the database.
	Cosmos developers provide assetlists in a relatively standardized format (examples found for specific chains here https://github.com/cosmos/chain-registry).
	This command will prepopulate the Cosmos Tax CLI database with values found in regsitries for the specific chains we provide support for.
	It will either use the chain-id specified in the application configuration to update the specific assetlist, or update-all if provided.
	`,
	PreRunE: setupUpdateDenoms,
	Run:     updateDenoms,
}

func setupUpdateDenoms(cmd *cobra.Command, args []string) error {
	bindFlags(cmd, viperConf)

	err := updateDenomsConfig.Validate()

	if err != nil {
		return err
	}

	// Logger
	logLevel := updateDenomsConfig.Log.Level
	logPath := updateDenomsConfig.Log.Path
	prettyLogging := updateDenomsConfig.Log.Pretty
	config.DoConfigureLogger(logPath, logLevel, prettyLogging)

	db, err := dbTypes.PostgresDbConnect(updateDenomsConfig.Database.Host, updateDenomsConfig.Database.Port, updateDenomsConfig.Database.Database,
		updateDenomsConfig.Database.User, updateDenomsConfig.Database.Password, strings.ToLower(updateDenomsConfig.Database.LogLevel))
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

	updateDenomsDbConnection = db

	return nil
}

func updateDenoms(cmd *cobra.Command, args []string) {
	cfg := updateDenomsConfig
	db := updateDenomsDbConnection

	switch {
	case cfg.Base.UpdateAll:
		config.Log.Infof("Running denom update task for all supported chains")
		for chainID, function := range tasks.ChainSpecificDenomUpsertFunctions {
			config.Log.Infof("Running denom update task for chain %s", chainID)
			function(db)
		}
	case cfg.Lens.ChainID != "":
		function, ok := tasks.ChainSpecificDenomUpsertFunctions[cfg.Lens.ChainID]
		if ok {
			config.Log.Infof("Running denom update task for chain %s found in config", cfg.Lens.ChainID)
			function(db)
			config.Log.Info("Done")
		} else {
			config.Log.Fatalf("No denom update functionality for chain-id %s", cfg.Lens.ChainID)
		}
	default:
		config.Log.Fatal("Please pass the flag --update-all or provide a chain-id in your application configuration")
	}

	err := tasks.ValidateDenoms(db)

	if err != nil {
		config.Log.Error("Error running post-validation for update-denoms")
		os.Exit(1)
	}
}
