package wasm

import (
	"fmt"

	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/DefiantLabs/cosmos-tax-cli/config"
	parsingTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules"
	txTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	MsgExecuteContract                 = "/cosmwasm.wasm.v1.MsgExecuteContract"
	MsgInstantiateContract             = "/cosmwasm.wasm.v1.MsgInstantiateContract"
	MsgInstantiateContract2            = "/cosmwasm.wasm.v1.MsgInstantiateContract2"
	MsgStoreCode                       = "/cosmwasm.wasm.v1.MsgStoreCode"
	MsgMigrateContract                 = "/cosmwasm.wasm.v1.MsgMigrateContract"
	MsgUpdateAdmin                     = "/cosmwasm.wasm.v1.MsgUpdateAdmin"
	MsgClearAdmin                      = "/cosmwasm.wasm.v1.MsgClearAdmin"
	MsgUpdateInstantiationAdmin        = "/cosmwasm.wasm.v1.MsgUpdateInstantiationAdmin"
	MsgUpdateParams                    = "/cosmwasm.wasm.v1.MsgUpdateParams"
	MsgSudoContract                    = "/cosmwasm.wasm.v1.MsgSudoContract"
	MsgPinCodes                        = "/cosmwasm.wasm.v1.MsgPinCodes"
	MsgUnpinCodes                      = "/cosmwasm.wasm.v1.MsgUnpinCodes"
	MsgStoreAndInstantiateContract     = "/cosmwasm.wasm.v1.MsgStoreAndInstantiateContract"
	MsgRemoveCodeUploadParamsAddresses = "/cosmwasm.wasm.v1.MsgRemoveCodeUploadParamsAddresses"
	MsgAddCodeUploadParamsAddresses    = "/cosmwasm.wasm.v1.MsgAddCodeUploadParamsAddresses"
	MsgStoreAndMigrateContract         = "/cosmwasm.wasm.v1.MsgStoreAndMigrateContract"
)

type ContractExecutionMessageHandler interface {
	ContractFriendlyName() string
	TopLevelFieldIdentifiers() []string
	TopLevelIdentifierType() any
	HandlerFuncs() []func() ContractExectionParserHandler
}

type ContractExectionParserHandler interface {
	txTypes.CosmosMessage
	SetCosmosMsgExecuteContract(*wasmTypes.MsgExecuteContract)
}

type ContractExecutionMessageHandlerByCodeID interface {
	ContractExecutionMessageHandler
	CodeID() uint64
}

type ContractExecutionMessageHandlerByContractAddress interface {
	ContractExecutionMessageHandler
	ContractAddress() string
}

type WrapperMsgExecuteContract struct {
	txTypes.Message
	CosmosMsgExecuteContract *wasmTypes.MsgExecuteContract
	ContractAddressRegistry  map[string]ContractExecutionMessageHandler
	CurrentHandler           ContractExecutionMessageHandler
	CurrentCosmosMessage     txTypes.CosmosMessage
	ContractAddress          string
}

func (w *WrapperMsgExecuteContract) HandleMsg(typeURL string, msg sdk.Msg, log *txTypes.LogMessage) error {
	w.Type = typeURL
	w.CosmosMsgExecuteContract = msg.(*wasmTypes.MsgExecuteContract)

	if handler, ok := w.ContractAddressRegistry[w.CosmosMsgExecuteContract.Contract]; ok {
		w.CurrentHandler = handler
		var handled bool
		var handledCosmosMessage txTypes.CosmosMessage
		for _, handlerFunc := range w.CurrentHandler.HandlerFuncs() {
			cosmosMessage := handlerFunc()
			// Allows us to avoid parsing twice
			cosmosMessage.SetCosmosMsgExecuteContract(w.CosmosMsgExecuteContract)
			err := cosmosMessage.HandleMsg(typeURL, msg, log)
			if err != nil {
				config.Log.Debug(fmt.Sprintf("Handler entry failed for contract address %s", w.CosmosMsgExecuteContract.Contract), err)
				continue
			}

			handled = true
			handledCosmosMessage = cosmosMessage
			break
		}

		if handled {
			w.CurrentCosmosMessage = handledCosmosMessage
		} else {
			return fmt.Errorf("no handler succeeded found for contract address %s using handler %s", w.CosmosMsgExecuteContract.Contract, w.CurrentHandler.ContractFriendlyName())
		}

		return nil
	}

	return nil
}

func (w *WrapperMsgExecuteContract) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	if w.CurrentCosmosMessage != nil {
		return w.CurrentCosmosMessage.ParseRelevantData()
	}

	return nil
}

func (w *WrapperMsgExecuteContract) GetType() string {
	return MsgExecuteContract
}

func (w *WrapperMsgExecuteContract) String() string {
	if w.CurrentCosmosMessage != nil {
		return w.CurrentCosmosMessage.String()
	}
	return fmt.Sprintf("MsgExecuteContract: No handler found for contract address %s", w.ContractAddress)
}
