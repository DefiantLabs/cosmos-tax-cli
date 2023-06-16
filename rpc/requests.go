package rpc

import (
	coretypes "github.com/cometbft/cometbft/rpc/core/types"

	probeClient "github.com/DefiantLabs/probe/client"
	probeEpochsTypes "github.com/DefiantLabs/probe/client/codec/osmosis/v15/x/epochs/types"
	probeProtorevTypes "github.com/DefiantLabs/probe/client/codec/osmosis/v15/x/protorev/types"
	probeQuery "github.com/DefiantLabs/probe/query"
	probeOsmosisQuery "github.com/DefiantLabs/probe/query/osmosis"
	"github.com/cosmos/cosmos-sdk/types/query"
	txTypes "github.com/cosmos/cosmos-sdk/types/tx"
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

// GetBlockByHeight makes a request to the Cosmos RPC API and returns all the transactions for a specific block
func GetBlockByHeight(cl *probeClient.ChainClient, height int64) (*coretypes.ResultBlockResults, error) {
	options := probeQuery.QueryOptions{Height: height}
	query := probeQuery.Query{Client: cl, Options: &options}
	resp, err := query.BlockResults()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetBlockTimestamp
func GetBlock(cl *probeClient.ChainClient, height int64) (*coretypes.ResultBlock, error) {
	options := probeQuery.QueryOptions{Height: height}
	query := probeQuery.Query{Client: cl, Options: &options}
	resp, err := query.Block()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetTxsByBlockHeight makes a request to the Cosmos RPC API and returns all the transactions for a specific block
func GetTxsByBlockHeight(cl *probeClient.ChainClient, height int64) (*txTypes.GetTxsEventResponse, error) {
	pg := query.PageRequest{Limit: 100}
	options := probeQuery.QueryOptions{Height: height, Pagination: &pg}
	query := probeQuery.Query{Client: cl, Options: &options}
	resp, err := query.TxByHeight(cl.Codec)
	if err != nil {
		return nil, err
	}

	// handle pagination if needed
	if resp != nil && resp.Pagination != nil {
		// if there are more total objects than we have so far, keep going
		for resp.Pagination.Total > uint64(len(resp.Txs)) {
			query.Options.Pagination.Offset = uint64(len(resp.Txs))
			chunkResp, err := query.TxByHeight(cl.Codec)
			if err != nil {
				return nil, err
			}
			resp.Txs = append(resp.Txs, chunkResp.Txs...)
			resp.TxResponses = append(resp.TxResponses, chunkResp.TxResponses...)
		}
	}

	return resp, nil
}

// IsCatchingUp true if the node is catching up to the chain, false otherwise
func IsCatchingUp(cl *probeClient.ChainClient) (bool, error) {
	query := probeQuery.Query{Client: cl, Options: &probeQuery.QueryOptions{}}
	ctx, cancel := query.GetQueryContext()
	defer cancel()

	resStatus, err := query.Client.RPCClient.Status(ctx)
	if err != nil {
		return false, err
	}
	return resStatus.SyncInfo.CatchingUp, nil
}

func GetLatestBlockHeight(cl *probeClient.ChainClient) (int64, error) {
	query := probeQuery.Query{Client: cl, Options: &probeQuery.QueryOptions{}}
	ctx, cancel := query.GetQueryContext()
	defer cancel()

	resStatus, err := query.Client.RPCClient.Status(ctx)
	if err != nil {
		return 0, err
	}
	return resStatus.SyncInfo.LatestBlockHeight, nil
}

// GetEpochsAtHeight makes a request to the Cosmos RPC API and returns the Epoch at a specific height
func GetEpochsAtHeight(cl *probeClient.ChainClient, height int64) (*probeEpochsTypes.QueryEpochsInfoResponse, error) {
	options := probeQuery.QueryOptions{}
	query := probeQuery.Query{Client: cl, Options: &options}
	resp, err := probeOsmosisQuery.EpochsAtHeightRPC(&query, height)
	return resp, err
}

// GetEpochsAtHeight makes a request to the Cosmos RPC API and returns the Epoch at a specific height
func GetProtorevDeveloperAccount(cl *probeClient.ChainClient) (*probeProtorevTypes.QueryGetProtoRevDeveloperAccountResponse, error) {
	options := probeQuery.QueryOptions{}
	query := probeQuery.Query{Client: cl, Options: &options}
	resp, err := probeOsmosisQuery.ProtorevDeveloperAccountRPC(&query)
	return resp, err
}
