package rpc

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/DefiantLabs/cosmos-indexer/config"
	"github.com/DefiantLabs/probe/client"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
)

func getHomePath(t *testing.T) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		assert.Nil(t, err)
	}
	return fmt.Sprintf("%v/.probe", homeDir)
}

// nolint:unused
var testConfig *config.Config

// nolint:unused
func setupConfig(t *testing.T) {
	argConfig, _, _, err := config.ParseArgs(os.Stderr, os.Args[1:])
	if err != nil {
		assert.Nil(t, err)
	}

	var location string
	if argConfig.ConfigFileLocation != "" {
		location = argConfig.ConfigFileLocation
	} else {
		location = "./config.toml"
	}

	fileConfig, err := config.GetConfig(location)
	if err != nil {
		assert.Nil(t, err)
	}

	cfg := config.MergeConfigs(fileConfig, argConfig)

	// 0 is an invalid starting block, set it to 1
	if cfg.Base.StartBlock == 0 {
		cfg.Base.StartBlock = 1
	}
	testConfig = &cfg
}

func TestDecodeIBCTypes(t *testing.T) {
	cl := GetOsmosisTestClient(t)
	resp, err := GetTxsByBlockHeight(cl, 2620000)
	assert.Empty(t, err)
	hasIbcType := false

	for txIdx := range resp.Txs {
		currTx := resp.Txs[txIdx]

		// Get the Messages and Message Logs
		for msgIdx := range currTx.Body.Messages {
			currMsg := currTx.Body.Messages[msgIdx].GetCachedValue()
			if currMsg != nil {
				typeURL := types.MsgTypeURL(currMsg.(types.Msg))
				if strings.Contains(typeURL, "MsgTransfer") {
					hasIbcType = true
				}
			} else {
				t.Error("tx message could not be processed. CachedValue is not present")
			}
		}
	}

	assert.True(t, hasIbcType)
}

func GetJunoTestClient(t *testing.T) *client.ChainClient {
	homepath := getHomePath(t)
	// IMPORTANT: the actual keyring-test will be searched for at the path {homepath}/keys/{ChainID}/keyring-test.
	// You can use probe default settings to generate that directory appropriately then move it to the desired path.
	// For example, 'probe keys restore default' will restore the key to the default keyring (e.g. /home/kyle/.probe/...)
	// and you can move all of the necessary keys to whatever homepath you want to use. Or you can use --home flag.
	cl, err := client.NewChainClient(GetJunoConfig(homepath, true), homepath, nil, nil)
	assert.Nil(t, err)
	return cl
}

func GetOsmosisTestClient(t *testing.T) *client.ChainClient {
	homepath := getHomePath(t)
	// IMPORTANT: the actual keyring-test will be searched for at the path {homepath}/keys/{ChainID}/keyring-test.
	// You can use probe default settings to generate that directory appropriately then move it to the desired path.
	// For example, 'probe keys restore default' will restore the key to the default keyring (e.g. /home/kyle/.probe/...)
	// and you can move all of the necessary keys to whatever homepath you want to use. Or you can use --home flag.
	cl, err := client.NewChainClient(GetOsmosisConfig(homepath, true), homepath, nil, nil)
	assert.Nil(t, err)
	return cl
}

func GetJunoConfig(keyHome string, debug bool) *client.ChainClientConfig {
	return &client.ChainClientConfig{
		Key:            "default",
		ChainID:        "testing",
		RPCAddr:        "http://localhost:26657",
		AccountPrefix:  "juno",
		KeyringBackend: "test",
		KeyDirectory:   keyHome,
		Debug:          debug,
		Timeout:        "10s",
		OutputFormat:   "json",
		Modules:        client.ModuleBasics,
	}
}

func GetOsmosisConfig(keyHome string, debug bool) *client.ChainClientConfig {
	log.Println(keyHome)
	return &client.ChainClientConfig{
		Key:            "default",
		ChainID:        "osmosis-1",
		RPCAddr:        "https://osmosis-mainnet-archive.allthatnode.com:26657",
		AccountPrefix:  "osmo",
		KeyringBackend: "test",
		KeyDirectory:   keyHome,
		Debug:          debug,
		Timeout:        "10s",
		OutputFormat:   "json",
		Modules:        client.ModuleBasics,
	}
}
