package config

import (
	"strings"

	"github.com/tendermint/starport/starport/pkg/cosmoscmd"
)

//SetChainConfig Set the chain prefix e.g. juno (prefix for account addresses).
func SetChainConfig(prefix string) {
	cosmoscmd.SetPrefixes(prefix)
}

func IsOsmosis(conf *Config) bool {
	return strings.Contains(
		strings.ToLower(conf.Lens.ChainID),
		"osmosis",
	)
}
