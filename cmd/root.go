package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/go-co-op/gocron"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gorm.io/gorm"

	dbTypes "github.com/DefiantLabs/cosmos-tax-cli/db"
)

var (
	cfgFile string        // config file location to load
	conf    config.Config // stores the unmarshaled config loaded from Viper, available to all commands in the cmd package
	rootCmd = &cobra.Command{
		Use:   "cosmos-tax-cli",
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cosmos-tax-cli/config.yaml)")

	// Log
	rootCmd.PersistentFlags().StringVar(&conf.Log.Level, "log.level", "info", "log level")
	rootCmd.PersistentFlags().BoolVar(&conf.Log.Pretty, "log.pretty", false, "pretty logs")
	rootCmd.PersistentFlags().StringVar(&conf.Log.Path, "log.path", "", "log path (default is $HOME/.cosmos-tax-cli/logs.txt")

	// Base
	// chain indexing
	rootCmd.PersistentFlags().BoolVar(&conf.Base.ChainIndexingEnabled, "base.index-chain", true, "enable chain indexing?")
	rootCmd.PersistentFlags().Int64Var(&conf.Base.StartBlock, "base.start-block", 0, "block to start indexing at (use -1 to resume from highest block indexed)")
	rootCmd.PersistentFlags().Int64Var(&conf.Base.EndBlock, "base.end-block", -1, "block to stop indexing at (use -1 to index indefinitely")
	rootCmd.PersistentFlags().BoolVar(&conf.Base.ReIndex, "base.reindex", false, "if true, this will re-attempt to index blocks we have already indexed (defaults to false)")
	rootCmd.PersistentFlags().BoolVar(&conf.Base.PreventReattempts, "base.prevent-reattempts", false, "prevent reattempts of failed blocks.")
	// block event indexing
	rootCmd.PersistentFlags().BoolVar(&conf.Base.BlockEventIndexingEnabled, "base.index-block-events", true, "enable block beginblocker and endblocker event indexing?")
	rootCmd.PersistentFlags().Int64Var(&conf.Base.BlockEventsStartBlock, "base.block-events-start-block", 0, "block to start indexing block events at")
	rootCmd.PersistentFlags().Int64Var(&conf.Base.BlockEventsEndBlock, "base.block-events-end-block", 0, "block to stop indexing block events at (use -1 to index indefinitely")
	// epoch event indexing
	rootCmd.PersistentFlags().BoolVar(&conf.Base.EpochEventIndexingEnabled, "base.index-epoch-events", true, "enable epoch beginblocker and endblocker event indexing?")
	rootCmd.PersistentFlags().Int64Var(&conf.Base.EpochEventsStartEpoch, "base.epoch-events-start-epoch", 0, "epoch number to start indexing block events at")
	rootCmd.PersistentFlags().Int64Var(&conf.Base.EpochEventsEndEpoch, "base.epoch-events-end-epoch", 0, "epoch number to stop indexing block events at")
	rootCmd.PersistentFlags().StringVar(&conf.Base.EpochIndexingIdentifier, "base.epoch-indexing-identifier", "", "epoch identifier to index")
	// other base setting
	rootCmd.PersistentFlags().BoolVar(&conf.Base.Dry, "base.dry", false, "index the chain but don't insert data in the DB.")
	rootCmd.PersistentFlags().StringVar(&conf.Base.API, "base.api", "", "node api endpoint")
	rootCmd.PersistentFlags().Float64Var(&conf.Base.Throttling, "base.throttling", 0.5, "throttle delay")
	rootCmd.PersistentFlags().Int64Var(&conf.Base.RPCWorkers, "base.rpc-workers", 1, "rpc workers")
	rootCmd.PersistentFlags().BoolVar(&conf.Base.WaitForChain, "base.wait-for-chain", false, "wait for chain to be in sync?")
	rootCmd.PersistentFlags().Int64Var(&conf.Base.WaitForChainDelay, "base.wait-for-chain-delay", 10, "seconds to wait between each check for node to catch up to the chain")
	rootCmd.PersistentFlags().Int64Var(&conf.Base.BlockTimer, "base.block-timer", 10000, "print out how long it takes to process this many blocks")
	rootCmd.PersistentFlags().BoolVar(&conf.Base.ExitWhenCaughtUp, "base.exit-when-caught-up", true, "mainly used for Osmosis rewards indexing")

	// Lens
	rootCmd.PersistentFlags().StringVar(&conf.Lens.RPC, "lens.rpc", "", "node rpc endpoint")
	rootCmd.PersistentFlags().StringVar(&conf.Lens.AccountPrefix, "lens.account-prefix", "", "lens account prefix")
	rootCmd.PersistentFlags().StringVar(&conf.Lens.ChainID, "lens.chain-id", "", "lens chain ID")
	rootCmd.PersistentFlags().StringVar(&conf.Lens.ChainName, "lens.chain-name", "", "lens chain name")

	// Database
	rootCmd.PersistentFlags().StringVar(&conf.Database.Host, "database.host", "", "database host")
	rootCmd.PersistentFlags().StringVar(&conf.Database.Port, "database.port", "5432", "database port")
	rootCmd.PersistentFlags().StringVar(&conf.Database.Database, "database.database", "", "database name")
	rootCmd.PersistentFlags().StringVar(&conf.Database.User, "database.user", "", "database user")
	rootCmd.PersistentFlags().StringVar(&conf.Database.Password, "database.password", "", "database password")
	rootCmd.PersistentFlags().StringVar(&conf.Database.LogLevel, "database.log-level", "", "database loglevel")
}

func initConfig() {
	v := viper.New()
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
		v.SetConfigType("toml")
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
			cfgFile = fmt.Sprintf("%s/.cosmos-tax-cli", home)
		}
		v.AddConfigPath(cfgFile)
		v.SetConfigType("toml")
		v.SetConfigName("config")
	}

	// Load defaults into a file at $HOME?
	var noConfig bool
	err := v.ReadInConfig()
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "Config File \"config\" Not Found"):
			noConfig = true
		case strings.Contains(err.Error(), "incomplete number"):
			log.Fatalf("Failed to read config file %v. This usually means you forgot to wrap a string in quotes.", err)
		default:
			log.Fatalf("Failed to read config file. Err: %v", err)
		}
	}

	ignoredKeys := config.CheckSuperfluousConfigKeys(v.AllKeys())
	if len(ignoredKeys) > 0 {
		config.Log.Warnf("Warning, the following invalid keys will be ignored: %v", ignoredKeys)
	}

	if !noConfig {
		log.Println("CFG successfully read from: ", cfgFile)
		// Merge config file settings with the command line options.
		bindFlags(rootCmd, v)
	}

	// Validate config
	err = conf.Validate()
	if err != nil {
		log.Fatalf("Failed to validate config. Err: %v", err)
	}
}

// Set config vars from cpnfig file not already specified on command line.
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		configName := f.Name

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(configName) {
			val := v.Get(configName)
			err := cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
			if err != nil {
				log.Fatalf("Failed to bind config file value %v. Err: %v", configName, err)
			}
		}
	})
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
	dbTypes.CacheIBCDenoms(db)

	return &cfg, cfg.Base.Dry, db, scheduler, nil
}
