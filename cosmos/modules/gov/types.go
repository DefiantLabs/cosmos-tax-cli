package gov

import (
	"fmt"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	parsingTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules"
	txModule "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-tax-cli/util"
	stdTypes "github.com/cosmos/cosmos-sdk/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govTypesV1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govTypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

const (
	MsgVote             = "/cosmos.gov.v1beta1.MsgVote"
	MsgDeposit          = "/cosmos.gov.v1beta1.MsgDeposit"        // handle additional deposits to the given proposal
	MsgSubmitProposal   = "/cosmos.gov.v1beta1.MsgSubmitProposal" // handle the initial deposit for the proposer
	MsgVoteWeighted     = "/cosmos.gov.v1beta1.MsgVoteWeighted"
	MsgVoteV1           = "/cosmos.gov.v1.MsgVote"
	MsgDepositV1        = "/cosmos.gov.v1.MsgDeposit"        // handle additional deposits to the given proposal
	MsgSubmitProposalV1 = "/cosmos.gov.v1.MsgSubmitProposal" // handle the initial deposit for the proposer
	MsgVoteWeightedV1   = "/cosmos.gov.v1.MsgVoteWeighted"
)

type WrapperMsgSubmitProposal struct {
	txModule.Message
	CosmosMsgSubmitProposal *govTypes.MsgSubmitProposal
	CoinReceived            stdTypes.Coin
	MultiCoinsReceived      stdTypes.Coins
	DepositReceiverAddress  string
}

type WrapperMsgDeposit struct {
	txModule.Message
	CosmosMsgDeposit       *govTypes.MsgDeposit
	CoinReceived           stdTypes.Coin
	MultiCoinsReceived     stdTypes.Coins
	DepositReceiverAddress string
}

type WrapperMsgSubmitProposalV1 struct {
	txModule.Message
	CosmosMsgSubmitProposal *govTypesV1.MsgSubmitProposal
	CoinReceived            stdTypes.Coin
	MultiCoinsReceived      stdTypes.Coins
	DepositReceiverAddress  string
}

type WrapperMsgDepositV1 struct {
	txModule.Message
	CosmosMsgDeposit       *govTypesV1.MsgDeposit
	CoinReceived           stdTypes.Coin
	MultiCoinsReceived     stdTypes.Coins
	DepositReceiverAddress string
}

func (sf *WrapperMsgSubmitProposal) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	relevantData := make([]parsingTypes.MessageRelevantInformation, len(sf.CosmosMsgSubmitProposal.InitialDeposit))

	for i, v := range sf.CosmosMsgSubmitProposal.InitialDeposit {
		var currRelevantData parsingTypes.MessageRelevantInformation
		currRelevantData.SenderAddress = sf.CosmosMsgSubmitProposal.Proposer
		currRelevantData.ReceiverAddress = sf.DepositReceiverAddress

		// Amount always seems to be an integer, float may be an extra unneeded step
		currRelevantData.AmountSent = v.Amount.BigInt()
		currRelevantData.DenominationSent = v.Denom

		// This is required since we do CSV parsing on the receiver here too
		currRelevantData.AmountReceived = v.Amount.BigInt()
		currRelevantData.DenominationReceived = v.Denom

		relevantData[i] = currRelevantData
	}

	return relevantData
}

func (sf *WrapperMsgDeposit) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	relevantData := make([]parsingTypes.MessageRelevantInformation, len(sf.CosmosMsgDeposit.Amount))

	for i, v := range sf.CosmosMsgDeposit.Amount {
		var currRelevantData parsingTypes.MessageRelevantInformation
		currRelevantData.SenderAddress = sf.CosmosMsgDeposit.Depositor
		currRelevantData.ReceiverAddress = sf.DepositReceiverAddress

		// Amount always seems to be an integer, float may be an extra unneeded step
		currRelevantData.AmountSent = v.Amount.BigInt()
		currRelevantData.DenominationSent = v.Denom

		// This is required since we do CSV parsing on the receiver here too
		currRelevantData.AmountReceived = v.Amount.BigInt()
		currRelevantData.DenominationReceived = v.Denom

		relevantData[i] = currRelevantData
	}

	return relevantData
}

