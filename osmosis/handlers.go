package osmosis

import (
	txTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/gamm"
)

//Unmarshal JSON to a particular type.
var MessageTypeHandler = map[string]func() txTypes.CosmosMessage{
	"/osmosis.gamm.v1beta1.MsgSwapExactAmountIn":      func() txTypes.CosmosMessage { return &gamm.WrapperMsgSwapExactAmountIn{} },
	"/osmosis.gamm.v1beta1.MsgSwapExactAmountOut":     func() txTypes.CosmosMessage { return &gamm.WrapperMsgSwapExactAmountOut{} },
	"/osmosis.gamm.v1beta1.MsgJoinSwapExternAmountIn": func() txTypes.CosmosMessage { return &gamm.WrapperMsgJoinSwapExternAmountIn{} },
	"/osmosis.gamm.v1beta1.MsgJoinPool":               func() txTypes.CosmosMessage { return &gamm.WrapperMsgJoinPool{} },
	"/osmosis.gamm.v1beta1.MsgExitPool":               func() txTypes.CosmosMessage { return &gamm.WrapperMsgExitPool{} },
}
