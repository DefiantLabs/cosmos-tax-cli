package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

//
var apiEndpoints = map[string]string{
	"blocks_endpoint": "/cosmos/base/tendermint/v1beta1/blocks/%d",
	"txs_endpoint":    "/cosmos/tx/v1beta1/txs/%s",
}

//GetBlockByHeight makes a request to the Cosmos REST API to get a block by height
func GetBlockByHeight(host string, height int) (GetBlockByHeightResponse, error) {

	var result GetBlockByHeightResponse

	requestEndpoint := fmt.Sprintf(apiEndpoints["blocks_endpoint"], height)

	resp, err := http.Get(fmt.Sprintf("%s%s", host, requestEndpoint))

	if err != nil {
		return result, err
	}

	defer resp.Body.Close()

	//TODO: need to check resp.Status

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	json.Unmarshal(body, &result)

	return result, nil
}

//GetTxByHash makes a request to the Cosmos REST API to get a transaction by hash
func GetTxByHash(host string, hash string) (GetTxByHashResponse, error) {

	var result GetTxByHashResponse

	requestEndpoint := fmt.Sprintf(apiEndpoints["txs_endpoint"], hash)
	resp, err := http.Get(fmt.Sprintf("%s%s", host, requestEndpoint))

	if err != nil {
		return result, err
	}

	defer resp.Body.Close()

	//TODO: need to check resp.Status

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	json.Unmarshal(body, &result)

	return result, nil
}