func (sf *WrapperMsgSubmitProposalV1) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	relevantData := make([]parsingTypes.MessageRelevantInformation, len(sf.CosmosMsgSubmitProposal.InitialDeposit))

	for i, v := range sf.CosmosMsgSubmitProposal.InitialDeposit {
		var currRelevantData parsingTypes.MessageRelevantInformation
		currRelevantData.SenderAddress = sf.CosmosMsgSubmitProposal.Proposer
		currRelevantData.ReceiverAddress = sf.DepositReceiverAddress

		// Amount always seems to be an integer, float may be an extra unneeded step
		currRelevantData.AmountSent = v.Amount.BigInt()
		currRelevantData.DenominationSent = v.Denom

		// This is required since we do CSV parsing on the receiver here too
		currRelevantData.AmountReceived = v.Amount.BigInt()
		currRelevantData.DenominationReceived = v.Denom

		relevantData[i] = currRelevantData
	}

	return relevantData
}

func (sf *WrapperMsgDepositV1) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	relevantData := make([]parsingTypes.MessageRelevantInformation, len(sf.CosmosMsgDeposit.Amount))

	for i, v := range sf.CosmosMsgDeposit.Amount {
		var currRelevantData parsingTypes.MessageRelevantInformation
		currRelevantData.SenderAddress = sf.CosmosMsgDeposit.Depositor
		currRelevantData.ReceiverAddress = sf.DepositReceiverAddress

		// Amount always seems to be an integer, float may be an extra unneeded step
		currRelevantData.AmountSent = v.Amount.BigInt()
		currRelevantData.DenominationSent = v.Denom

		// This is required since we do CSV parsing on the receiver here too
		currRelevantData.AmountReceived = v.Amount.BigInt()
		currRelevantData.DenominationReceived = v.Denom

		relevantData[i] = currRelevantData
	}

	return relevantData
}

// Proposal with an initial deposit
func (sf *WrapperMsgSubmitProposal) HandleMsg(msgType string, msg stdTypes.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.CosmosMsgSubmitProposal = msg.(*govTypes.MsgSubmitProposal)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// If there was an initial deposit, there will be a transfer log with sender and amount
	proposerDepositedCoinsEvt := txModule.GetEventWithType(bankTypes.EventTypeTransfer, log)
	if proposerDepositedCoinsEvt == nil {
		return nil
	}

	coinsReceived, err := txModule.GetValueForAttribute("amount", proposerDepositedCoinsEvt)
	if err != nil {
		return err
	}

	recipientAccount, err := txModule.GetValueForAttribute("recipient", proposerDepositedCoinsEvt)
	if err != nil {
		return err
	}

	sf.DepositReceiverAddress = recipientAccount

	// This may be able to be optimized by doing one or the other
	coin, err := stdTypes.ParseCoinNormalized(coinsReceived)
	if err != nil {
		sf.MultiCoinsReceived, err = stdTypes.ParseCoinsNormalized(coinsReceived)
		if err != nil {
			config.Log.Error("Error parsing coins normalized", err)
			return err
		}
	} else {
		sf.CoinReceived = coin
	}

	return err
}

// Additional deposit
func (sf *WrapperMsgDeposit) HandleMsg(msgType string, msg stdTypes.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.CosmosMsgDeposit = msg.(*govTypes.MsgDeposit)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// If there was an initial deposit, there will be a transfer log with sender and amount
	proposerDepositedCoinsEvt := txModule.GetEventWithType(bankTypes.EventTypeTransfer, log)
	if proposerDepositedCoinsEvt == nil {
		return nil
	}

	coinsReceived, err := txModule.GetValueForAttribute("amount", proposerDepositedCoinsEvt)
	if err != nil {
		return err
	}

	// This may be able to be optimized by doing one or the other
	coin, err := stdTypes.ParseCoinNormalized(coinsReceived)
	if err != nil {
		return err
	}

	recipientAccount, err := txModule.GetValueForAttribute("recipient", proposerDepositedCoinsEvt)
	if err != nil {
		return err
	}

	sf.DepositReceiverAddress = recipientAccount

	if err != nil {
		sf.MultiCoinsReceived, err = stdTypes.ParseCoinsNormalized(coinsReceived)
		if err != nil {
			config.Log.Error("Error parsing coins normalized", err)
			return err
		}
	} else {
		sf.CoinReceived = coin
	}

	return err
}

