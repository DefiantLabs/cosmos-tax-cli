package bank

import (
	parsingTypes "cosmos-exporter/cosmos/modules"
	tx "cosmos-exporter/cosmos/modules/tx"
	"encoding/json"
	"fmt"

	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

//CosmUnmarshal(): Unmarshal JSON for MsgSend.
//Note that MsgSend ignores the TxLogMessage because it isn't needed.
func (sf *WrapperMsgSend) CosmUnmarshal(msgType string, raw []byte, log *tx.TxLogMessage) error {
	sf.Type = msgType
	if err := json.Unmarshal(raw, &sf.CosmosMsgSend); err != nil {
		fmt.Println("Error parsing message: " + err.Error())
		return err
	}

	return nil
}

func (sf *WrapperMsgSend) String() string {
	return fmt.Sprintf("MsgSend: Address %s received %s from %s \n",
		sf.CosmosMsgSend.ToAddress, sf.CosmosMsgSend.Amount, sf.CosmosMsgSend.FromAddress)
}

func (sf *WrapperMsgSend) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	var relevantData []parsingTypes.MessageRelevantInformation = make([]parsingTypes.MessageRelevantInformation, len(sf.CosmosMsgSend.Amount))

	for i, v := range sf.CosmosMsgSend.Amount {
		var currRelevantData parsingTypes.MessageRelevantInformation
		currRelevantData.SenderAddress = sf.CosmosMsgSend.FromAddress
		currRelevantData.ReceiverAddress = sf.CosmosMsgSend.ToAddress
		//Amount always seems to be an integer, float may be an extra uneeded step
		currRelevantData.Amount = float64(v.Amount.Int64())
		currRelevantData.Denomination = v.Denom

		relevantData[i] = currRelevantData
	}

	return relevantData
}

type WrapperMsgSend struct {
	tx.Message
	CosmosMsgSend bankTypes.MsgSend
}
