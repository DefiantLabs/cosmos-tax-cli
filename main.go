package main

import (
	"fmt"
	"log"
	"os"

	"github.com/DefiantLabs/cosmos-tax-cli/cmd"
	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/spf13/viper"
)

func main() {
	var conf config.Config
	cfgFile := ""

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		viper.SetConfigType("toml")
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Failed to find user home dir. Err: %v", err)
		}
		defaultCfgLocation := fmt.Sprintf("%s/.cosmos-tax-cli", home)

		viper.AddConfigPath(defaultCfgLocation)
		viper.SetConfigType("toml")
		viper.SetConfigName("config")
	}

	//Load defaults into a file at $HOME?
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Failed to read config file. Err: %v", err)
	}

	// Unmarshal the config into struct
	err = viper.Unmarshal(&conf)
	if err != nil {
		log.Fatalf("Failed to unmarshal config. Err: %v", err)
	}

	//TODO: Consider creating a set of defaults

	// Validate config
	err = conf.Validate()
	if err != nil {
		log.Fatalf("Failed to validate config. Err: %v", err)
	}

	cmd.SetupIndexer(conf)
	cmd.Index(conf)
}
