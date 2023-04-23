package liquidity

import (
	"fmt"

	"github.com/DefiantLabs/cosmos-tax-cli/tendermint/events"
	sdkTypes "github.com/cosmos/cosmos-sdk/types"
)

type WrapperBlockEventDepositToPool struct {
}

func (sf *WrapperBlockEventDepositToPool) GetType() string {
	return events.BlockEventDepositToPool
}

func (sf *WrapperBlockEventDepositToPool) HandleEvent(eventType string, event sdkTypes.Event) error {
	return nil
}

func (sf *WrapperBlockEventDepositToPool) ParseRelevantData() {
	//TODO: Implement Parsing of relevant data
}

func (sf *WrapperBlockEventDepositToPool) String() string {
	return fmt.Sprintf("Tendermint Liquidity event %s", sf.GetType())
}
