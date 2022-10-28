package distribution

import (
	"fmt"
	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"go.uber.org/zap"

	txModule "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"

	parsingTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules"

	stdTypes "github.com/cosmos/cosmos-sdk/types"
	distTypes "github.com/cosmos/cosmos-sdk/x/distribution/types"

	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

var IsMsgFundCommunityPool = map[string]bool{
	"/cosmos.distribution.v1beta1.MsgFundCommunityPool": true,
}

var IsMsgWithdrawValidatorCommission = map[string]bool{
	"/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission": true,
	"withdraw-rewards": true, //NOTE/TODO: not 100% sure if this is only on delegator or validator withdrawal...
}

var IsMsgWithdrawDelegatorReward = map[string]bool{
	"/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward": true,
	"withdraw-rewards": true, //NOTE/TODO: not 100% sure if this is only on delegator or validator withdrawal...
}

type WrapperMsgFundCommunityPool struct {
	txModule.Message
	CosmosMsgFundCommunityPool *distTypes.MsgFundCommunityPool
	Depositor                  string
	Funds                      stdTypes.Coins
}

type WrapperMsgWithdrawValidatorCommission struct {
	txModule.Message
	CosmosMsgWithdrawValidatorCommission *distTypes.MsgWithdrawValidatorCommission
	DelegatorReceiverAddress             string
	CoinsReceived                        stdTypes.Coin
	MultiCoinsReceived                   stdTypes.Coins
}

type WrapperMsgWithdrawDelegatorReward struct {
	txModule.Message
	CosmosMsgWithdrawDelegatorReward *distTypes.MsgWithdrawDelegatorReward
	CoinsReceived                    stdTypes.Coin
	MultiCoinsReceived               stdTypes.Coins
}

// HandleMsg: Handle type checking for MsgFundCommunityPool
func (sf *WrapperMsgFundCommunityPool) HandleMsg(msgType string, msg stdTypes.Msg, log *txModule.TxLogMessage) error {
	sf.Type = msgType
	sf.CosmosMsgFundCommunityPool = msg.(*distTypes.MsgFundCommunityPool)

	//Confirm that the action listed in the message log matches the Message type
	valid_log := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !valid_log {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	//Funds sent and sender address are pulled from the parsed Cosmos Msg
	sf.Depositor = sf.CosmosMsgFundCommunityPool.Depositor
	sf.Funds = sf.CosmosMsgFundCommunityPool.Amount

	return nil
}

// HandleMsg: Handle type checking for MsgWithdrawDelegatorReward
func (sf *WrapperMsgWithdrawValidatorCommission) HandleMsg(msgType string, msg stdTypes.Msg, log *txModule.TxLogMessage) error {
	sf.Type = msgType
	sf.CosmosMsgWithdrawValidatorCommission = msg.(*distTypes.MsgWithdrawValidatorCommission)

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

// CosmUnmarshal(): Unmarshal JSON for MsgWithdrawDelegatorReward
func (sf *WrapperMsgWithdrawDelegatorReward) HandleMsg(msgType string, msg stdTypes.Msg, log *txModule.TxLogMessage) error {
	sf.Type = msgType
	sf.CosmosMsgWithdrawDelegatorReward = msg.(*distTypes.MsgWithdrawDelegatorReward)

	//Confirm that the action listed in the message log matches the Message type
	valid_log := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !valid_log {
		config.Log.Error("Msg log invalid")
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	//The attribute in the log message that shows you the delegator withdrawal address and amount received
	delegatorReceivedCoinsEvt := txModule.GetEventWithType(bankTypes.EventTypeCoinReceived, log)
	if delegatorReceivedCoinsEvt == nil {
		config.Log.Error("Failed to get delegatorReceivedCoinsEvt from msg")
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	delegator_address := txModule.GetValueForAttribute(bankTypes.AttributeKeyReceiver, delegatorReceivedCoinsEvt)
	coins_received := txModule.GetValueForAttribute("amount", delegatorReceivedCoinsEvt)

	//This may be able to be optimized by doing one or the other
	coin, err := stdTypes.ParseCoinNormalized(coins_received)
	if err != nil {
		coins, err := stdTypes.ParseCoinsNormalized(coins_received)
		if err != nil {
			config.Log.Error("Error parsing coins normalized", zap.Error(err))
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

func (sf *WrapperMsgFundCommunityPool) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	var relevantData []parsingTypes.MessageRelevantInformation = make([]parsingTypes.MessageRelevantInformation, len(sf.Funds))

	for i, v := range sf.Funds {
		relevantData[i].AmountSent = v.Amount.BigInt()
		relevantData[i].DenominationSent = v.Denom
		relevantData[i].SenderAddress = sf.Depositor
	}

	return relevantData
}

func (sf *WrapperMsgWithdrawValidatorCommission) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	if sf.CoinsReceived.IsNil() {
		var relevantData []parsingTypes.MessageRelevantInformation = make([]parsingTypes.MessageRelevantInformation, len(sf.MultiCoinsReceived))

		for i, v := range sf.MultiCoinsReceived {
			relevantData[i] = parsingTypes.MessageRelevantInformation{
				AmountReceived:       v.Amount.BigInt(),
				DenominationReceived: v.Denom,
				SenderAddress:        "",
				ReceiverAddress:      sf.DelegatorReceiverAddress,
			}
		}

		return relevantData
	} else {
		var relevantData []parsingTypes.MessageRelevantInformation = make([]parsingTypes.MessageRelevantInformation, 1)
		relevantData[0] = parsingTypes.MessageRelevantInformation{
			AmountReceived:       sf.CoinsReceived.Amount.BigInt(),
			DenominationReceived: sf.CoinsReceived.Denom,
			SenderAddress:        "",
			ReceiverAddress:      sf.DelegatorReceiverAddress,
		}
		return relevantData
	}
}

func (sf *WrapperMsgWithdrawDelegatorReward) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	if sf.CoinsReceived.IsNil() {
		relevantData := make([]parsingTypes.MessageRelevantInformation, len(sf.MultiCoinsReceived))
		for i, v := range sf.MultiCoinsReceived {
			relevantData[i] = parsingTypes.MessageRelevantInformation{
				AmountReceived:       v.Amount.BigInt(),
				DenominationReceived: v.Denom,
				SenderAddress:        "",
				ReceiverAddress:      sf.CosmosMsgWithdrawDelegatorReward.DelegatorAddress,
			}
		}
		return relevantData
	} else {
		relevantData := make([]parsingTypes.MessageRelevantInformation, 1)
		relevantData[0] = parsingTypes.MessageRelevantInformation{
			AmountReceived:       sf.CoinsReceived.Amount.BigInt(),
			DenominationReceived: sf.CoinsReceived.Denom,
			SenderAddress:        "",
			ReceiverAddress:      sf.CosmosMsgWithdrawDelegatorReward.DelegatorAddress,
		}
		return relevantData
	}
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

func (sf *WrapperMsgFundCommunityPool) String() string {
	coinsReceivedString := sf.CosmosMsgFundCommunityPool.Amount.String()
	depositorAddress := sf.CosmosMsgFundCommunityPool.Depositor

	return fmt.Sprintf("MsgFundCommunityPool: Depositor %s gave %s\n",
		depositorAddress, coinsReceivedString)
}
