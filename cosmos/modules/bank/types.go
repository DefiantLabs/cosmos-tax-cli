package bank

import (
	"fmt"
	"strings"

	parsingTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules"
	txModule "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

var IsMsgSend = map[string]bool{
	"MsgSend":                      true,
	"/cosmos.bank.v1beta1.MsgSend": true,
}

var IsMsgMultiSend = map[string]bool{
	"MsgMultiSend":                      true,
	"/cosmos.bank.v1beta1.MsgMultiSend": true,
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

func (sf *WrapperMsgMultiSend) HandleMsg(msgType string, msg sdk.Msg, log *txModule.TxLogMessage) error {
	sf.Type = msgType
	sf.CosmosMsgMultiSend = msg.(*bankTypes.MsgMultiSend)

	//Make sure the standard ordering of Inputs -> Outputs applies (where send Input[i] == received Output[i])
	//This is assuming Inputs[i] corresponds to Outputs[i]
	//Is this safe to assume? From testing it looks like it
	for i, input := range sf.CosmosMsgMultiSend.Inputs {
		correspondingOutput := sf.CosmosMsgMultiSend.Outputs[i]

		for ii, coinSent := range input.Coins {
			correspondingCoin := correspondingOutput.Coins[ii]

			if !correspondingCoin.IsEqual(coinSent) {
				return fmt.Errorf("Error processing MultiSend, inputs and outputs mismatch, send %s != received %s in standard ordering", coinSent, correspondingCoin)
			} else {
				sf.SenderReceiverAmounts = append(sf.SenderReceiverAmounts, SenderReceiverAmount{Sender: input.Address, Receiver: correspondingOutput.Address, Amount: coinSent})
			}
		}
	}

	return nil
}

func (sf *WrapperMsgSend) String() string {
	return fmt.Sprintf("MsgSend: Address %s received %s from %s \n",
		sf.CosmosMsgSend.ToAddress, sf.CosmosMsgSend.Amount, sf.CosmosMsgSend.FromAddress)
}

func (sf *WrapperMsgMultiSend) String() string {
	var sendsAndReceives []string
	for _, v := range sf.SenderReceiverAmounts {
		sendsAndReceives = append(sendsAndReceives, fmt.Sprintf("%s %s -> %s", v.Amount, v.Sender, v.Receiver))
	}
	return fmt.Sprintf("MsgMultiSend: %s\n", strings.Join(sendsAndReceives, ", "))
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

		//This is required since we do CSV parsing on the receiver here too
		currRelevantData.AmountReceived = v.Amount.BigInt()
		currRelevantData.DenominationReceived = v.Denom

		relevantData[i] = currRelevantData
	}

	return relevantData
}

func (sf *WrapperMsgMultiSend) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	var relevantData []parsingTypes.MessageRelevantInformation

	for _, senderReceiverAmount := range sf.SenderReceiverAmounts {

		var currRelevantData parsingTypes.MessageRelevantInformation
		currRelevantData.SenderAddress = senderReceiverAmount.Sender
		currRelevantData.ReceiverAddress = senderReceiverAmount.Receiver

		currRelevantData.AmountSent = senderReceiverAmount.Amount.Amount.BigInt()
		currRelevantData.DenominationSent = senderReceiverAmount.Amount.Denom

		currRelevantData.AmountReceived = senderReceiverAmount.Amount.Amount.BigInt()
		currRelevantData.DenominationReceived = senderReceiverAmount.Amount.Denom

		relevantData = append(relevantData, currRelevantData)
	}

	return relevantData
}

type WrapperMsgSend struct {
	txModule.Message
	CosmosMsgSend *bankTypes.MsgSend
}

type WrapperMsgMultiSend struct {
	txModule.Message
	CosmosMsgMultiSend    *bankTypes.MsgMultiSend
	SenderReceiverAmounts []SenderReceiverAmount
}

type SenderReceiverAmount struct {
	Sender   string
	Receiver string
	Amount   sdk.Coin
}
