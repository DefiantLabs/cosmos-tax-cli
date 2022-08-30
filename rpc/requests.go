package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	denoms "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/denoms"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	txTypes "github.com/cosmos/cosmos-sdk/types/tx"
	lensClient "github.com/strangelove-ventures/lens/client"
	lensQuery "github.com/strangelove-ventures/lens/client/query"
)

var apiEndpoints = map[string]string{
	"blocks_endpoint":              "/cosmos/base/tendermint/v1beta1/blocks/%d",
	"latest_block_endpoint":        "/blocks/latest",
	"txs_by_block_height_endpoint": "/cosmos/tx/v1beta1/txs?events=tx.height=%d&pagination.limit=100&order_by=ORDER_BY_UNSPECIFIED",
	"denoms_metadata":              "/cosmos/bank/v1beta1/denoms_metadata",
}

func GetEndpoint(key string) string {
	return apiEndpoints[key]
}

//GetBlockByHeight makes a request to the Cosmos RPC API and returns all the transactions for a specific block
func GetBlockByHeight(cl *lensClient.ChainClient, height int64) (*coretypes.ResultBlockResults, error) {
	options := lensQuery.QueryOptions{Height: height}
	query := lensQuery.Query{Client: cl, Options: &options}
	resp, err := query.BlockResults()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

//GetTxsByBlockHeight makes a request to the Cosmos RPC API and returns all the transactions for a specific block
func GetTxsByBlockHeight(cl *lensClient.ChainClient, height int64) (*txTypes.GetTxsEventResponse, error) {
	options := lensQuery.QueryOptions{Height: height}
	query := lensQuery.Query{Client: cl, Options: &options}
	resp, err := query.TxByHeight(cl.Codec)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

//IsCatchingUp true if the node is catching up to the chain, false otherwise
func IsCatchingUp(cl *lensClient.ChainClient) (bool, error) {
	query := lensQuery.Query{Client: cl, Options: &lensQuery.QueryOptions{}}
	ctx, cancel := query.GetQueryContext()
	defer cancel()

	resStatus, err := query.Client.RPCClient.Status(ctx)
	if err != nil {
		return false, err
	}
	return resStatus.SyncInfo.CatchingUp, nil
}

func GetLatestBlockHeight(cl *lensClient.ChainClient) (int64, error) {
	query := lensQuery.Query{Client: cl, Options: &lensQuery.QueryOptions{}}
	ctx, cancel := query.GetQueryContext()
	defer cancel()

	resStatus, err := query.Client.RPCClient.Status(ctx)
	if err != nil {
		return 0, err
	}
	return resStatus.SyncInfo.LatestBlockHeight, nil
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

func GetDenomsMetadatas(host string) (denoms.GetDenomsMetadatasResponse, error) {

	//TODO paginate
	var result denoms.GetDenomsMetadatasResponse

	requestEndpoint := apiEndpoints["denoms_metadata"]

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
