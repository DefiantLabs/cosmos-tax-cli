package ibc

import (
	"fmt"

	parsingTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules"
	txModule "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-tax-cli/util"
	"github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"

	stdTypes "github.com/cosmos/cosmos-sdk/types"
)

const (
	MsgTransfer = "/ibc.applications.transfer.v1.MsgTransfer"

	// Explicitly ignored messages for tx parsing purposes
	MsgAcknowledgement    = "/ibc.core.channel.v1.MsgAcknowledgement"
	MsgChannelOpenTry     = "/ibc.core.channel.v1.MsgChannelOpenTry"
	MsgChannelOpenConfirm = "/ibc.core.channel.v1.MsgChannelOpenConfirm"
	MsgChannelOpenInit    = "/ibc.core.channel.v1.MsgChannelOpenInit"
	MsgChannelOpenAck     = "/ibc.core.channel.v1.MsgChannelOpenAck"
	MsgRecvPacket         = "/ibc.core.channel.v1.MsgRecvPacket"
	MsgTimeout            = "/ibc.core.channel.v1.MsgTimeout"
	MsgTimeoutOnClose     = "/ibc.core.channel.v1.MsgTimeoutOnClose"

	MsgConnectionOpenTry     = "/ibc.core.connection.v1.MsgConnectionOpenTry"
	MsgConnectionOpenConfirm = "/ibc.core.connection.v1.MsgConnectionOpenConfirm"
	MsgConnectionOpenInit    = "/ibc.core.connection.v1.MsgConnectionOpenInit"
	MsgConnectionOpenAck     = "/ibc.core.connection.v1.MsgConnectionOpenAck"

	MsgCreateClient = "/ibc.core.client.v1.MsgCreateClient"
	MsgUpdateClient = "/ibc.core.client.v1.MsgUpdateClient"
)

type WrapperMsgTransfer struct {
	txModule.Message
	CosmosMsgTransfer *types.MsgTransfer
	SenderAddress     string
	ReceiverAddress   string
	Amount            *stdTypes.Coin
}

// HandleMsg: Handle type checking for MsgFundCommunityPool
func (sf *WrapperMsgTransfer) HandleMsg(msgType string, msg stdTypes.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.CosmosMsgTransfer = msg.(*types.MsgTransfer)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// Funds sent and sender address are pulled from the parsed Cosmos Msg
	sf.SenderAddress = sf.CosmosMsgTransfer.Sender
	sf.ReceiverAddress = sf.CosmosMsgTransfer.Receiver
	sf.Amount = &sf.CosmosMsgTransfer.Token

	return nil
}

func (sf *WrapperMsgTransfer) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	if sf.Amount != nil {
		return []parsingTypes.MessageRelevantInformation{{
			SenderAddress:        sf.SenderAddress,
			ReceiverAddress:      sf.ReceiverAddress,
			AmountSent:           sf.Amount.Amount.BigInt(),
			AmountReceived:       sf.Amount.Amount.BigInt(),
			DenominationSent:     sf.Amount.Denom,
			DenominationReceived: sf.Amount.Denom,
		}}
	}
	return nil
}

func (sf *WrapperMsgTransfer) String() string {
	if sf.Amount == nil {
		return fmt.Sprintf("MsgTransfer: IBC transfer from %s to %s did not include an amount", sf.SenderAddress, sf.ReceiverAddress)
	}
	return fmt.Sprintf("MsgTransfer: IBC transfer of %s from %s to %s", sf.CosmosMsgTransfer.Token, sf.SenderAddress, sf.ReceiverAddress)
}
