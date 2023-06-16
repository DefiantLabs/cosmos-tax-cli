package protorev

import (
	"encoding/base64"
	"fmt"

	abciTypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/DefiantLabs/cosmos-indexer/cosmos/events"
	dbTypes "github.com/DefiantLabs/cosmos-indexer/db"
	osmosisEvents "github.com/DefiantLabs/cosmos-indexer/osmosis/events"
)

// protorevDeveloperAddress is the address of the developer account that receives rewards on the weekly Epoch.
// This will need to be prepopulated by the indexer before module startup
var protorevDeveloperAddress string

func SetDeveloperAddress(address string) {
	protorevDeveloperAddress = address
}

type WrapperBlockCoinReceived struct {
	Event           abciTypes.Event
	RewardsReceived sdk.Coins
	ReceiverAddress string
	ProtorevEvent   bool
}

func (sf *WrapperBlockCoinReceived) GetType() string {
	return osmosisEvents.BlockEventDistribution
}

func (sf *WrapperBlockCoinReceived) HandleEvent(eventType string, event abciTypes.Event) error {
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

	if receiverAddr != "" && receiverAmount != "" && receiverAddr == protorevDeveloperAddress {
		coins, err := sdk.ParseCoinsNormalized(receiverAmount)
		if err != nil {
			return err
		}
		sf.ReceiverAddress = receiverAddr
		sf.RewardsReceived = coins
		sf.ProtorevEvent = true
	} else {
		sf.ProtorevEvent = false
	}

	return nil
}

func (sf *WrapperBlockCoinReceived) ParseRelevantData() []events.EventRelevantInformation {
	if !sf.ProtorevEvent {
		return nil
	}

	relevantData := make([]events.EventRelevantInformation, len(sf.RewardsReceived))

	for i, coin := range sf.RewardsReceived {
		relevantData[i] = events.EventRelevantInformation{
			Address:      sf.ReceiverAddress,
			Amount:       coin.Amount.BigInt(),
			Denomination: coin.Denom,
			EventSource:  dbTypes.OsmosisProtorevDeveloperRewardDistribution,
		}
	}

	return relevantData
}

func (sf *WrapperBlockCoinReceived) String() string {
	if !sf.ProtorevEvent {
		return "Coin received event is not a Protorev event"
	}
	return fmt.Sprintf("Osmosis Protorev event %s: Address %s received %s rewards.", sf.GetType(), sf.ReceiverAddress, sf.RewardsReceived)
}
