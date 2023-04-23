package liquidity

import (
	"fmt"

	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/events"
	tendermintEvents "github.com/DefiantLabs/cosmos-tax-cli/tendermint/events"
	sdk "github.com/cosmos/cosmos-sdk/types"
	abciTypes "github.com/tendermint/tendermint/abci/types"
)

type WrapperBlockEventDepositToPool struct {
	Event            abciTypes.Event
	Address          string
	AcceptedCoins    sdk.Coins
	PoolId           string
	Success          string
	PoolCoinReceived sdk.Coin
}

func (sf *WrapperBlockEventDepositToPool) GetType() string {
	return tendermintEvents.BlockEventDepositToPool
}

func (sf *WrapperBlockEventDepositToPool) HandleEvent(eventType string, event abciTypes.Event) error {
	sf.Event = event
	var poolCoinAmount string
	var poolCoinDenom string
	for _, attribute := range event.Attributes {
		switch string(attribute.Key) {
		case "depositor":
			sf.Address = string(attribute.Value)
		case "accepted_coins":
			acceptedCoins, err := sdk.ParseCoinsNormalized(string(attribute.Value))
			if err != nil {
				return err
			}
			sf.AcceptedCoins = acceptedCoins
		case "success":
			sf.Success = string(attribute.Value)
		case "pool_id":
			sf.PoolId = string(attribute.Value)
		case "pool_coin_amount":
			poolCoinAmount = string(attribute.Value)
		case "pool_coin_denom":
			poolCoinDenom = string(attribute.Value)
		}
	}

	poolCoinReceived, err := sdk.ParseCoinNormalized(poolCoinAmount + poolCoinDenom)
	if err != nil {
		return err
	}

	sf.PoolCoinReceived = poolCoinReceived

	return nil
}

func (sf *WrapperBlockEventDepositToPool) ParseRelevantData() []events.EventRelevantInformation {
	//TODO: Implement Parsing of relevant data
	return nil
}

func (sf *WrapperBlockEventDepositToPool) String() string {
	return fmt.Sprintf("Tendermint Liquidity event %s: Address %s deposited %s into pool %s and received %s with status %s", sf.GetType(), sf.Address, sf.AcceptedCoins, sf.PoolId, sf.PoolCoinReceived, sf.Success)
}
