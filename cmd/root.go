package cmd

import (
	"fmt"
	"go.uber.org/zap"
	"os"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/core"
	"github.com/go-co-op/gocron"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gorm.io/gorm"

	configHelpers "github.com/DefiantLabs/cosmos-tax-cli/config"
	dbTypes "github.com/DefiantLabs/cosmos-tax-cli/db"
)

var (
	cfgFile string        //config file location to load
	conf    config.Config //stores the unmarshaled config loaded from Viper, available to all commands in the cmd package
	rootCmd = &cobra.Command{
		Use: "cosmos-tax-cli",
		//TODO: Get user-friendly descriptions approved
		Short: "A CLI tool for indexing and querying on-chain data",
		Long: `Cosmos Tax CLI is a CLI tool for indexing and querying Cosmos-based blockchains,
		with a heavy focus on taxable events.`,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	//initConfig on initialize of cobra guarantees config struct will be set before all subcommands are executed
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cosmos-tax-cli/config.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		viper.SetConfigType("toml")
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		defaultCfgLocation := fmt.Sprintf("%s/.cosmos-tax-cli", home)

		viper.AddConfigPath(defaultCfgLocation)
		viper.SetConfigType("toml")
		viper.SetConfigName("config")
	}

	//TODO: What do we do on first time app run if config file doesnt exit?
	//Load defaults into a file at $HOME?
	//Require users to run an init command?
	if err := viper.ReadInConfig(); err == nil {
		err := viper.Unmarshal(&conf)
		cobra.CheckErr(err)
		//TODO: validate the config by making sure values exist for required struct values
		//Either set required values on Viper or
		//Write a function to check explicitly
		//Consider creating a set of defaults
	} else {
		cobra.CheckErr(err)
	}
}

// TODO: Refactor all of this code. Move to config folder, make it work for multiple chains.
// Separate the DB logic, scheduler logic, and blockchain logic into different functions.
//
// setup does pre-run setup configurations.
//   - Loads the application config from config.tml, cli args and parses/merges
//   - Connects to the database and returns the db object
//   - Returns various values used throughout the application
func setup(config config.Config) (*configHelpers.Config, *gorm.DB, *gocron.Scheduler, error) {
	//Logger
	logLevel := config.Log.Level
	logPath := config.Log.Path
	configHelpers.DoConfigureLogger(logPath, logLevel)

	//0 is an invalid starting block, set it to 1
	if config.Base.StartBlock == 0 {
		config.Base.StartBlock = 1
	}

	db, err := dbTypes.PostgresDbConnect(config.Database.Host, config.Database.Port, config.Database.Database,
		config.Database.User, config.Database.Password, config.Log.Level)
	if err != nil {
		configHelpers.Log.Fatal("Could not establish connection to the database", zap.Error(err))
	}

	sqldb, _ := db.DB()
	sqldb.SetMaxIdleConns(10)
	sqldb.SetMaxOpenConns(100)
	sqldb.SetConnMaxLifetime(time.Hour)

	//TODO: make mapping for all chains, globally initialized
	core.SetupAddressRegex(config.Base.AddressRegex)   //e.g. "juno(valoper)?1[a-z0-9]{38}"
	core.SetupAddressPrefix(config.Base.AddressPrefix) //e.g. juno

	scheduler := gocron.NewScheduler(time.UTC)

	//run database migrations at every runtime
	err = dbTypes.MigrateModels(db)
	if err != nil {
		configHelpers.Log.Error("Error running DB migrations", zap.Error(err))
		return nil, nil, nil, err
	}

	//We should stop relying on the denom cache now that we are running this as a CLI tool only
	dbTypes.CacheDenoms(db)

	return &config, db, scheduler, nil
}
