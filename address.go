package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"regexp"

	tx "cosmos-exporter/cosmos/modules/tx"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	legacybech32 "github.com/cosmos/cosmos-sdk/types/bech32/legacybech32"
)

//consider not using globals
var addressRegex *regexp.Regexp
var addressPrefix string

func setupAddressRegex(addressRegexPattern string) {
	addressRegex, _ = regexp.Compile(addressRegexPattern)
}

func setupAddressPrefix(addressPrefixString string) {
	addressPrefix = addressPrefixString
}

func ExtractTransactionAddresses(tx tx.MergedTx) []string {
	messagesAddresses := WalkFindStrings(tx.Tx.Body.Messages, addressRegex)
	//Consider walking logs - needs benchmarking compared to whole string search on raw log
	logAddresses := addressRegex.FindAllString(tx.TxResponse.RawLog, -1)
	addresses := append(messagesAddresses, logAddresses...)
	addressMap := make(map[string]string)
	for _, v := range addresses {
		addressMap[v] = ""
	}
	uniqueAddresses := make([]string, len(addressMap))
	i := 0
	for k := range addressMap {
		uniqueAddresses[i] = k
		i++
	}
	return uniqueAddresses
}

func ParseSignerAddress(pubkeyString string, keytype string) (string, error) {
	pubkey, err := getPubKeyFromRawString(pubkeyString, keytype)
	if err != nil {
		fmt.Println("Error getting public key from raw string")
		fmt.Println(err)
		return "", err
	}

	//this panics if conversion fails, TODO add recovery and error handling
	bech32address := cosmostypes.MustBech32ifyAddressBytes(addressPrefix, pubkey.Address().Bytes())
	return bech32address, nil
}

//the following code is taken from here https://github.com/cosmos/cosmos-sdk/blob/9ff6d5441db2260e7877724df65c0f2b8251d991/client/debug/main.go
//they do a check in bytesToPubkey for the keytype of "ed25519", we may want to pass in the keytype but this seems to work
//with secp256k1 keys without passing in the keytype.
//The key type seems to be in the @type key of in the public_key block in signer_infos so we could potentially pass it in there
func getPubKeyFromRawString(pkstr string, keytype string) (cryptotypes.PubKey, error) {
	bz, err := hex.DecodeString(pkstr)
	if err == nil {
		pk, ok := bytesToPubkey(bz, keytype)
		if ok {
			return pk, nil
		}
	}

	bz, err = base64.StdEncoding.DecodeString(pkstr)
	if err == nil {
		pk, ok := bytesToPubkey(bz, keytype)
		if ok {
			return pk, nil
		}
	}

	pk, err := legacybech32.UnmarshalPubKey(legacybech32.AccPK, pkstr)
	if err == nil {
		return pk, nil
	}

	pk, err = legacybech32.UnmarshalPubKey(legacybech32.ValPK, pkstr)
	if err == nil {
		return pk, nil
	}

	pk, err = legacybech32.UnmarshalPubKey(legacybech32.ConsPK, pkstr)
	if err == nil {
		return pk, nil
	}

	return nil, fmt.Errorf("pubkey '%s' invalid; expected hex, base64, or bech32 of correct size", pkstr)
}

func bytesToPubkey(bz []byte, keytype string) (cryptotypes.PubKey, bool) {
	if keytype == "ed25519" {
		if len(bz) == ed25519.PubKeySize {
			return &ed25519.PubKey{Key: bz}, true
		}
	}

	if len(bz) == secp256k1.PubKeySize {
		return &secp256k1.PubKey{Key: bz}, true
	}
	return nil, false
}
