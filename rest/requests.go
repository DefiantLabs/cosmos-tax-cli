package rest

import (
	tx "cosmos-exporter/cosmos/modules/tx"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

var apiEndpoints = map[string]string{
	"blocks_endpoint":              "/cosmos/base/tendermint/v1beta1/blocks/%d",
	"latest_block_endpoint":        "/blocks/latest",
	"txs_by_block_height_endpoint": "/cosmos/tx/v1beta1/txs?events=tx.height=%d&pagination.limit=100&pagination.offset=%d&order_by=ORDER_BY_UNSPECIFIED",
}

//GetBlockByHeight makes a request to the Cosmos REST API to get a block by height
func GetBlockByHeight(host string, height uint64) (tx.GetBlockByHeightResponse, error) {

	var result tx.GetBlockByHeightResponse

	requestEndpoint := fmt.Sprintf(apiEndpoints["blocks_endpoint"], height)

	resp, err := http.Get(fmt.Sprintf("%s%s", host, requestEndpoint))

	if err != nil {
		return result, err
	}

	defer resp.Body.Close()

	err = checkResponseErrorCode(requestEndpoint, resp)
	if err != nil {
		return result, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(body, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

//GetTxsByBlockHeight makes multiple requests to the Cosmos REST API until all the transactions for a block are paginated through and captured
func GetTxsByBlockHeightPaginated(host string, height uint64, throttling int64) (tx.GetTxByBlockHeightResponse, error) {

	var results tx.GetTxByBlockHeightResponse
	var offset uint64 = 0

	results, err := GetTxsByBlockHeight(host, height, offset)
	if err != nil {
		fmt.Println("Error getting transactions by block height", err)
		return results, err
	}

	paginationTotal, _ := strconv.Atoi(results.Pagination.Total)

	if len(results.Txs) < paginationTotal {
		offset = uint64(len(results.Txs))
		for {
			if throttling != 0 {
				time.Sleep(time.Second * time.Duration(throttling))
			}
			nextResult, err := GetTxsByBlockHeight(host, height, offset)
			if err != nil {
				fmt.Println("Error getting transactions by block height", err)
				return results, err
			}

			//append paged results to original results
			results.Txs = append(results.Txs, nextResult.Txs...)
			results.TxResponses = append(results.TxResponses, nextResult.TxResponses...)

			if len(results.Txs) == paginationTotal {
				break
			} else {
				offset = uint64(len(results.Txs))
			}
		}
	}

	return results, nil
}

//GetTxsByBlockHeight makes a request to the Cosmos REST API and returns all the transactions for a specific block
func GetTxsByBlockHeight(host string, height uint64, offset uint64) (tx.GetTxByBlockHeightResponse, error) {

	var result tx.GetTxByBlockHeightResponse

	requestEndpoint := fmt.Sprintf(apiEndpoints["txs_by_block_height_endpoint"], height, offset)

	resp, err := http.Get(fmt.Sprintf("%s%s", host, requestEndpoint))

	if err != nil {
		return result, err
	}

	defer resp.Body.Close()

	err = checkResponseErrorCode(requestEndpoint, resp)

	if err != nil {
		return result, err
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return result, err
	}

	err = json.Unmarshal(body, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func GetLatestBlock(host string) (tx.GetLatestBlockResponse, error) {

	var result tx.GetLatestBlockResponse

	requestEndpoint := apiEndpoints["latest_block_endpoint"]

	resp, err := http.Get(fmt.Sprintf("%s%s", host, requestEndpoint))

	if err != nil {
		return result, err
	}

	defer resp.Body.Close()

	err = checkResponseErrorCode(requestEndpoint, resp)

	if err != nil {
		return result, err
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return result, err
	}

	err = json.Unmarshal(body, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func checkResponseErrorCode(requestEndpoint string, resp *http.Response) error {

	if resp.StatusCode != 200 {
		fmt.Println("Error getting response")
		body, _ := ioutil.ReadAll(resp.Body)
		errorString := fmt.Sprintf("Error getting response for endpoint %s: Status %s Body %s", requestEndpoint, resp.Status, body)

		err := errors.New(errorString)

		return err
	}

	return nil

}
