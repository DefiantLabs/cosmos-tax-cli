package main

import (
	"github.com/BurntSushi/toml"
	"github.com/imdario/mergo"
)

type Config struct {
	Database           database
	Api                api
	ConfigFileLocation string
	Base               base
	Log                log
}

type database struct {
	Host     string
	Port     string
	Database string
	User     string
	Password string
}

type api struct {
	Host string
}

type base struct {
	StartBlock uint64
}

type log struct {
	Level string
}

func GetConfig(configFileLocation string) (Config, error) {
	var conf Config
	_, err := toml.DecodeFile(configFileLocation, &conf)
	return conf, err
}

func MergeConfigs(def Config, overide Config) Config {

	mergo.Merge(&overide, def)

	return overide
}
