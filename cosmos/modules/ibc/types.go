package ibc

import (
	//ibcTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/ibclegacy/applications/transfer/types"
	txModule "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	stdTypes "github.com/cosmos/cosmos-sdk/types"
)

const (
	MsgTransfer        = "/ibc.applications.transfer.v1.MsgTransfer"
	MsgAcknowledgement = "/ibc.core.channel.v1.MsgAcknowledgement"
	MsgRecvPacket      = "/ibc.core.channel.v1.MsgRecvPacket"
	MsgTimeout         = "/ibc.core.channel.v1.MsgTimeout"
	MsgUpdateClient    = "/ibc.core.client.v1.MsgUpdateClient"
)

type WrapperMsgTransfer struct {
	txModule.Message
	//CosmosMsgTransfer *ibcTypes.MsgTransfer
	SenderAddress   string
	ReceiverAddress string
	Amount          *stdTypes.Coin
}

/*
// HandleMsg: Handle type checking for MsgFundCommunityPool
func (sf *WrapperMsgTransfer) HandleMsg(msgType string, msg stdTypes.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.CosmosMsgTransfer = msg.(*ibcTypes.MsgTransfer)

	//Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	//Funds sent and sender address are pulled from the parsed Cosmos Msg
	sf.SenderAddress = sf.CosmosMsgTransfer.Sender
	sf.ReceiverAddress = sf.CosmosMsgTransfer.Receiver
	sf.Amount = &sf.CosmosMsgTransfer.Token

	return nil
}


*/
