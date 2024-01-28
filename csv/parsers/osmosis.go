package parsers

import (
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/concentratedliquidity"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/gamm"
)

var IsOsmosisJoin = map[string]bool{
	gamm.MsgJoinSwapExternAmountIn: true,
	gamm.MsgJoinSwapShareAmountOut: true,
	gamm.MsgJoinPool:               true,
}

var IsOsmosisExit = map[string]bool{
	gamm.MsgExitSwapShareAmountIn:   true,
	gamm.MsgExitSwapExternAmountOut: true,
	gamm.MsgExitPool:                true,
}

var IsOsmosisConcentratedLiquidity = map[string]bool{
	concentratedliquidity.MsgAddToPosition:     true,
	concentratedliquidity.MsgWithdrawPosition:  true,
	concentratedliquidity.MsgCreatePosition:    true,
	concentratedliquidity.MsgTransferPositions: true,
}

// IsOsmosisLpTxGroup is used as a guard for adding messages to the group.
var IsOsmosisLpTxGroup = make(map[string]bool)

func init() {
	for messageType := range IsOsmosisJoin {
		IsOsmosisLpTxGroup[messageType] = true
	}

	for messageType := range IsOsmosisExit {
		IsOsmosisLpTxGroup[messageType] = true
	}
}
