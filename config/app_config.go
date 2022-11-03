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
	API                api //deprecated in favor of lens.Rpc (at least in this app)
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
	if util.StrNotSet(conf.Base.AddressRegex) {
		return errors.New("base addressRegex must be set")
	}
	if util.StrNotSet(conf.Base.AddressPrefix) {
		return errors.New("base addressPrefix must be set")
	}
	if conf.Base.StartBlock == 0 { //TODO: Verify that 0 is not a valid starting block..
		return errors.New("base startblock must be set")
	}
	if conf.Base.EndBlock == 0 {
		return errors.New("base endblock must be set")
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
	if util.StrNotSet(conf.Lens.Key) {
		return errors.New("lens key must be set")
	}
	if util.StrNotSet(conf.Lens.AccountPrefix) {
		return errors.New("lens accountPrefix must be set")
	}
	if util.StrNotSet(conf.Lens.KeyringBackend) {
		return errors.New("lens keyringBackend must be set")
	}
	if util.StrNotSet(conf.Lens.ChainID) {
		return errors.New("lens chainID must be set")
	}
	if util.StrNotSet(conf.Lens.ChainName) {
		return errors.New("lens chainName must be set")
	}

	return nil
}

type database struct {
	Host     string
	Port     string
	Database string
	User     string
	Password string
}

type lens struct {
	Homepath       string
	RPC            string
	Key            string
	AccountPrefix  string
	KeyringBackend string
	ChainID        string
	ChainName      string
}

type api struct {
	Host string
}

type base struct {
	AddressRegex       string
	AddressPrefix      string
	StartBlock         int64
	EndBlock           int64
	Throttling         int64
	BlockTimer         int64
	WaitForChain       bool
	WaitForChainDelay  int64
	IndexingEnabled    bool
	ExitWhenCaughtUp   bool
	OsmosisRewardsOnly bool
}

type log struct {
	Level string
	Path  string
}

func GetConfig(configFileLocation string) (conf Config, err error) {
	_, err = toml.DecodeFile(configFileLocation, &conf)
	return
}

func MergeConfigs(def Config, overide Config) Config {
	err := mergo.Merge(&overide, def)
	if err != nil {
		lg.Panicf("Config merge failed. Err: %v", err)
	}

	return overide
}
