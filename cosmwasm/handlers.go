package cosmwasm

import (
	txTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmwasm/modules/wasm"
	"github.com/DefiantLabs/cosmos-tax-cli/rpc"
	"github.com/DefiantLabs/lens/client"
)

var defaultMessageTypeHandler = map[string][]func() txTypes.CosmosMessage{}

var contractAddressRegistry = map[string]wasm.ContractExecutionMessageHandler{}

func GetCosmWasmMessageTypeHandlers(customContractAddressHandlers []wasm.ContractExecutionMessageHandler, lensClient *client.ChainClient) (map[string][]func() txTypes.CosmosMessage, error) {
	msgExecuteContractHandlers, err := configureMsgExecuteContractHandler(customContractAddressHandlers, lensClient)
	if err != nil {
		return nil, err
	}

	defaultMessageTypeHandler[wasm.MsgExecuteContract] = msgExecuteContractHandlers

	return defaultMessageTypeHandler, nil
}

// Configures a handler wrapper that will allow using registry values to find custom message handlers
func configureMsgExecuteContractHandler(customContractAddressHandlers []wasm.ContractExecutionMessageHandler, lensClient *client.ChainClient) ([]func() txTypes.CosmosMessage, error) {
	for _, handler := range customContractAddressHandlers {
		if castHandler, ok := handler.(wasm.ContractExecutionMessageHandlerByContractAddress); ok {
			contractAddressRegistry[castHandler.ContractAddress()] = handler
		} else if castHandler, ok := handler.(wasm.ContractExecutionMessageHandlerByCodeID); ok {
			// Query for code ID contract addresses using wasm querying
			resp, err := rpc.GetContractsByCodeIDAtHeight(lensClient, castHandler.CodeID(), 0)
			if err != nil {
				return nil, err
			}

			for _, contractAddress := range resp.Contracts {
				contractAddressRegistry[contractAddress] = handler
			}
		}
	}

	configuredExecuteContractWrapper := wasm.WrapperMsgExecuteContract{
		ContractAddressRegistry: contractAddressRegistry,
	}

	return []func() txTypes.CosmosMessage{
		func() txTypes.CosmosMessage {
			return configuredExecuteContractWrapper
		},
	}, nil
}
