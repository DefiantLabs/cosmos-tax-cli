package core

import (
	"fmt"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	eventTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/events"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmoshub"
	cosmoshubTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmoshub"
	dbTypes "github.com/DefiantLabs/cosmos-tax-cli/db"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

// TODO: Implement beginBlockEventTypeHandlers when ready
var endBlockerEventTypeHandlers = map[string][]func() eventTypes.CosmosEvent{}

func ChainSpecificEndBlockerEventTypeHandlerBootstrap(chainID string) {
	var chainSpecificEndBlockerEventTypeHandler map[string][]func() eventTypes.CosmosEvent
	if chainID == cosmoshub.ChainID {
		chainSpecificEndBlockerEventTypeHandler = cosmoshubTypes.EndBlockerEventTypeHandlers
	}
	for key, value := range chainSpecificEndBlockerEventTypeHandler {
		if list, ok := endBlockerEventTypeHandlers[key]; ok {
			endBlockerEventTypeHandlers[key] = append(value, list...)
		} else {
			endBlockerEventTypeHandlers[key] = value
		}
	}
}

func ProcessRPCBlockEvents(blockResults *ctypes.ResultBlockResults) []dbTypes.TaxableEvent {
	if len(endBlockerEventTypeHandlers) != 0 {
		for _, event := range blockResults.EndBlockEvents {
			handlers, ok := endBlockerEventTypeHandlers[event.Type]

			if !ok {
				continue
			}

			for _, handler := range handlers {
				cosmosEventHandler := handler()
				cosmosEventHandler.HandleEvent(event.Type, event)
				var relevantData = cosmosEventHandler.ParseRelevantData()

				for _, data := range relevantData {
					fmt.Println(data)
				}

				config.Log.Debug(fmt.Sprintf("[Block: %v] Cosmos event of known type: %s", blockResults.Height, cosmosEventHandler))
			}
		}
	}

	return nil
}
