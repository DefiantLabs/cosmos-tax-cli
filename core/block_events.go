package core

import (
	"fmt"

	eventTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/events"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmoshub"
	cosmoshubTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmoshub"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

// TODO: Implement beginBlockEventTypeHandlers when ready
var endBlockerEventTypeHandlers = map[string][]func() eventTypes.CosmosEvent{}

func ChainSpecificEndBlockerEventTypeHandlerBootstrap(chainID string) {
	var chainSpecificEndBlockerEventTypeHandler map[string][]func() eventTypes.CosmosEvent
	if chainID == cosmoshub.ChainID {
		fmt.Println("Bootstrapping end blocker event type handlers for cosmoshub")
		chainSpecificEndBlockerEventTypeHandler = cosmoshubTypes.EndBlockerEventTypeHandlers
	}
	for key, value := range chainSpecificEndBlockerEventTypeHandler {
		if list, ok := endBlockerEventTypeHandlers[key]; ok {
			endBlockerEventTypeHandlers[key] = append(value, list...)
		} else {
			endBlockerEventTypeHandlers[key] = value
		}
	}
	fmt.Printf("%+v\n", endBlockerEventTypeHandlers)

}

func ProcessRPCBlockByHeightEvents(blockResults *ctypes.ResultBlockResults) {
	if len(endBlockerEventTypeHandlers) != 0 {
		for _, event := range blockResults.EndBlockEvents {
			handlers, ok := endBlockerEventTypeHandlers[event.Type]

			if !ok {
				continue
			}

			for range handlers {
				fmt.Println("Handling", event.Type)
			}
		}
	}
}
