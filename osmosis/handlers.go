package osmosis

import (
	txTypes "github.com/DefiantLabs/cosmos-indexer/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-indexer/osmosis/modules/concentratedliquidity"
	"github.com/DefiantLabs/cosmos-indexer/osmosis/modules/gamm"
	"github.com/DefiantLabs/cosmos-indexer/osmosis/modules/poolmanager"
)

// MessageTypeHandler is used to unmarshal JSON to a particular type.
var MessageTypeHandler = map[string][]func() txTypes.CosmosMessage{
	gamm.MsgSwapExactAmountIn:                       {func() txTypes.CosmosMessage { return &gamm.WrapperMsgSwapExactAmountIn{} }, func() txTypes.CosmosMessage { return &gamm.WrapperMsgSwapExactAmountIn2{} }, func() txTypes.CosmosMessage { return &gamm.WrapperMsgSwapExactAmountIn3{} }, func() txTypes.CosmosMessage { return &gamm.WrapperMsgSwapExactAmountIn4{} }},
	gamm.MsgSwapExactAmountOut:                      {func() txTypes.CosmosMessage { return &gamm.WrapperMsgSwapExactAmountOut{} }},
	gamm.MsgJoinSwapExternAmountIn:                  {func() txTypes.CosmosMessage { return &gamm.WrapperMsgJoinSwapExternAmountIn{} }, func() txTypes.CosmosMessage { return &gamm.WrapperMsgJoinSwapExternAmountIn2{} }},
	gamm.MsgJoinSwapShareAmountOut:                  {func() txTypes.CosmosMessage { return &gamm.WrapperMsgJoinSwapShareAmountOut{} }, func() txTypes.CosmosMessage { return &gamm.WrapperMsgJoinSwapShareAmountOut2{} }},
	gamm.MsgJoinPool:                                {func() txTypes.CosmosMessage { return &gamm.WrapperMsgJoinPool{} }},
	gamm.MsgExitSwapShareAmountIn:                   {func() txTypes.CosmosMessage { return &gamm.WrapperMsgExitSwapShareAmountIn{} }, func() txTypes.CosmosMessage { return &gamm.WrapperMsgExitSwapShareAmountIn2{} }},
	gamm.MsgExitSwapExternAmountOut:                 {func() txTypes.CosmosMessage { return &gamm.WrapperMsgExitSwapExternAmountOut{} }},
	gamm.MsgExitPool:                                {func() txTypes.CosmosMessage { return &gamm.WrapperMsgExitPool{} }, func() txTypes.CosmosMessage { return &gamm.WrapperMsgExitPool2{} }},
	gamm.MsgCreatePool:                              {func() txTypes.CosmosMessage { return &gamm.WrapperMsgCreatePool{} }, func() txTypes.CosmosMessage { return &gamm.WrapperMsgCreatePool2{} }},
	poolmanager.MsgSwapExactAmountIn:                {func() txTypes.CosmosMessage { return &poolmanager.WrapperMsgSwapExactAmountIn{} }},
	poolmanager.MsgSwapExactAmountOut:               {func() txTypes.CosmosMessage { return &poolmanager.WrapperMsgSwapExactAmountOut{} }},
	concentratedliquidity.MsgCreatePosition:         {func() txTypes.CosmosMessage { return &concentratedliquidity.WrapperMsgCreatePosition{} }},
	concentratedliquidity.MsgWithdrawPosition:       {func() txTypes.CosmosMessage { return &concentratedliquidity.WrapperMsgWithdrawPosition{} }},
	concentratedliquidity.MsgCollectSpreadRewards:   {func() txTypes.CosmosMessage { return &concentratedliquidity.WrapperMsgCollectSpreadRewards{} }},
	concentratedliquidity.MsgCreateConcentratedPool: {func() txTypes.CosmosMessage { return &concentratedliquidity.WrappeMsgCreateConcentratedPool{} }},
}
