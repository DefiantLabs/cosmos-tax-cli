package staking

import (
	"fmt"
	"strings"

	txModule "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"

	parsingTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules"
	stdTypes "github.com/cosmos/cosmos-sdk/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakeTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

var IsMsgDelegate = map[string]bool{
	"/cosmos.staking.v1beta1.MsgDelegate": true,
}

var IsMsgUndelegate = map[string]bool{
	"/cosmos.staking.v1beta1.MsgUndelegate": true,
}

var IsMsgBeginRedelegate = map[string]bool{
	"/cosmos.staking.v1beta1.MsgBeginRedelegate": true,
}

type WrapperMsgDelegate struct {
	txModule.Message
	CosmosMsgDelegate    *stakeTypes.MsgDelegate
	DelegatorAddress     string
	AutoWithdrawalReward *stdTypes.Coin
}

type WrapperMsgUndelegate struct {
	txModule.Message
	CosmosMsgUndelegate  *stakeTypes.MsgUndelegate
	DelegatorAddress     string
	AutoWithdrawalReward *stdTypes.Coin
}

type WrapperMsgBeginRedelegate struct {
	txModule.Message
	CosmosMsgBeginRedelegate *stakeTypes.MsgBeginRedelegate
	DelegatorAddress         string
	AutoWithdrawalRewards    stdTypes.Coins
}

// HandleMsg: Handle type checking for MsgFundCommunityPool
func (sf *WrapperMsgDelegate) HandleMsg(msgType string, msg stdTypes.Msg, log *txModule.TxLogMessage) error {
	sf.Type = msgType
	sf.CosmosMsgDelegate = msg.(*stakeTypes.MsgDelegate)

	//Confirm that the action listed in the message log matches the Message type
	valid_log := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !valid_log {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	//The attribute in the log message that shows you the delegator rewards auto-received
	delegatorReceivedCoinsEvt := txModule.GetEventWithType(bankTypes.EventTypeTransfer, log)
	if delegatorReceivedCoinsEvt == nil {
		sf.AutoWithdrawalReward = nil
		sf.DelegatorAddress = sf.CosmosMsgDelegate.DelegatorAddress
	} else {
		coins_received := txModule.GetValueForAttribute("amount", delegatorReceivedCoinsEvt)
		coin, err := stdTypes.ParseCoinNormalized(coins_received)
		if err == nil {
			sf.AutoWithdrawalReward = &coin
		} else {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
		}
		sf.DelegatorAddress = txModule.GetValueForAttribute("recipient", delegatorReceivedCoinsEvt)
	}

	return nil
}

func (sf *WrapperMsgUndelegate) HandleMsg(msgType string, msg stdTypes.Msg, log *txModule.TxLogMessage) error {
	sf.Type = msgType
	sf.CosmosMsgUndelegate = msg.(*stakeTypes.MsgUndelegate)

	//Confirm that the action listed in the message log matches the Message type
	valid_log := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !valid_log {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	//The attribute in the log message that shows you the delegator rewards auto-received
	sf.DelegatorAddress = sf.CosmosMsgUndelegate.DelegatorAddress
	delegatorReceivedCoinsEvt := txModule.GetEventWithType(bankTypes.EventTypeCoinReceived, log)
	if delegatorReceivedCoinsEvt == nil {
		sf.AutoWithdrawalReward = nil
		sf.DelegatorAddress = sf.CosmosMsgUndelegate.DelegatorAddress
	} else {
		var receivers []string
		var amounts []string

		//Pair off amounts and receivers in order
		for _, v := range delegatorReceivedCoinsEvt.Attributes {
			if v.Key == "receiver" {
				receivers = append(receivers, v.Value)
			} else if v.Key == "amount" {
				amounts = append(amounts, v.Value)
			}
		}

		//Find delegator address in receivers if its there, find its paired amount and set as the withdrawn rewards
		for i, v := range receivers {
			if v == sf.DelegatorAddress {
				coin, err := stdTypes.ParseCoinNormalized(amounts[i])
				if err == nil {
					sf.AutoWithdrawalReward = &coin
				} else {
					return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
				}
			}
		}
	}

	return nil
}

// HandleMsg: Handle type checking for MsgFundCommunityPool
func (sf *WrapperMsgBeginRedelegate) HandleMsg(msgType string, msg stdTypes.Msg, log *txModule.TxLogMessage) error {
	sf.Type = msgType
	sf.CosmosMsgBeginRedelegate = msg.(*stakeTypes.MsgBeginRedelegate)

	//Confirm that the action listed in the message log matches the Message type
	valid_log := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !valid_log {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	//The attribute in the log message that shows you the delegator rewards auto-received
	delegatorReceivedCoinsEvt := txModule.GetEventWithType(bankTypes.EventTypeCoinReceived, log)
	sf.DelegatorAddress = sf.CosmosMsgBeginRedelegate.DelegatorAddress
	if delegatorReceivedCoinsEvt == nil {
		sf.AutoWithdrawalRewards = make(stdTypes.Coins, 0)
	} else {
		var receivers []string
		var amounts []string

		//Pair off amounts and receivers in order
		for _, v := range delegatorReceivedCoinsEvt.Attributes {
			if v.Key == "receiver" {
				receivers = append(receivers, v.Value)
			} else if v.Key == "amount" {
				amounts = append(amounts, v.Value)
			}
		}

		//Find delegator address in receivers if its there, find its paired amount and set as the withdrawn rewards
		//We use a cosmos.Coins array type for redelegations as redelegating could force withdrawal from both validators
		for i, v := range receivers {
			if v == sf.DelegatorAddress {
				coin, err := stdTypes.ParseCoinNormalized(amounts[i])
				if err == nil {
					sf.AutoWithdrawalRewards = append(sf.AutoWithdrawalRewards, coin)
				} else {
					return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
				}
			}
		}
	}

	return nil
}

func (sf *WrapperMsgDelegate) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	var relevantData []parsingTypes.MessageRelevantInformation
	if sf.AutoWithdrawalReward != nil {
		data := parsingTypes.MessageRelevantInformation{}
		data.AmountReceived = sf.AutoWithdrawalReward.Amount.BigInt()
		data.DenominationReceived = sf.AutoWithdrawalReward.Denom
		data.ReceiverAddress = sf.DelegatorAddress
		relevantData = append(relevantData, data)
	}
	return relevantData
}

func (sf *WrapperMsgUndelegate) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	var relevantData []parsingTypes.MessageRelevantInformation
	if sf.AutoWithdrawalReward != nil {
		data := parsingTypes.MessageRelevantInformation{}
		data.AmountReceived = sf.AutoWithdrawalReward.Amount.BigInt()
		data.DenominationReceived = sf.AutoWithdrawalReward.Denom
		data.ReceiverAddress = sf.DelegatorAddress
		relevantData = append(relevantData, data)
	}
	return relevantData
}

func (sf *WrapperMsgBeginRedelegate) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	var relevantData []parsingTypes.MessageRelevantInformation
	for _, coin := range sf.AutoWithdrawalRewards {
		data := parsingTypes.MessageRelevantInformation{}
		data.AmountReceived = coin.Amount.BigInt()
		data.DenominationReceived = coin.Denom
		data.ReceiverAddress = sf.DelegatorAddress
		relevantData = append(relevantData, data)
	}
	return relevantData
}

func (sf *WrapperMsgDelegate) String() string {
	if sf.AutoWithdrawalReward == nil {
		return fmt.Sprintf("MsgDelegate: Delegator %s did not auto-withdrawal rewards\n", sf.DelegatorAddress)
	}
	return fmt.Sprintf("MsgDelegate: Delegator %s auto-withdrew %s\n", sf.DelegatorAddress, sf.AutoWithdrawalReward)
}

func (sf *WrapperMsgUndelegate) String() string {
	if sf.AutoWithdrawalReward == nil {
		return fmt.Sprintf("MsgUndelegate: Delegator %s did not auto-withdrawal rewards\n", sf.DelegatorAddress)
	}
	return fmt.Sprintf("MsgUndelegate: Delegator %s auto-withdrew %s\n", sf.DelegatorAddress, sf.AutoWithdrawalReward)
}

func (sf *WrapperMsgBeginRedelegate) String() string {

	var coinsRecievedStrings []string
	for _, coin := range sf.AutoWithdrawalRewards {
		coinsRecievedStrings = append(coinsRecievedStrings, coin.String())
	}

	if len(coinsRecievedStrings) > 0 {
		return fmt.Sprintf("MsgBeginRedelegate: Delegator %s auto-withdrew %s\n", sf.DelegatorAddress, strings.Join(coinsRecievedStrings, ", "))
	}
	return fmt.Sprintf("MsgBeginRedelegate: Delegator %s did not auto-withdrawal rewards\n", sf.DelegatorAddress)
}
