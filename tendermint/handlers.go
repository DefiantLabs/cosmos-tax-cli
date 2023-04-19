package tendermint

import (
	txTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-tax-cli/tendermint/modules/liquidity"
)

var MessageTypeHandler = map[string][]func() txTypes.CosmosMessage{
	liquidity.MsgDepositWithinBatch:  {func() txTypes.CosmosMessage { return &liquidity.WrapperMsgDepositWithinBatch{} }},
	liquidity.MsgWithdrawWithinBatch: {func() txTypes.CosmosMessage { return &liquidity.WrapperMsgWithdrawWithinBatch{} }},
}
