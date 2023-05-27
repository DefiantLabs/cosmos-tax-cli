package config

import (
	lensClient "github.com/DefiantLabs/probe/client"
)

func GetLensClient(conf lens) *lensClient.ChainClient {
	// IMPORTANT: the actual keyring-test will be searched for at the path {homepath}/keys/{ChainID}/keyring-test.
	// You can use lens default settings to generate that directory appropriately then move it to the desired path.
	// For example, 'lens keys restore default' will restore the key to the default keyring (e.g. /home/kyle/.lens/...)
	// and you can move all of the necessary keys to whatever homepath you want to use. Or you can use --home flag.
	cl, err := lensClient.NewChainClient(GetLensConfig(conf, true), "", nil, nil)
	if err != nil {
		Log.Fatalf("Error connecting to chain. Err: %v", err)
	}
	// RegisterAdditionalTypes(cl)
	return cl
}

// func RegisterAdditionalTypes(cc *lensClient.ChainClient) {
// 	// Register IBC types
// 	// ibcTypes.RegisterLegacyAminoCodec(cc.Codec.Amino)
// 	ibcTypes.RegisterInterfaces(cc.Codec.InterfaceRegistry)
// }

func GetLensConfig(conf lens, debug bool) *lensClient.ChainClientConfig {
	return &lensClient.ChainClientConfig{
		Key:            "default",
		ChainID:        conf.ChainID,
		RPCAddr:        conf.RPC,
		AccountPrefix:  conf.AccountPrefix,
		KeyringBackend: "test",
		Debug:          debug,
		Timeout:        "10s",
		OutputFormat:   "json",
		Modules:        lensClient.ModuleBasics,
	}
}
