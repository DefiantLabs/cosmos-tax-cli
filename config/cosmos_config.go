package config

import "github.com/tendermint/starport/starport/pkg/cosmoscmd"

//SetChainConfig Set the chain prefix e.g. juno (prefix for account addresses).
func SetChainConfig(prefix string) {
	cosmoscmd.SetPrefixes(prefix)
}
