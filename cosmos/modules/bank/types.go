package bank

import (
	"fmt"

	parsingTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules"
	txModule "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

var IsMsgSend = map[string]bool{
	"MsgSend":                      true,
	"/cosmos.bank.v1beta1.MsgSend": true,
}

//HandleMsg: Unmarshal JSON for MsgSend.
//Note that MsgSend ignores the TxLogMessage because it isn't needed.
func (sf *WrapperMsgSend) HandleMsg(msgType string, msg sdk.Msg, log *txModule.TxLogMessage) error {
	sf.Type = msgType
	sf.CosmosMsgSend = msg.(*bankTypes.MsgSend)

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
		currRelevantData.AmountSent = v.Amount.BigInt()
		currRelevantData.DenominationSent = v.Denom

		relevantData[i] = currRelevantData
	}

	return relevantData
}

type WrapperMsgSend struct {
	txModule.Message
	CosmosMsgSend *bankTypes.MsgSend
}
