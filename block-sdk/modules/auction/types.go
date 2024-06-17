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

// The Skip Block SDK MsgAuctionBid message allows for the creation of a bid to execute a set of Transactions at the top of a block.
// It is part of the MEV Lane x/auction module (https://docs.skip.money/blocksdk/lanes/existing-lanes/mev).
// Research is being done on how the application should handle this message in the TX message parser.
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
