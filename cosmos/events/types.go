package events

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type CosmosEvent interface {
	HandleEvent(string, sdk.Event) error
	ParseRelevantData()
	GetType() string
	String() string
}
