package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
	"github.com/DefiantLabs/cosmos-tax-cli-private/core"
	"github.com/go-co-op/gocron"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gorm.io/gorm"

	dbTypes "github.com/DefiantLabs/cosmos-tax-cli-private/db"
)

var (
	cfgFile string        // config file location to load
	conf    config.Config // stores the unmarshaled config loaded from Viper, available to all commands in the cmd package
	rootCmd = &cobra.Command{
		Use: "cosmos-tax-cli-private",
		// TODO: Get user-friendly descriptions approved
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
	// initConfig on initialize of cobra guarantees config struct will be set before all subcommands are executed
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cosmos-tax-cli-private/config.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		viper.SetConfigType("toml")
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Failed to find user home dir. Err: %v", err)
		}
		defaultCfgLocation := fmt.Sprintf("%s/.cosmos-tax-cli-private", home)

		viper.AddConfigPath(defaultCfgLocation)
		viper.SetConfigType("toml")
		viper.SetConfigName("config")
	}

	// Load defaults into a file at $HOME?
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Failed to read config file. Err: %v", err)
	}

	// Unmarshal the config into struct
	err = viper.Unmarshal(&conf)
	if err != nil {
		log.Fatalf("Failed to unmarshal config. Err: %v", err)
	}

	// TODO: Consider creating a set of defaults

	// Validate config
	err = conf.Validate()
	if err != nil {
		log.Fatalf("Failed to validate config. Err: %v", err)
	}
}

// TODO: Refactor all of this code. Move to config folder, make it work for multiple chains.
// Separate the DB logic, scheduler logic, and blockchain logic into different functions.
//
// setup does pre-run setup configurations.
//   - Loads the application config from config.tml, cli args and parses/merges
//   - Connects to the database and returns the db object
//   - Returns various values used throughout the application
func setup(cfg config.Config) (*config.Config, *gorm.DB, *gocron.Scheduler, error) {
	// Logger
	logLevel := cfg.Log.Level
	logPath := cfg.Log.Path
	config.DoConfigureLogger(logPath, logLevel)

	// 0 is an invalid starting block, set it to 1
	if cfg.Base.StartBlock == 0 {
		cfg.Base.StartBlock = 1
	}

	db, err := dbTypes.PostgresDbConnect(cfg.Database.Host, cfg.Database.Port, cfg.Database.Database,
		cfg.Database.User, cfg.Database.Password, cfg.Log.Level)
	if err != nil {
		config.Log.Fatal("Could not establish connection to the database", zap.Error(err))
	}

	sqldb, _ := db.DB()
	sqldb.SetMaxIdleConns(10)
	sqldb.SetMaxOpenConns(100)
	sqldb.SetConnMaxLifetime(time.Hour)

	// TODO: make mapping for all chains, globally initialized
	core.SetupAddressRegex(cfg.Base.AddressRegex)   // e.g. "juno(valoper)?1[a-z0-9]{38}"
	core.SetupAddressPrefix(cfg.Base.AddressPrefix) // e.g. juno

	scheduler := gocron.NewScheduler(time.UTC)

	// run database migrations at every runtime
	err = dbTypes.MigrateModels(db)
	if err != nil {
		config.Log.Error("Error running DB migrations", zap.Error(err))
		return nil, nil, nil, err
	}

	// We should stop relying on the denom cache now that we are running this as a CLI tool only
	dbTypes.CacheDenoms(db)

	return &cfg, db, scheduler, nil
}
