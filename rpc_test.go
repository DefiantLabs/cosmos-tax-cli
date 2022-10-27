package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/core"

	"github.com/strangelove-ventures/lens/client"
	lensClient "github.com/strangelove-ventures/lens/client"
	lensQuery "github.com/strangelove-ventures/lens/client/query"

	"github.com/go-co-op/gocron"
)

// setup does pre-run setup configurations.
//   - Loads the application config from config.tml, cli args and parses/merges
//   - Connects to the database and returns the db object
//   - Returns various values used throughout the application
//
//nolint:unused
func setup_rpc() (*config.Config, *gocron.Scheduler, error) {
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

func TestRpc(t *testing.T) {
	block := 2
	err := lens_query_bank(int64(block))
	if err != nil {
		t.Fatal("Failed to write CSV to disk")
	}

	err = rpc_query_tx(int64(block))
	if err != nil {
		t.Fatal("Error calling rpc_query_tx. Err: ", err)
	}
}

func GetTestClient() *lensClient.ChainClient {
	//IMPORTANT: the actual keyring-test will be searched for at the path {homepath}/keys/{ChainID}/keyring-test.
	//You can use lens default settings to generate that directory appropriately then move it to the desired path.
	//For example, 'lens keys restore default' will restore the key to the default keyring (e.g. /home/kyle/.lens/...)
	//and you can move all of the necessary keys to whatever homepath you want to use. Or you can use --home flag.
	homepath := "/home/kyle/.lens"
	cl, _ := lensClient.NewChainClient(GetJunoConfig(homepath, true), homepath, nil, nil)
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

func lens_query_bank(height int64) error {
	cl := GetTestClient()
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

func rpc_query_tx(height int64) error {
	cl := GetTestClient()
	//requestEndpoint := fmt.Sprintf(rest.GetEndpoint("txs_by_block_height_endpoint"), height)
	options := lensQuery.QueryOptions{Height: height}
	query := lensQuery.Query{Client: cl, Options: &options}
	resp, err := query.TxByHeight(cl.Codec)
	if err != nil {
		return err
	}
	j_resp, err := json.Marshal(*resp)
	fmt.Printf("Resp: %s\n", j_resp)
	return err
}
