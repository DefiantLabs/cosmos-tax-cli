package rpc

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
	"github.com/DefiantLabs/cosmos-tax-cli-private/core"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/go-co-op/gocron"
	"github.com/strangelove-ventures/lens/client"
	lensClient "github.com/strangelove-ventures/lens/client"
	lensQuery "github.com/strangelove-ventures/lens/client/query"
	"github.com/stretchr/testify/assert"
)

func getHomePath(t *testing.T) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		assert.Nil(t, err)
	}
	return fmt.Sprintf("%v/.lens", homeDir)
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

// setup does pre-run setup configurations.
//   - Loads the application config from config.tml, cli args and parses/merges
//   - Connects to the database and returns the db object
//   - Returns various values used throughout the application
//
// nolint:unused
func setupRPC(t *testing.T) *gocron.Scheduler {
	if testConfig == nil {
		setupConfig(t)
	}
	// TODO: create config values for the prefixes here
	// Could potentially check Node info at startup and pass in ourselves?
	core.SetupAddressRegex(testConfig.Base.AddressRegex)
	core.SetupAddressPrefix(testConfig.Base.AddressPrefix)

	scheduler := gocron.NewScheduler(time.UTC)
	return scheduler
}

func TestRPC(t *testing.T) {
	block := 2620000
	err := lensQueryBank(t, int64(block))
	if err != nil {
		assert.Nil(t, err, "should not error writing to CSV")
	}

	err = rpcQueryTx(t, int64(block))
	if err != nil {
		assert.Nil(t, err, "should not error calling rpc")
	}
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

func GetJunoTestClient(t *testing.T) *lensClient.ChainClient {
	homepath := getHomePath(t)
	// IMPORTANT: the actual keyring-test will be searched for at the path {homepath}/keys/{ChainID}/keyring-test.
	// You can use lens default settings to generate that directory appropriately then move it to the desired path.
	// For example, 'lens keys restore default' will restore the key to the default keyring (e.g. /home/kyle/.lens/...)
	// and you can move all of the necessary keys to whatever homepath you want to use. Or you can use --home flag.
	cl, err := lensClient.NewChainClient(GetJunoConfig(homepath, true), homepath, nil, nil)
	assert.Nil(t, err)
	config.RegisterAdditionalTypes(cl)
	return cl
}

func GetOsmosisTestClient(t *testing.T) *lensClient.ChainClient {
	homepath := getHomePath(t)
	// IMPORTANT: the actual keyring-test will be searched for at the path {homepath}/keys/{ChainID}/keyring-test.
	// You can use lens default settings to generate that directory appropriately then move it to the desired path.
	// For example, 'lens keys restore default' will restore the key to the default keyring (e.g. /home/kyle/.lens/...)
	// and you can move all of the necessary keys to whatever homepath you want to use. Or you can use --home flag.
	cl, err := lensClient.NewChainClient(GetOsmosisConfig(homepath, true), homepath, nil, nil)
	assert.Nil(t, err)
	config.RegisterAdditionalTypes(cl)
	return cl
}

func GetJunoConfig(keyHome string, debug bool) *lensClient.ChainClientConfig {
	return &lensClient.ChainClientConfig{
		Key:            "default",
		ChainID:        "testing",
		RPCAddr:        "http://localhost:26657",
		GRPCAddr:       "http://localhost:26657",
		AccountPrefix:  "juno",
		KeyringBackend: "test",
		GasAdjustment:  1.2,
		GasPrices:      "0ustake",
		KeyDirectory:   keyHome,
		Debug:          debug,
		Timeout:        "10s",
		OutputFormat:   "json",
		SignModeStr:    "direct",
		Modules:        client.ModuleBasics,
	}
}

func GetOsmosisConfig(keyHome string, debug bool) *lensClient.ChainClientConfig {
	log.Println(keyHome)
	return &lensClient.ChainClientConfig{
		Key:            "default",
		ChainID:        "osmosis-1",
		RPCAddr:        "https://osmosis-mainnet-archive.allthatnode.com:26657",
		GRPCAddr:       "https://osmosis-mainnet-archive.allthatnode.com:26657",
		AccountPrefix:  "osmo",
		KeyringBackend: "test",
		GasAdjustment:  1.2,
		GasPrices:      "0uosmo",
		KeyDirectory:   keyHome,
		Debug:          debug,
		Timeout:        "10s",
		OutputFormat:   "json",
		SignModeStr:    "direct",
		Modules:        client.ModuleBasics,
	}
}

func lensQueryBank(t *testing.T, height int64) error {
	cl := GetOsmosisTestClient(t)
	keyNameOrAddress := cl.Config.Key
	address, err := cl.AccountFromKeyOrAddress(keyNameOrAddress)
	if err != nil {
		log.Println("Error getting account from key or address: ", keyNameOrAddress)
		return err
	}
	encodedAddr := cl.MustEncodeAccAddr(address)
	options := lensQuery.QueryOptions{Height: height}
	query := lensQuery.Query{Client: cl, Options: &options}
	balance, err := query.Balances(encodedAddr)
	fmt.Printf("Balance: %s\n", balance)
	return err
}

func rpcQueryTx(t *testing.T, height int64) error {
	cl := GetOsmosisTestClient(t)
	// requestEndpoint := fmt.Sprintf(rest.GetEndpoint("txs_by_block_height_endpoint"), height)
	options := lensQuery.QueryOptions{Height: height}
	query := lensQuery.Query{Client: cl, Options: &options}
	resp, err := query.TxByHeight(cl.Codec)
	if err != nil {
		return err
	}
	jResp, err := json.Marshal(*resp)
	fmt.Printf("Resp: %s\n", jResp)
	return err
}
