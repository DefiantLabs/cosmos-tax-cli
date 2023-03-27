package osmosis

import (
	txTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/gamm"
)

// Unmarshal JSON to a particular type.
var MessageTypeHandler = map[string][]func() txTypes.CosmosMessage{
	gamm.MsgSwapExactAmountIn:       {func() txTypes.CosmosMessage { return &gamm.WrapperMsgSwapExactAmountIn{} }, func() txTypes.CosmosMessage { return &gamm.WrapperMsgSwapExactAmountIn2{} }, func() txTypes.CosmosMessage { return &gamm.WrapperMsgSwapExactAmountIn3{} }, func() txTypes.CosmosMessage { return &gamm.WrapperMsgSwapExactAmountIn4{} }},
	gamm.MsgSwapExactAmountOut:      {func() txTypes.CosmosMessage { return &gamm.WrapperMsgSwapExactAmountOut{} }},
	gamm.MsgJoinSwapExternAmountIn:  {func() txTypes.CosmosMessage { return &gamm.WrapperMsgJoinSwapExternAmountIn{} }},
	gamm.MsgJoinSwapShareAmountOut:  {func() txTypes.CosmosMessage { return &gamm.WrapperMsgJoinSwapShareAmountOut{} }},
	gamm.MsgJoinPool:                {func() txTypes.CosmosMessage { return &gamm.WrapperMsgJoinPool{} }},
	gamm.MsgExitSwapShareAmountIn:   {func() txTypes.CosmosMessage { return &gamm.WrapperMsgExitSwapShareAmountIn{} }, func() txTypes.CosmosMessage { return &gamm.WrapperMsgExitSwapShareAmountIn2{} }},
	gamm.MsgExitSwapExternAmountOut: {func() txTypes.CosmosMessage { return &gamm.WrapperMsgExitSwapExternAmountOut{} }},
	gamm.MsgExitPool:                {func() txTypes.CosmosMessage { return &gamm.WrapperMsgExitPool{} }, func() txTypes.CosmosMessage { return &gamm.WrapperMsgExitPool2{} }},
	gamm.MsgCreatePool:              {func() txTypes.CosmosMessage { return &gamm.WrapperMsgCreatePool{} }, func() txTypes.CosmosMessage { return &gamm.WrapperMsgCreatePool2{} }},
}
