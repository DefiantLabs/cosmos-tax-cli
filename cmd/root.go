package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
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
		Use:   "cosmos-tax-cli-private",
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

	// Log
	rootCmd.PersistentFlags().StringVar(&conf.Log.Level, "log.level", "info", "log level")
	rootCmd.PersistentFlags().BoolVar(&conf.Log.Pretty, "log.pretty", false, "pretty logs")
	rootCmd.PersistentFlags().StringVar(&conf.Log.Path, "log.path", "", "log path (default is $HOME/.cosmos-tax-cli-private/logs.txt")

	// Base
	// chain indexing
	rootCmd.PersistentFlags().BoolVar(&conf.Base.ChainIndexingEnabled, "base.indexChain", true, "enable chain indexing?")
	rootCmd.PersistentFlags().Int64Var(&conf.Base.StartBlock, "base.startBlock", 0, "block to start indexing at (use -1 to resume from highest block indexed)")
	rootCmd.PersistentFlags().Int64Var(&conf.Base.EndBlock, "base.endBlock", -1, "block to stop indexing at (use -1 to index indefinitely")
	rootCmd.PersistentFlags().BoolVar(&conf.Base.PreventReattempts, "base.preventReattempts", false, "prevent reattempts of failed blocks.")
	// reward indexing
	rootCmd.PersistentFlags().BoolVar(&conf.Base.RewardIndexingEnabled, "base.indexRewards", true, "enable osmosis reward indexing?")
	rootCmd.PersistentFlags().Int64Var(&conf.Base.RewardStartBlock, "base.rewardStartBlock", 0, "block to start indexing rewards at")
	rootCmd.PersistentFlags().Int64Var(&conf.Base.RewardEndBlock, "base.rewardEndBlock", 0, "block to stop indexing rewards at (use -1 to index indefinitely")
	// other base setting
	rootCmd.PersistentFlags().BoolVar(&conf.Base.Dry, "base.dry", false, "index the chain but don't insert data in the DB.")
	rootCmd.PersistentFlags().StringVar(&conf.Base.API, "base.api", "", "node api endpoint")
	rootCmd.PersistentFlags().Float64Var(&conf.Base.Throttling, "base.throttling", 0.5, "throttle delay")
	rootCmd.PersistentFlags().Int64Var(&conf.Base.RPCWorkers, "base.rpcworkers", 1, "rpc workers")
	rootCmd.PersistentFlags().BoolVar(&conf.Base.WaitForChain, "base.waitforchain", false, "wait for chain to be in sync?")

	// Lens
	rootCmd.PersistentFlags().StringVar(&conf.Lens.RPC, "lens.rpc", "", "node rpc endpoint")
	rootCmd.PersistentFlags().StringVar(&conf.Lens.AccountPrefix, "lens.accountPrefix", "", "lens account prefix")
	rootCmd.PersistentFlags().StringVar(&conf.Lens.ChainID, "lens.chainID", "", "lens chain ID")
	rootCmd.PersistentFlags().StringVar(&conf.Lens.ChainName, "lens.chainName", "", "lens chain name")

	// Database
	rootCmd.PersistentFlags().StringVar(&conf.Database.Host, "db.host", "", "database host")
	rootCmd.PersistentFlags().StringVar(&conf.Database.Port, "db.port", "5432", "database port")
	rootCmd.PersistentFlags().StringVar(&conf.Database.Database, "db.database", "", "database name")
	rootCmd.PersistentFlags().StringVar(&conf.Database.User, "db.user", "", "database user")
	rootCmd.PersistentFlags().StringVar(&conf.Database.Password, "db.password", "", "database password")
	rootCmd.PersistentFlags().StringVar(&conf.Database.LogLevel, "db.loglevel", "", "database loglevel")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		viper.SetConfigType("toml")
	} else {
		// Check in current working dir
		pwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("Could not determine current working dir. Err: %v", err)
		}
		if _, err := os.Stat(fmt.Sprintf("%v/config.toml", pwd)); err == nil {
			cfgFile = pwd
		} else {
			// file not in current working dir. Check home dir instead
			// Find home directory.
			home, err := os.UserHomeDir()
			if err != nil {
				log.Fatalf("Failed to find user home dir. Err: %v", err)
			}
			cfgFile = fmt.Sprintf("%s/.cosmos-tax-cli-private", home)
		}
		viper.AddConfigPath(cfgFile)
		viper.SetConfigType("toml")
		viper.SetConfigName("config")
	}

	// Load defaults into a file at $HOME?
	var noConfig bool
	err := viper.ReadInConfig()
	if err != nil {
		if strings.Contains(err.Error(), "Config File \"config\" Not Found") {
			noConfig = true
		} else {
			log.Fatalf("Failed to read config file. Err: %v", err)
		}
	}

	if !noConfig {
		log.Println("CFG successfully read from: ", cfgFile)
		// Unmarshal the config into struct
		err = viper.Unmarshal(&conf)
		if err != nil {
			log.Fatalf("Failed to unmarshal config. Err: %v", err)
		}
	}

	// Validate config
	err = conf.Validate()
	if err != nil {
		log.Fatalf("Failed to validate config. Err: %v", err)
	}
}

// Separate the DB logic, scheduler logic, and blockchain logic into different functions.
//
// setup does pre-run setup configurations.
//   - Loads the application config from config.tml, cli args and parses/merges
//   - Connects to the database and returns the db object
//   - Returns various values used throughout the application
func setup(cfg config.Config) (*config.Config, bool, *gorm.DB, *gocron.Scheduler, error) {
	// Logger
	logLevel := cfg.Log.Level
	logPath := cfg.Log.Path
	prettyLogging := cfg.Log.Pretty
	config.DoConfigureLogger(logPath, logLevel, prettyLogging)

	// 0 is an invalid starting block, set it to 1
	if cfg.Base.StartBlock == 0 {
		cfg.Base.StartBlock = 1
	}

	db, err := dbTypes.PostgresDbConnect(cfg.Database.Host, cfg.Database.Port, cfg.Database.Database,
		cfg.Database.User, cfg.Database.Password, strings.ToLower(cfg.Database.LogLevel))
	if err != nil {
		config.Log.Fatal("Could not establish connection to the database", err)
	}

	sqldb, _ := db.DB()
	sqldb.SetMaxIdleConns(10)
	sqldb.SetMaxOpenConns(100)
	sqldb.SetConnMaxLifetime(time.Hour)

	scheduler := gocron.NewScheduler(time.UTC)

	// run database migrations at every runtime
	err = dbTypes.MigrateModels(db)
	if err != nil {
		config.Log.Error("Error running DB migrations", err)
		return nil, false, nil, nil, err
	}

	// We should stop relying on the denom cache now that we are running this as a CLI tool only
	dbTypes.CacheDenoms(db)

	return &cfg, cfg.Base.Dry, db, scheduler, nil
}
