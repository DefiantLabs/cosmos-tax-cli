package config

import (
	"errors"
	"fmt"
	lg "log"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/DefiantLabs/cosmos-indexer/util"
	"github.com/imdario/mergo"
)

type Config struct {
	Database           database
	API                api // deprecated in favor of probe.Rpc (at least in this app)
	ConfigFileLocation string
	Base               base
	Log                log
	Probe              probe
	Client             client
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
		return errors.New("database name (i.e. database) must be set")
	}
	if util.StrNotSet(conf.Database.User) {
		return errors.New("database user must be set")
	}
	if util.StrNotSet(conf.Database.Password) {
		return errors.New("database password must be set")
	}

	// Base Checks
	if conf.Base.StartBlock == 0 {
		return errors.New("base start-block must be set")
	}
	if conf.Base.EndBlock == 0 {
		return errors.New("base end-block must be set")
	}
	// If block event indexes are not valid, error
	if conf.Base.BlockEventsStartBlock < 0 {
		return errors.New("block-events-start-block must be valid")
	}
	if conf.Base.BlockEventsEndBlock < -1 {
		return errors.New("block-events-end-block must be valid")
	}

	// Check if API is provided, and if so, set default ports if not set
	if conf.Base.API != "" {
		if strings.Count(conf.Base.API, ":") != 2 {
			if strings.HasPrefix(conf.Base.API, "https:") {
				conf.Base.API = fmt.Sprintf("%s:443", conf.Base.API)
			} else if strings.HasPrefix(conf.Base.API, "http:") {
				conf.Base.API = fmt.Sprintf("%s:80", conf.Base.API)
			}
		}
	}
	// Throttling can safely default to 0
	// BlockTimer can safely default to 0
	// WaitForChain can safely default to false
	// WaitForChainDelay can safely default to 0
	// ChainIndexingEnabled can safely default to false
	// ExitWhenCaughtUp can safely default to false

	// Log
	// Both level and path can safely be blank

	// Probe
	if util.StrNotSet(conf.Probe.RPC) {
		return errors.New("probe rpc must be set")
	}
	// add port if not set
	if strings.Count(conf.Probe.RPC, ":") != 2 {
		if strings.HasPrefix(conf.Probe.RPC, "https:") {
			conf.Probe.RPC = fmt.Sprintf("%s:443", conf.Probe.RPC)
		} else if strings.HasPrefix(conf.Probe.RPC, "http:") {
			conf.Probe.RPC = fmt.Sprintf("%s:80", conf.Probe.RPC)
		}
	}

	if util.StrNotSet(conf.Probe.AccountPrefix) {
		return errors.New("probe account-prefix must be set")
	}
	if util.StrNotSet(conf.Probe.ChainID) {
		return errors.New("probe chain-id must be set")
	}
	if util.StrNotSet(conf.Probe.ChainName) {
		return errors.New("probe chain-name must be set")
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
	LogLevel string `mapstructure:"log-level"`
}

type probe struct {
	RPC           string
	AccountPrefix string `mapstructure:"account-prefix"`
	ChainID       string `mapstructure:"chain-id"`
	ChainName     string `mapstructure:"chain-name"`
}

type api struct {
	Host string
}

type client struct {
	Model string
}

type base struct {
	API                       string
	StartBlock                int64 `mapstructure:"start-block"`
	EndBlock                  int64 `mapstructure:"end-block"`
	ReIndex                   bool
	PreventReattempts         bool `mapstructure:"prevent-reattempts"`
	Throttling                float64
	RPCWorkers                int64 `mapstructure:"rpc-workers"`
	BlockTimer                int64 `mapstructure:"block-timer"`
	WaitForChain              bool  `mapstructure:"wait-for-chain"`
	WaitForChainDelay         int64 `mapstructure:"wait-for-chain-delay"`
	ChainIndexingEnabled      bool  `mapstructure:"index-chain"`
	ExitWhenCaughtUp          bool  `mapstructure:"exit-when-caught-up"`
	BlockEventIndexingEnabled bool  `mapstructure:"index-block-events"`
	Dry                       bool
	BlockEventsStartBlock     int64  `mapstructure:"block-events-start-block"`
	BlockEventsEndBlock       int64  `mapstructure:"block-events-end-block"`
	EpochEventIndexingEnabled bool   `mapstructure:"index-epoch-events"`
	EpochIndexingIdentifier   string `mapstructure:"epoch-indexing-identifier"`
	EpochEventsStartEpoch     int64  `mapstructure:"epoch-events-start-epoch"`
	EpochEventsEndEpoch       int64  `mapstructure:"epoch-events-end-epoch"`
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

func CheckSuperfluousConfigKeys(keys []string) []string {
	validKeys := make(map[string]struct{})
	// add DB keys
	for _, key := range getValidConfigKeys(database{}) {
		validKeys[key] = struct{}{}
	}
	// add API keys
	for _, key := range getValidConfigKeys(api{}) {
		validKeys[key] = struct{}{}
	}
	// add base keys
	for _, key := range getValidConfigKeys(base{}) {
		validKeys[key] = struct{}{}
	}
	// add log keys
	for _, key := range getValidConfigKeys(log{}) {
		validKeys[key] = struct{}{}
	}
	// add probe keys
	for _, key := range getValidConfigKeys(probe{}) {
		validKeys[key] = struct{}{}
	}

	// Check keys
	ignoredKeys := make([]string, 0)
	for _, key := range keys {
		if _, ok := validKeys[key]; !ok {
			ignoredKeys = append(ignoredKeys, key)
		}
	}

	return ignoredKeys
}

func getValidConfigKeys(section any) (keys []string) {
	v := reflect.ValueOf(section)
	typeOfS := v.Type()

	for i := 0; i < v.NumField(); i++ {
		name := typeOfS.Field(i).Tag.Get("mapstructure")
		if name == "" {
			name = typeOfS.Field(i).Name
		}

		key := fmt.Sprintf("%v.%v", typeOfS.Name(), strings.ReplaceAll(strings.ToLower(name), " ", ""))
		keys = append(keys, key)
	}
	return
}
