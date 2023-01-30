package osmosis

import (
	txTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/gamm"
)

// Unmarshal JSON to a particular type.
var MessageTypeHandler = map[string]func() txTypes.CosmosMessage{
	gamm.MsgSwapExactAmountIn:       func() txTypes.CosmosMessage { return &gamm.WrapperMsgSwapExactAmountIn{} },
	gamm.MsgSwapExactAmountOut:      func() txTypes.CosmosMessage { return &gamm.WrapperMsgSwapExactAmountOut{} },
	gamm.MsgJoinSwapExternAmountIn:  func() txTypes.CosmosMessage { return &gamm.WrapperMsgJoinSwapExternAmountIn{} },
	gamm.MsgJoinSwapShareAmountOut:  func() txTypes.CosmosMessage { return &gamm.WrapperMsgJoinSwapShareAmountOut{} },
	gamm.MsgJoinPool:                func() txTypes.CosmosMessage { return &gamm.WrapperMsgJoinPool{} },
	gamm.MsgExitSwapShareAmountIn:   func() txTypes.CosmosMessage { return &gamm.WrapperMsgExitSwapShareAmountIn{} },
	gamm.MsgExitSwapExternAmountOut: func() txTypes.CosmosMessage { return &gamm.WrapperMsgExitSwapExternAmountOut{} },
	gamm.MsgExitPool:                func() txTypes.CosmosMessage { return &gamm.WrapperMsgExitPool{} },
}
