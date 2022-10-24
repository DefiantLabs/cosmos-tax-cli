package config

import (
	"github.com/BurntSushi/toml"
	"github.com/imdario/mergo"
	lg "log"
)

type Config struct {
	Database           database
	Api                api //deprecated in favor of lens.Rpc (at least in this app)
	ConfigFileLocation string
	Base               base
	Log                log
	Lens               lens
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
	Rpc            string
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
