package rpc

import (
	"encoding/json"
	"fmt"
	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
	"github.com/DefiantLabs/cosmos-tax-cli-private/core"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/strangelove-ventures/lens/client"
	lensClient "github.com/strangelove-ventures/lens/client"
	lensQuery "github.com/strangelove-ventures/lens/client/query"

	"github.com/go-co-op/gocron"
)

func getHomePath(t *testing.T) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		assert.Nil(t, err)
	}
	return fmt.Sprintf("%v/.lens", homeDir)
}

// setup does pre-run setup configurations.
//   - Loads the application config from config.tml, cli args and parses/merges
//   - Connects to the database and returns the db object
//   - Returns various values used throughout the application
//
//nolint:unused
func setupRPC() (*config.Config, *gocron.Scheduler, error) {
	argConfig, err := config.ParseArgs(os.Stderr, os.Args[1:])

	if err != nil {
		return nil, nil, err
	}

	var location string
	if argConfig.ConfigFileLocation != "" {
		location = argConfig.ConfigFileLocation
	} else {
		location = "./config.toml"
	}

	fileConfig, err := config.GetConfig(location)

	if err != nil {
		fmt.Println("Error opening configuration file", err)
		return nil, nil, err
	}

	cfg := config.MergeConfigs(fileConfig, argConfig)

	//0 is an invalid starting block, set it to 1
	if cfg.Base.StartBlock == 0 {
		cfg.Base.StartBlock = 1
	}

	//TODO: create config values for the prefixes here
	//Could potentially check Node info at startup and pass in ourselves?
	core.SetupAddressRegex("juno(valoper)?1[a-z0-9]{38}")
	core.SetupAddressPrefix("juno")

	scheduler := gocron.NewScheduler(time.UTC)
	return &cfg, scheduler, nil
}

func TestRPC(t *testing.T) {
	block := 2
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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		assert.Nil(t, err)
	}
	cl := GetOsmosisTestClient(fmt.Sprintf("%v/.lens", homeDir))
	resp, err := GetTxsByBlockHeight(cl, 2620000)
	assert.Empty(t, err)
	hasIbcType := false

	for txIdx := range resp.Txs {
		currTx := resp.Txs[txIdx]

		//Get the Messages and Message Logs
		for msgIdx := range currTx.Body.Messages {
			currMsg := currTx.Body.Messages[msgIdx].GetCachedValue()
			if currMsg != nil {
				msg := currMsg.(types.Msg)
				typeURL := types.MsgTypeURL(msg)
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

func GetTestClient(t *testing.T) *lensClient.ChainClient {
	homepath := getHomePath(t)
	//IMPORTANT: the actual keyring-test will be searched for at the path {homepath}/keys/{ChainID}/keyring-test.
	//You can use lens default settings to generate that directory appropriately then move it to the desired path.
	//For example, 'lens keys restore default' will restore the key to the default keyring (e.g. /home/kyle/.lens/...)
	//and you can move all of the necessary keys to whatever homepath you want to use. Or you can use --home flag.
	cl, err := lensClient.NewChainClient(GetJunoConfig(homepath, true), homepath, nil, nil)
	assert.Nil(t, err)
	config.RegisterAdditionalTypes(cl)
	return cl
}

func GetOsmosisTestClient(homepath string) *lensClient.ChainClient {
	//IMPORTANT: the actual keyring-test will be searched for at the path {homepath}/keys/{ChainID}/keyring-test.
	//You can use lens default settings to generate that directory appropriately then move it to the desired path.
	//For example, 'lens keys restore default' will restore the key to the default keyring (e.g. /home/kyle/.lens/...)
	//and you can move all of the necessary keys to whatever homepath you want to use. Or you can use --home flag.
	cl, _ := lensClient.NewChainClient(GetOsmosisConfig(homepath, true), homepath, nil, nil)
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
	cl := GetTestClient(t)
	keyNameOrAddress := cl.Config.Key

	address, err := cl.AccountFromKeyOrAddress(keyNameOrAddress)
	if err != nil {
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
	cl := GetTestClient(t)
	//requestEndpoint := fmt.Sprintf(rest.GetEndpoint("txs_by_block_height_endpoint"), height)
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
