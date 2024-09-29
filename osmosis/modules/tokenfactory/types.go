package tokenfactory

import (
	"fmt"

	parsingTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules"
	txModule "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-tax-cli/util"
	sdk "github.com/cosmos/cosmos-sdk/types"
	tfTypes "github.com/osmosis-labs/osmosis/v26/x/tokenfactory/types"
)

const (
	MsgCreateDenom       = "/osmosis.tokenfactory.v1beta1.MsgCreateDenom"
	MsgMint              = "/osmosis.tokenfactory.v1beta1.MsgMint"
	MsgBurn              = "/osmosis.tokenfactory.v1beta1.MsgBurn"
	MsgSetDenomMetadata  = "/osmosis.tokenfactory.v1beta1.MsgSetDenomMetadata"
	MsgSetBeforeSendHook = "/osmosis.tokenfactory.v1beta1.MsgSetBeforeSendHook"
	MsgChangeAdmin       = "/osmosis.tokenfactory.v1beta1.MsgChangeAdmin"
)

// Create interface definition for MsgMint
type WrapperMsgMint struct {
	txModule.Message
	OsmosisMsgMint *tfTypes.MsgMint
	Address        string
	CoinReceived   sdk.Coin
}

func (sf *WrapperMsgMint) String() string {
	return fmt.Sprintf("MsgMint: %s received %s",
		sf.Address, sf.CoinReceived.String())
}

func (sf *WrapperMsgMint) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgMint = msg.(*tfTypes.MsgMint)

	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// This logic is pulled from the Osmosis codebase of who gets minted to.
	// See here: https://github.com/osmosis-labs/osmosis/blob/7c81b90825ab2efe92444ac167191b8d041e0c21/x/tokenfactory/keeper/msg_server.go#L63-L65

	sf.Address = sf.OsmosisMsgMint.MintToAddress

	if sf.Address == "" {
		sf.Address = sf.OsmosisMsgMint.Sender
	}

	sf.CoinReceived = sf.OsmosisMsgMint.Amount

	return nil
}

func (sf *WrapperMsgMint) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	relevantData := make([]parsingTypes.MessageRelevantInformation, 1)

	relevantData[0] = parsingTypes.MessageRelevantInformation{
		AmountReceived:       sf.CoinReceived.Amount.BigInt(),
		DenominationReceived: sf.CoinReceived.Denom,
		ReceiverAddress:      sf.Address,
	}
	return relevantData
}

type WrapperMsgBurn struct {
	txModule.Message
	OsmosisMsgBurn *tfTypes.MsgBurn
	Address        string
	CoinSent       sdk.Coin
}

func (sf *WrapperMsgBurn) String() string {
	return fmt.Sprintf("MsgBurn: %s sent %s",
		sf.Address, sf.CoinSent.String())
}

func (sf *WrapperMsgBurn) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgBurn = msg.(*tfTypes.MsgBurn)

	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	sf.Address = sf.OsmosisMsgBurn.BurnFromAddress

	if sf.Address == "" {
		sf.Address = sf.OsmosisMsgBurn.Sender
	}

	sf.CoinSent = sf.OsmosisMsgBurn.Amount

	return nil
}

func (sf *WrapperMsgBurn) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	relevantData := make([]parsingTypes.MessageRelevantInformation, 1)

	relevantData[0] = parsingTypes.MessageRelevantInformation{
		AmountSent:       sf.CoinSent.Amount.BigInt(),
		DenominationSent: sf.CoinSent.Denom,
		SenderAddress:    sf.Address,
	}
	return relevantData
}
