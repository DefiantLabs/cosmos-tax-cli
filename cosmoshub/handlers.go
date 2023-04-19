package cosmoshub

import (
	txTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-tax-cli/tendermint"
)

var MessageTypeHandler = map[string][]func() txTypes.CosmosMessage{}

func init() {
	//CosmosHub has tendermint liquidity module handler
	// If we ever integrate more modules developed by Tendermint, we may want to filter those out of here
	for k, v := range tendermint.MessageTypeHandler {
		MessageTypeHandler[k] = v
	}
}
