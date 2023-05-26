package ibc

import (
	"fmt"

	parsingTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules"
	txModule "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-tax-cli/util"
	stdTypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
	chantypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
)

const (
	MsgRecvPacket      = "/ibc.core.channel.v1.MsgRecvPacket"
	MsgAcknowledgement = "/ibc.core.channel.v1.MsgAcknowledgement"

	// Explicitly ignored messages for tx parsing purposes
	MsgTransfer           = "/ibc.applications.transfer.v1.MsgTransfer"
	MsgChannelOpenTry     = "/ibc.core.channel.v1.MsgChannelOpenTry"
	MsgChannelOpenConfirm = "/ibc.core.channel.v1.MsgChannelOpenConfirm"
	MsgChannelOpenInit    = "/ibc.core.channel.v1.MsgChannelOpenInit"
	MsgChannelOpenAck     = "/ibc.core.channel.v1.MsgChannelOpenAck"

	MsgTimeout        = "/ibc.core.channel.v1.MsgTimeout"
	MsgTimeoutOnClose = "/ibc.core.channel.v1.MsgTimeoutOnClose"

	MsgConnectionOpenTry     = "/ibc.core.connection.v1.MsgConnectionOpenTry"
	MsgConnectionOpenConfirm = "/ibc.core.connection.v1.MsgConnectionOpenConfirm"
	MsgConnectionOpenInit    = "/ibc.core.connection.v1.MsgConnectionOpenInit"
	MsgConnectionOpenAck     = "/ibc.core.connection.v1.MsgConnectionOpenAck"

	MsgCreateClient = "/ibc.core.client.v1.MsgCreateClient"
	MsgUpdateClient = "/ibc.core.client.v1.MsgUpdateClient"
)

type WrapperMsgRecvPacket struct {
	txModule.Message
	MsgRecvPacket   *chantypes.MsgRecvPacket
	Sequence        uint64
	SenderAddress   string
	ReceiverAddress string
	Amount          stdTypes.Coin
}

func (w *WrapperMsgRecvPacket) HandleMsg(msgType string, msg stdTypes.Msg, log *txModule.LogMessage) error {
	w.Type = msgType
	w.MsgRecvPacket = msg.(*chantypes.MsgRecvPacket)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(w.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// Unmarshal the json encoded packet data so we can access sender, receiver and denom info
	var data types.FungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(w.MsgRecvPacket.Packet.GetData(), &data); err != nil {
		return err
	}

	w.SenderAddress = data.Sender
	w.ReceiverAddress = data.Receiver
	w.Sequence = w.MsgRecvPacket.Packet.Sequence

	amount, ok := stdTypes.NewIntFromString(data.Amount)
	if !ok {
		return fmt.Errorf("failed to convert denom amount to sdk.Int, got(%s)", data.Amount)
	}

	w.Amount = stdTypes.NewCoin(data.Denom, amount)

	return nil
}

func (w *WrapperMsgRecvPacket) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	if w.Amount.IsNil() {
		return nil
	}

	// MsgRecvPacket indicates a user has received assets on this chain so amount sent will always be 0
	amountSent := stdTypes.NewInt(0)

	return []parsingTypes.MessageRelevantInformation{{
		SenderAddress:        w.SenderAddress,
		ReceiverAddress:      w.ReceiverAddress,
		AmountSent:           amountSent.BigInt(),
		AmountReceived:       w.Amount.Amount.BigInt(),
		DenominationSent:     "",
		DenominationReceived: w.Amount.Denom,
	}}
}

func (w *WrapperMsgRecvPacket) String() string {
	if w.Amount.IsNil() {
		return fmt.Sprintf("MsgRecvPacket: IBC transfer from %s to %s did not include an amount\n", w.SenderAddress, w.ReceiverAddress)
	}
	return fmt.Sprintf("MsgRecvPacket: IBC transfer of %s from %s to %s\n", w.Amount, w.SenderAddress, w.ReceiverAddress)
}

type WrapperMsgAcknowledgement struct {
	txModule.Message
	MsgAcknowledgement *chantypes.MsgAcknowledgement
	Sequence           uint64
	SenderAddress      string
	ReceiverAddress    string
	Amount             stdTypes.Coin
}

func (w *WrapperMsgAcknowledgement) HandleMsg(msgType string, msg stdTypes.Msg, log *txModule.LogMessage) error {
	w.Type = msgType
	w.MsgAcknowledgement = msg.(*chantypes.MsgAcknowledgement)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(w.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// Unmarshal the json encoded packet data so we can access sender, receiver and denom info
	var data types.FungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(w.MsgAcknowledgement.Packet.GetData(), &data); err != nil {
		// If there was a failure then this ack was not for a token transfer packet,
		// currently we only consider successful token transfers taxable events.
		return err
	}

	w.SenderAddress = data.Sender
	w.ReceiverAddress = data.Receiver
	w.Sequence = w.MsgAcknowledgement.Packet.Sequence

	amount, ok := stdTypes.NewIntFromString(data.Amount)
	if !ok {
		return fmt.Errorf("failed to convert denom amount to sdk.Int, got(%s)", data.Amount)
	}

	w.Amount = stdTypes.NewCoin(data.Denom, amount)

	// Acknowledgements can contain an error & we only want to index successful acks,
	// so we need to check the ack bytes to determine if it was a result or an error.
	var ack chantypes.Acknowledgement
	if err := types.ModuleCdc.UnmarshalJSON(w.MsgAcknowledgement.Acknowledgement, &ack); err != nil {
		return fmt.Errorf("cannot unmarshal ICS-20 transfer packet acknowledgement: %v", err)
	}

	switch ack.Response.(type) {
	case *chantypes.Acknowledgement_Error:
		return fmt.Errorf("acknowledgement contained an error, %s", ack.Response)
	default:
		// the acknowledgement succeeded on the receiving chain
		return nil
	}
}

func (w *WrapperMsgAcknowledgement) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	if w.Amount.IsNil() {
		return nil
	}

	// MsgAcknowledgement indicates a user has successfully sent a packet
	// so the received amount will always be zero
	amountReceived := stdTypes.NewInt(0)

	return []parsingTypes.MessageRelevantInformation{{
		SenderAddress:        w.SenderAddress,
		ReceiverAddress:      w.ReceiverAddress,
		AmountSent:           w.Amount.Amount.BigInt(),
		AmountReceived:       amountReceived.BigInt(),
		DenominationSent:     w.Amount.Denom,
		DenominationReceived: "",
	}}
}

func (w *WrapperMsgAcknowledgement) String() string {
	if w.Amount.IsNil() {
		return fmt.Sprintf("MsgAcknowledgement: IBC transfer from %s to %s did not include an amount\n", w.SenderAddress, w.ReceiverAddress)
	}
	return fmt.Sprintf("MsgAcknowledgement: IBC transfer of %s from %s to %s\n", w.Amount, w.SenderAddress, w.ReceiverAddress)
}
