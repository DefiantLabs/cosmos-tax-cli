package gamm

import (
	"fmt"
	"time"

	parsingTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules"
	txModule "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gammTypes "github.com/osmosis-labs/osmosis/v9/x/gamm/types"
)

var IsMsgSwapExactAmountIn = map[string]bool{
	"/osmosis.gamm.v1beta1.MsgSwapExactAmountIn": true,
}

type WrapperMsgSwapExactAmountIn struct {
	txModule.Message
	OsmosisMsgSwapExactAmountIn *gammTypes.MsgSwapExactAmountIn
	Address                     string
	TokenOut                    sdk.Coin
	TokenIn                     sdk.Coin
}

func (sf *WrapperMsgSwapExactAmountIn) String() string {
	var tokenSwappedOut string
	var tokenSwappedIn string
	if !sf.TokenOut.IsNil() {
		tokenSwappedOut = sf.TokenOut.String()
	}
	if !sf.TokenIn.IsNil() {
		tokenSwappedIn = sf.TokenIn.String()
	}

	return fmt.Sprintf("MsgSwapExactAmountIn: %s swapped in %s and received %s\n",
		sf.Address, tokenSwappedIn, tokenSwappedOut)
}

func (sf *WrapperMsgSwapExactAmountIn) HandleMsg(msgType string, msg sdk.Msg, log *txModule.TxLogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgSwapExactAmountIn = msg.(*gammTypes.MsgSwapExactAmountIn)

	//Confirm that the action listed in the message log matches the Message type
	valid_log := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !valid_log {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	//The attribute in the log message that shows you the tokens swapped
	tokensSwappedEvt := txModule.GetEventWithType(gammTypes.TypeEvtTokenSwapped, log)
	if tokensSwappedEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	//Address of whoever initiated the swap. Will be both sender/receiver.
	senderReceiver := txModule.GetValueForAttribute("sender", tokensSwappedEvt)
	if senderReceiver == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderReceiver

	//This gets the first token swapped in (if there are multiple pools we do not care about intermediates)
	tokenInStr := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensIn, tokensSwappedEvt)
	tokenIn, err := sdk.ParseCoinNormalized(tokenInStr)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenIn = tokenIn

	//This gets the last token swapped out (if there are multiple pools we do not care about intermediates)
	tokenOutStr := txModule.GetLastValueForAttribute(gammTypes.AttributeKeyTokensOut, tokensSwappedEvt)
	tokenOut, err := sdk.ParseCoinNormalized(tokenOutStr)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenOut = tokenOut

	return err
}

type ArbitrageTx struct {
	TokenIn   sdk.Coin
	TokenOut  sdk.Coin
	BlockTime time.Time
}

func (sf *WrapperMsgSwapExactAmountIn) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	var relevantData []parsingTypes.MessageRelevantInformation = make([]parsingTypes.MessageRelevantInformation, 1)
	relevantData[0] = parsingTypes.MessageRelevantInformation{
		AmountSent:           sf.TokenIn.Amount.BigInt(),
		DenominationSent:     sf.TokenIn.Denom,
		AmountReceived:       sf.TokenOut.Amount.BigInt(),
		DenominationReceived: sf.TokenOut.Denom,
		SenderAddress:        sf.Address,
		ReceiverAddress:      sf.Address,
	}
	return relevantData
}
