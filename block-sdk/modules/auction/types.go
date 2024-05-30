package auction

import (
	parsingTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules"
	txModule "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	MsgUpdateParams = "/sdk.auction.v1.MsgUpdateParams"
	MsgAuctionBid   = "/sdk.auction.v1.MsgAuctionBid"
)

// WrapperMsgAuctionBid is a wrapper for MsgAuctionBid

type WrapperMsgAuctionBid struct {
	txModule.Message
}

func (sf *WrapperMsgAuctionBid) String() string {
	return ""
}

func (sf *WrapperMsgAuctionBid) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	return nil
}

func (sf *WrapperMsgAuctionBid) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	return nil
}
