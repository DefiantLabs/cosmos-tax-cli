package staking

import (
	txModule "cosmos-exporter/cosmos/modules/tx"
	"encoding/json"
	"fmt"

	parsingTypes "cosmos-exporter/cosmos/modules"

	stdTypes "github.com/cosmos/cosmos-sdk/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distTypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

var IsMsgWithdrawValidatorCommission = map[string]bool{
	"/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission": true,
	"withdraw-rewards": true, //NOTE/TODO: not 100% sure if this is only on delegator or validator withdrawal...
}

var IsMsgWithdrawDelegatorReward = map[string]bool{
	"/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward": true,
	"withdraw-rewards": true, //NOTE/TODO: not 100% sure if this is only on delegator or validator withdrawal...
}

type WrapperMsgWithdrawValidatorCommission struct {
	txModule.Message
	CosmosMsgWithdrawValidatorCommission distTypes.MsgWithdrawValidatorCommission
	DelegatorReceiverAddress             string
	CoinsReceived                        stdTypes.Coin
	MultiCoinsReceived                   stdTypes.Coins
}

type WrapperMsgWithdrawDelegatorReward struct {
	txModule.Message
	CosmosMsgWithdrawDelegatorReward distTypes.MsgWithdrawDelegatorReward
	CoinsReceived                    stdTypes.Coin
	MultiCoinsReceived               stdTypes.Coins
}

func (sf *WrapperMsgWithdrawDelegatorReward) String() string {
	var coinsReceivedString string
	if !sf.CoinsReceived.IsNil() {
		coinsReceivedString = sf.CoinsReceived.String()
	} else {
		coinsReceivedString = sf.MultiCoinsReceived.String()
	}

	return fmt.Sprintf("MsgWithdrawDelegatorReward: Delegator %s received %s\n",
		sf.CosmosMsgWithdrawDelegatorReward.DelegatorAddress, coinsReceivedString)
}

func (sf *WrapperMsgWithdrawValidatorCommission) String() string {

	var coinsReceivedString string
	if !sf.CoinsReceived.IsNil() {
		coinsReceivedString = sf.CoinsReceived.String()
	} else {
		coinsReceivedString = sf.MultiCoinsReceived.String()
	}

	return fmt.Sprintf("WrapperMsgWithdrawValidatorCommission: Validator %s commission withdrawn. Delegator %s received %s\n",
		sf.CosmosMsgWithdrawValidatorCommission.ValidatorAddress, sf.DelegatorReceiverAddress, coinsReceivedString)
}

//CosmUnmarshal(): Unmarshal JSON for MsgWithdrawDelegatorReward
func (sf *WrapperMsgWithdrawValidatorCommission) CosmUnmarshal(msgType string, raw []byte, log *txModule.TxLogMessage) error {
	sf.Type = msgType
	if err := json.Unmarshal(raw, &sf.CosmosMsgWithdrawValidatorCommission); err != nil {
		fmt.Println("Error parsing message: " + err.Error())
		return err
	}

	//Confirm that the action listed in the message log matches the Message type
	valid_log := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !valid_log {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	//The attribute in the log message that shows you the delegator withdrawal address and amount received
	delegatorReceivedCoinsEvt := txModule.GetEventWithType(bankTypes.EventTypeCoinReceived, log)
	if delegatorReceivedCoinsEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	sf.DelegatorReceiverAddress = txModule.GetValueForAttribute(bankTypes.AttributeKeyReceiver, delegatorReceivedCoinsEvt)
	coins_received := txModule.GetValueForAttribute("amount", delegatorReceivedCoinsEvt)

	coin, err := stdTypes.ParseCoinNormalized(coins_received)
	if err != nil {
		coins, err := stdTypes.ParseCoinsNormalized(coins_received)
		if err != nil {
			fmt.Println("Error parsing coins normalized")
			fmt.Println(err)
			return err
		}
		sf.MultiCoinsReceived = coins
	} else {
		sf.CoinsReceived = coin
	}

	return err
}

func (sf *WrapperMsgWithdrawValidatorCommission) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	if sf.CoinsReceived.IsNil() {
		var relevantData []parsingTypes.MessageRelevantInformation = make([]parsingTypes.MessageRelevantInformation, len(sf.MultiCoinsReceived))

		for i, v := range sf.MultiCoinsReceived {
			relevantData[i] = parsingTypes.MessageRelevantInformation{
				Amount:          float64(v.Amount.Int64()),
				Denomination:    v.Denom,
				SenderAddress:   "",
				ReceiverAddress: sf.DelegatorReceiverAddress,
			}
		}

		return relevantData
	} else {
		var relevantData []parsingTypes.MessageRelevantInformation = make([]parsingTypes.MessageRelevantInformation, 1)
		relevantData[0] = parsingTypes.MessageRelevantInformation{
			Amount:          float64(sf.CoinsReceived.Amount.Int64()),
			Denomination:    sf.CoinsReceived.Denom,
			SenderAddress:   "",
			ReceiverAddress: sf.DelegatorReceiverAddress,
		}
		return relevantData
	}
}

//CosmUnmarshal(): Unmarshal JSON for MsgWithdrawDelegatorReward
func (sf *WrapperMsgWithdrawDelegatorReward) CosmUnmarshal(msgType string, raw []byte, log *txModule.TxLogMessage) error {
	sf.Type = msgType
	if err := json.Unmarshal(raw, &sf.CosmosMsgWithdrawDelegatorReward); err != nil {
		fmt.Println("Error parsing message: " + err.Error())
		return err
	}

	//Confirm that the action listed in the message log matches the Message type
	valid_log := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !valid_log {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	//The attribute in the log message that shows you the delegator withdrawal address and amount received
	delegatorReceivedCoinsEvt := txModule.GetEventWithType(bankTypes.EventTypeCoinReceived, log)
	if delegatorReceivedCoinsEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	delegator_address := txModule.GetValueForAttribute(bankTypes.AttributeKeyReceiver, delegatorReceivedCoinsEvt)
	coins_received := txModule.GetValueForAttribute("amount", delegatorReceivedCoinsEvt)

	//This may be able to be optimized by doing one or the other
	coin, err := stdTypes.ParseCoinNormalized(coins_received)
	if err != nil {
		coins, err := stdTypes.ParseCoinsNormalized(coins_received)
		if err != nil {
			fmt.Println("Error parsing coins normalized")
			fmt.Println(err)
			return err
		}
		sf.MultiCoinsReceived = coins
	} else {
		sf.CoinsReceived = coin
	}
	if sf.CosmosMsgWithdrawDelegatorReward.DelegatorAddress != delegator_address {
		return fmt.Errorf("transaction delegator address %s does not match log event '%s' delegator address %s",
			sf.CosmosMsgWithdrawDelegatorReward.DelegatorAddress, bankTypes.EventTypeCoinReceived, delegator_address)
	}

	return err
}

func (sf *WrapperMsgWithdrawDelegatorReward) ParseRelevantData() []parsingTypes.MessageRelevantInformation {

	if sf.CoinsReceived.IsNil() {
		var relevantData []parsingTypes.MessageRelevantInformation = make([]parsingTypes.MessageRelevantInformation, len(sf.MultiCoinsReceived))

		for i, v := range sf.MultiCoinsReceived {
			relevantData[i] = parsingTypes.MessageRelevantInformation{
				Amount:          float64(v.Amount.Int64()),
				Denomination:    v.Denom,
				SenderAddress:   "",
				ReceiverAddress: sf.CosmosMsgWithdrawDelegatorReward.DelegatorAddress,
			}
		}

		return relevantData
	} else {
		var relevantData []parsingTypes.MessageRelevantInformation = make([]parsingTypes.MessageRelevantInformation, 1)
		relevantData[0] = parsingTypes.MessageRelevantInformation{
			Amount:          float64(sf.CoinsReceived.Amount.Int64()),
			Denomination:    sf.CoinsReceived.Denom,
			SenderAddress:   "",
			ReceiverAddress: sf.CosmosMsgWithdrawDelegatorReward.DelegatorAddress,
		}
		return relevantData
	}

}
