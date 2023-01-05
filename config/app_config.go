package config

import (
	"errors"
	lg "log"

	"github.com/BurntSushi/toml"
	"github.com/DefiantLabs/cosmos-tax-cli-private/util"
	"github.com/imdario/mergo"
)

type Config struct {
	Database           database
	API                api // deprecated in favor of lens.Rpc (at least in this app)
	ConfigFileLocation string
	Base               base
	Log                log
	Lens               lens
}

// Validate will validate the config for required fields
func (conf *Config) Validate() error {
	// Database Checks
	if util.StrNotSet(conf.Database.Host) {
		return errors.New("database host must be set")
	}
	if util.StrNotSet(conf.Database.Port) {
		return errors.New("database port must be set")
	}
	if util.StrNotSet(conf.Database.Database) {
		return errors.New("database name (i.e. Database) must be set")
	}
	if util.StrNotSet(conf.Database.User) {
		return errors.New("database user must be set")
	}
	if util.StrNotSet(conf.Database.Password) {
		return errors.New("database password must be set")
	}

	// Base Checks
	if conf.Base.StartBlock == 0 { // TODO: Verify that 0 is not a valid starting block..
		return errors.New("base startblock must be set")
	}
	if conf.Base.EndBlock == 0 {
		return errors.New("base endblock must be set")
	}
	// If rewards indexes are not valid, error
	if conf.Base.RewardStartBlock < 0 {
		return errors.New("rewards startblock must be valid")
	}
	if conf.Base.RewardEndBlock < -1 {
		return errors.New("rewards endblock must be valid")
	}
	// If rewards indexs are not set, use base start/end
	if conf.Base.RewardStartBlock == 0 {
		conf.Base.RewardStartBlock = conf.Base.StartBlock
	}
	if conf.Base.RewardEndBlock == 0 {
		conf.Base.RewardEndBlock = conf.Base.EndBlock
	}
	// Throttling can safely default to 0
	// BlockTimer can safely default to 0
	// WaitForChain can safely default to false
	// WaitForChainDelay can safely default to 0
	// IndexingEnabled can safely default to false TODO: but do we want this?
	// ExitWhenCaughtUp can safely default to false
	// OsmosisRewardsOnly can safely default to false

	// Log
	// Both level and path can safely be blank

	// Lens
	if util.StrNotSet(conf.Lens.RPC) {
		return errors.New("lens rpc must be set")
	}
	if util.StrNotSet(conf.Lens.AccountPrefix) {
		return errors.New("lens accountPrefix must be set")
	}
	if util.StrNotSet(conf.Lens.ChainID) {
		return errors.New("lens chainID must be set")
	}
	if util.StrNotSet(conf.Lens.ChainName) {
		return errors.New("lens chainName must be set")
	}

	return nil
}

// ValidateClientConfig will validate the config for fields required by the client
func (conf *Config) ValidateClientConfig() error {
	// Database Checks
	if util.StrNotSet(conf.Database.Host) {
		return errors.New("database host must be set")
	}
	if util.StrNotSet(conf.Database.Port) {
		return errors.New("database port must be set")
	}
	if util.StrNotSet(conf.Database.Database) {
		return errors.New("database name (i.e. Database) must be set")
	}
	if util.StrNotSet(conf.Database.User) {
		return errors.New("database user must be set")
	}
	if util.StrNotSet(conf.Database.Password) {
		return errors.New("database password must be set")
	}

	return nil
}

type database struct {
	Host     string
	Port     string
	Database string
	User     string
	Password string
	LogLevel string
}

type lens struct {
	RPC           string
	AccountPrefix string
	ChainID       string
	ChainName     string
}

type api struct {
	Host string
}

type base struct {
	API                   string
	StartBlock            int64
	EndBlock              int64
	Throttling            float64
	RPCWorkers            int64
	BlockTimer            int64
	WaitForChain          bool
	WaitForChainDelay     int64
	IndexingEnabled       bool
	ExitWhenCaughtUp      bool
	RewardIndexingEnabled bool
	Dry                   bool
	RewardStartBlock      int64
	RewardEndBlock        int64
	CreateCSVFile         bool
	CSVFile               string
}

type log struct {
	Level  string
	Path   string
	Pretty bool
}

func GetConfig(configFileLocation string) (Config, error) {
	var conf Config
	_, err := toml.DecodeFile(configFileLocation, &conf)
	return conf, err
}

func MergeConfigs(def Config, overide Config) Config {
	err := mergo.Merge(&overide, def)
	if err != nil {
		lg.Panicf("Config merge failed. Err: %v", err)
	}

	return overide
}
