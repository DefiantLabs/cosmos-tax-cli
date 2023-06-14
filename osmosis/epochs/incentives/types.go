package incentives

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/events"
	dbTypes "github.com/DefiantLabs/cosmos-tax-cli/db"
	osmosisEvents "github.com/DefiantLabs/cosmos-tax-cli/osmosis/events"
	abciTypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type WrapperBlockDistribution struct {
	Event           abciTypes.Event
	RewardsReceived sdk.Coins
	ReceiverAddress string
}

func (sf *WrapperBlockDistribution) GetType() string {
	return osmosisEvents.BlockEventDistribution
}

func (sf *WrapperBlockDistribution) HandleEvent(eventType string, event abciTypes.Event) error {
	var receiverAddr string
	var receiverAmount string

	for _, attr := range event.Attributes {

		key, err := base64.StdEncoding.DecodeString(attr.Key)
		if err != nil {
			return fmt.Errorf("could not decode event attribute key: %w", err)
		}
		value, err := base64.StdEncoding.DecodeString(attr.Value)
		if err != nil {
			return fmt.Errorf("could not decode event attribute value: %w", err)
		}

		if string(key) == "receiver" {
			receiverAddr = string(value)
		}
		if string(key) == "amount" {
			receiverAmount = string(value)
		}
	}

	if receiverAddr != "" && receiverAmount != "" {
		coins, err := sdk.ParseCoinsNormalized(receiverAmount)
		if err != nil {
			return err
		}
		sf.ReceiverAddress = receiverAddr
		sf.RewardsReceived = coins
	} else {
		return errors.New("rewards received or address were not present")
	}

	return nil
}

func (sf *WrapperBlockDistribution) ParseRelevantData() []events.EventRelevantInformation {
	relevantData := make([]events.EventRelevantInformation, len(sf.RewardsReceived))

	for i, coin := range sf.RewardsReceived {
		relevantData[i] = events.EventRelevantInformation{
			Address:      sf.ReceiverAddress,
			Amount:       coin.Amount.BigInt(),
			Denomination: coin.Denom,
			EventSource:  dbTypes.OsmosisRewardDistribution,
		}
	}

	return relevantData
}

func (sf *WrapperBlockDistribution) String() string {
	return fmt.Sprintf("Osmosis Incentives event %s: Address %s received %s rewards.", sf.GetType(), sf.ReceiverAddress, sf.RewardsReceived)
}
