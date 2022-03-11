package staking

import (
	"cosmos-exporter/cosmos/modules/tx"
	txModule "cosmos-exporter/cosmos/modules/tx"
	"encoding/json"
	"fmt"

	stdTypes "github.com/cosmos/cosmos-sdk/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distTypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

type WrapperMsgWithdrawValidatorCommission struct {
	txModule.Message
	CosmosMsgWithdrawValidatorCommission distTypes.MsgWithdrawValidatorCommission
	DelegatorReceiverAddress             string
	CoinsReceived                        stdTypes.Coin
}

type WrapperMsgWithdrawDelegatorReward struct {
	txModule.Message
	CosmosMsgWithdrawDelegatorReward distTypes.MsgWithdrawDelegatorReward
	CoinsReceived                    stdTypes.Coin
}

func (sf *WrapperMsgWithdrawDelegatorReward) String() string {
	return fmt.Sprintf("MsgWithdrawDelegatorReward: Delegator %s received %s\n",
		sf.CosmosMsgWithdrawDelegatorReward.DelegatorAddress, sf.CoinsReceived)
}

func (sf *WrapperMsgWithdrawValidatorCommission) String() string {
	return fmt.Sprintf("WrapperMsgWithdrawValidatorCommission: Validator %s commission withdrawn. Delegator %s received %s\n",
		sf.CosmosMsgWithdrawValidatorCommission.ValidatorAddress, sf.DelegatorReceiverAddress, sf.CoinsReceived)
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
		return err
	}

	sf.CoinsReceived = coin
	return err
}

//CosmUnmarshal(): Unmarshal JSON for MsgWithdrawDelegatorReward
func (sf *WrapperMsgWithdrawDelegatorReward) CosmUnmarshal(msgType string, raw []byte, log *tx.TxLogMessage) error {
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
	coin, err := stdTypes.ParseCoinNormalized(coins_received)
	if err != nil {
		return err
	}
	if sf.CosmosMsgWithdrawDelegatorReward.DelegatorAddress != delegator_address {
		return fmt.Errorf("transaction delegator address %s does not match log event '%s' delegator address %s",
			sf.CosmosMsgWithdrawDelegatorReward.DelegatorAddress, bankTypes.EventTypeCoinReceived, delegator_address)
	}

	sf.CoinsReceived = coin
	return err
}
