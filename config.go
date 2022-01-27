package main

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	Database database
	Api      api
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

func GetConfig(configFileLocation string) (Config, error) {
	var conf Config
	_, err := toml.DecodeFile(configFileLocation, &conf)
	return conf, err
}