// Proposal with an initial deposit
func (sf *WrapperMsgSubmitProposalV1) HandleMsg(msgType string, msg stdTypes.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.CosmosMsgSubmitProposal = msg.(*govTypesV1.MsgSubmitProposal)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// If there was an initial deposit, there will be a transfer log with sender and amount
	proposerDepositedCoinsEvt := txModule.GetEventWithType(bankTypes.EventTypeTransfer, log)
	if proposerDepositedCoinsEvt == nil {
		return nil
	}

	coinsReceived, err := txModule.GetValueForAttribute("amount", proposerDepositedCoinsEvt)
	if err != nil {
		return err
	}

	recipientAccount, err := txModule.GetValueForAttribute("recipient", proposerDepositedCoinsEvt)
	if err != nil {
		return err
	}

	sf.DepositReceiverAddress = recipientAccount

	// This may be able to be optimized by doing one or the other
	coin, err := stdTypes.ParseCoinNormalized(coinsReceived)
	if err != nil {
		sf.MultiCoinsReceived, err = stdTypes.ParseCoinsNormalized(coinsReceived)
		if err != nil {
			config.Log.Error("Error parsing coins normalized", err)
			return err
		}
	} else {
		sf.CoinReceived = coin
	}

	return err
}

// Additional deposit
func (sf *WrapperMsgDepositV1) HandleMsg(msgType string, msg stdTypes.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.CosmosMsgDeposit = msg.(*govTypesV1.MsgDeposit)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// If there was an initial deposit, there will be a transfer log with sender and amount
	proposerDepositedCoinsEvt := txModule.GetEventWithType(bankTypes.EventTypeTransfer, log)
	if proposerDepositedCoinsEvt == nil {
		return nil
	}

	coinsReceived, err := txModule.GetValueForAttribute("amount", proposerDepositedCoinsEvt)
	if err != nil {
		return err
	}

	// This may be able to be optimized by doing one or the other
	coin, err := stdTypes.ParseCoinNormalized(coinsReceived)
	if err != nil {
		return err
	}

	recipientAccount, err := txModule.GetValueForAttribute("recipient", proposerDepositedCoinsEvt)
	if err != nil {
		return err
	}

	sf.DepositReceiverAddress = recipientAccount

	if err != nil {
		sf.MultiCoinsReceived, err = stdTypes.ParseCoinsNormalized(coinsReceived)
		if err != nil {
			config.Log.Error("Error parsing coins normalized", err)
			return err
		}
	} else {
		sf.CoinReceived = coin
	}

	return err
}

func (sf *WrapperMsgDeposit) String() string {
	return fmt.Sprintf("MsgDeposit: Address %s deposited %s",
		sf.CosmosMsgDeposit.Depositor, sf.CosmosMsgDeposit.Amount)
}

func (sf *WrapperMsgSubmitProposal) String() string {
	return fmt.Sprintf("MsgSubmit: Address %s deposited %s",
		sf.CosmosMsgSubmitProposal.Proposer, sf.CosmosMsgSubmitProposal.InitialDeposit)
}

func (sf *WrapperMsgDepositV1) String() string {
	return fmt.Sprintf("MsgDeposit: Address %s deposited %s",
		sf.CosmosMsgDeposit.Depositor, sf.CosmosMsgDeposit.Amount)
}

func (sf *WrapperMsgSubmitProposalV1) String() string {
	return fmt.Sprintf("MsgSubmit: Address %s deposited %s",
		sf.CosmosMsgSubmitProposal.Proposer, sf.CosmosMsgSubmitProposal.InitialDeposit)
}
