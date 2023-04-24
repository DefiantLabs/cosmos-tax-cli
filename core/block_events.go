package core

import (
	"errors"
	"fmt"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
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

func ProcessRPCBlockEvents(blockResults *ctypes.ResultBlockResults) ([]eventTypes.EventRelevantInformation, error) {
	var taxableEvents []eventTypes.EventRelevantInformation
	if len(endBlockerEventTypeHandlers) != 0 {
		for _, event := range blockResults.EndBlockEvents {
			handlers, handlersFound := endBlockerEventTypeHandlers[event.Type]

			if !handlersFound {
				continue
			}

			var err error = nil
			for _, handler := range handlers {
				cosmosEventHandler := handler()
				err = cosmosEventHandler.HandleEvent(event.Type, event)
				if err != nil {
					config.Log.Debug(fmt.Sprintf("[Block: %v] Cosmos Block EndBlocker event of known type: %s. Handler failed", blockResults.Height, cosmosEventHandler), err)
					continue
				}
				var relevantData = cosmosEventHandler.ParseRelevantData()

				taxableEvents = append(taxableEvents, relevantData...)

				config.Log.Debug(fmt.Sprintf("[Block: %v] Cosmos Block EndBlocker event of known type: %s", blockResults.Height, cosmosEventHandler))
				break
			}

			// If err is not nil here, all handlers failed
			if err != nil {
				return nil, errors.New(fmt.Sprintf("Could not handle event type %s, all handlers failed", event.Type))
			}
		}
	}

	return taxableEvents, nil
}
