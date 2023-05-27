package poolmanager

import (
	"fmt"
	"strconv"

	parsingTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules"
	txModule "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-tax-cli/util"
	poolManagerTypes "github.com/DefiantLabs/probe/client/codec/osmosis/v15/x/poolmanager/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	MsgSwapExactAmountIn  = "/osmosis.poolmanager.v1beta1.MsgSwapExactAmountIn"
	MsgSwapExactAmountOut = "/osmosis.poolmanager.v1beta1.MsgSwapExactAmountOut"
)

type WrapperMsgSwapExactAmountIn struct {
	txModule.Message
	OsmosisMsgSwapExactAmountIn *poolManagerTypes.MsgSwapExactAmountIn
	Address                     string
	TokenOut                    sdk.Coin
	TokenIn                     sdk.Coin
}

type WrapperMsgSwapExactAmountOut struct {
	txModule.Message
	OsmosisMsgSwapExactAmountOut *poolManagerTypes.MsgSwapExactAmountOut
	Address                      string
	TokenOut                     sdk.Coin
	TokenIn                      sdk.Coin
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

	return fmt.Sprintf("MsgSwapExactAmountIn (pool-manager): %s swapped in %s and received %s",
		sf.Address, tokenSwappedIn, tokenSwappedOut)
}

func (sf *WrapperMsgSwapExactAmountOut) String() string {
	var tokenSwappedOut string
	var tokenSwappedIn string
	if !sf.TokenOut.IsNil() {
		tokenSwappedOut = sf.TokenOut.String()
	}
	if !sf.TokenIn.IsNil() {
		tokenSwappedIn = sf.TokenIn.String()
	}
	return fmt.Sprintf("MsgSwapExactAmountOut (pool-manager): %s swapped in %s and received %s",
		sf.Address, tokenSwappedIn, tokenSwappedOut)
}

func (sf *WrapperMsgSwapExactAmountIn) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgSwapExactAmountIn = msg.(*poolManagerTypes.MsgSwapExactAmountIn)
	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	sf.TokenIn = sf.OsmosisMsgSwapExactAmountIn.TokenIn
	sf.Address = sf.OsmosisMsgSwapExactAmountIn.Sender

	// The attribute in the log message that shows you the tokens swapped
	tokensSwappedEvt := txModule.GetEventWithType("token_swapped", log)
	if tokensSwappedEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// The last route in the hops gives the token out denom and pool ID for the final output
	lastRoute := sf.OsmosisMsgSwapExactAmountIn.Routes[len(sf.OsmosisMsgSwapExactAmountIn.Routes)-1]
	lastRouteDenom := lastRoute.TokenOutDenom
	lastRoutePoolID := lastRoute.PoolId

	tokenOutStr := txModule.GetLastValueForAttribute("tokens_out", tokensSwappedEvt)
	tokenOutPoolID := txModule.GetLastValueForAttribute("pool_id", tokensSwappedEvt)

	tokenOut, err := sdk.ParseCoinNormalized(tokenOutStr)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// Sanity check last route swap
	if tokenOut.Denom != lastRouteDenom || strconv.FormatUint(lastRoutePoolID, 10) != tokenOutPoolID {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	sf.TokenOut = tokenOut

	return err
}

// This code is currently untested since I cannot find a TX execution for this
// It should be fine for the time being since it is following the same pattern established for GAMM SwapExactAmountOut, which the poolmanager will call
func (sf *WrapperMsgSwapExactAmountOut) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgSwapExactAmountOut = msg.(*poolManagerTypes.MsgSwapExactAmountOut)

	// The attribute in the log message that shows you the tokens swapped
	tokensSwappedEvt := txModule.GetEventWithType("token_swapped", log)
	if tokensSwappedEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// This gets the first token swapped in (if there are multiple pools we do not care about intermediates)
	tokenInStr := txModule.GetValueForAttribute("tokens_in", tokensSwappedEvt)
	tokenIn, err := sdk.ParseCoinNormalized(tokenInStr)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenIn = tokenIn

	sf.Address = sf.OsmosisMsgSwapExactAmountOut.Sender
	sf.TokenOut = sf.OsmosisMsgSwapExactAmountOut.TokenOut
	return err
}

func (sf *WrapperMsgSwapExactAmountIn) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	relevantData := make([]parsingTypes.MessageRelevantInformation, 1)
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

func (sf *WrapperMsgSwapExactAmountOut) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	relevantData := make([]parsingTypes.MessageRelevantInformation, 1)
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
