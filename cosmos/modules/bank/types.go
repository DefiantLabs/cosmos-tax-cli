package bank

import (
	"encoding/json"
	"fmt"

	parsingTypes "cosmos-exporter/cosmos/modules"
	txModule "cosmos-exporter/cosmos/modules/tx"

	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

var IsMsgSend = map[string]bool{
	"MsgSend":                      true,
	"/cosmos.bank.v1beta1.MsgSend": true,
}

//CosmUnmarshal(): Unmarshal JSON for MsgSend.
//Note that MsgSend ignores the TxLogMessage because it isn't needed.
func (sf *WrapperMsgSend) CosmUnmarshal(msgType string, raw []byte, log *txModule.TxLogMessage) error {
	sf.Type = msgType
	if err := json.Unmarshal(raw, &sf.CosmosMsgSend); err != nil {
		fmt.Println("Error parsing message: " + err.Error())
		return err
	}

	//Confirm that the action listed in the message log matches the Message type
	valid_log := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !valid_log {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	//The attribute in the log message that shows you the delegator withdrawal address and amount received
	receivedCoinsEvt := txModule.GetEventWithType(bankTypes.EventTypeCoinReceived, log)
	if receivedCoinsEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	receiver_address := txModule.GetValueForAttribute(bankTypes.AttributeKeyReceiver, receivedCoinsEvt)
	//coins_received := txModule.GetValueForAttribute("amount", receivedCoinsEvt)

	if sf.CosmosMsgSend.ToAddress != receiver_address {
		return fmt.Errorf("transaction receiver address %s does not match log event '%s' receiver address %s",
			sf.CosmosMsgSend.ToAddress, bankTypes.EventTypeCoinReceived, receiver_address)
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
	txModule.Message
	CosmosMsgSend bankTypes.MsgSend
}
