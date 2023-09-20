package gamm

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	parsingTypes "github.com/DefiantLabs/cosmos-indexer/cosmos/modules"
	txModule "github.com/DefiantLabs/cosmos-indexer/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-indexer/util"
	osmosisOldTypes "github.com/DefiantLabs/lens/extra-codecs/osmosis/gamm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	gammBalancerPoolModelsTypes "github.com/osmosis-labs/osmosis/v19/x/gamm/pool-models/balancer"
	gammTypes "github.com/osmosis-labs/osmosis/v19/x/gamm/types"
)

const (
	MsgSwapExactAmountIn       = "/osmosis.gamm.v1beta1.MsgSwapExactAmountIn"
	MsgSwapExactAmountOut      = "/osmosis.gamm.v1beta1.MsgSwapExactAmountOut"
	MsgJoinSwapExternAmountIn  = "/osmosis.gamm.v1beta1.MsgJoinSwapExternAmountIn"
	MsgJoinSwapShareAmountOut  = "/osmosis.gamm.v1beta1.MsgJoinSwapShareAmountOut"
	MsgJoinPool                = "/osmosis.gamm.v1beta1.MsgJoinPool"
	MsgExitSwapShareAmountIn   = "/osmosis.gamm.v1beta1.MsgExitSwapShareAmountIn"
	MsgExitSwapExternAmountOut = "/osmosis.gamm.v1beta1.MsgExitSwapExternAmountOut"
	MsgExitPool                = "/osmosis.gamm.v1beta1.MsgExitPool"
	MsgCreatePool              = "/osmosis.gamm.v1beta1.MsgCreatePool"
	MsgCreateBalancerPool      = "/osmosis.gamm.v1beta1.MsgCreateBalancerPool"
	OldMsgCreateBalancerPool   = "/osmosis.gamm.poolmodels.balancer.v1beta1.MsgCreateBalancerPool"
)

const (
	EventTypeTransfer    = "transfer"
	EventTypeClaim       = "claim"
	EventAttributeAmount = "amount"
)

type WrapperMsgSwapExactAmountIn struct {
	txModule.Message
	OsmosisMsgSwapExactAmountIn *gammTypes.MsgSwapExactAmountIn
	Address                     string
	TokenOut                    sdk.Coin
	TokenIn                     sdk.Coin
}

// Same as WrapperMsgSwapExactAmountIn but with different handlers.
// This is due to the Osmosis SDK emitting different Events (chain upgrades).
type WrapperMsgSwapExactAmountIn2 struct {
	WrapperMsgSwapExactAmountIn
}

// Same as WrapperMsgSwapExactAmountIn but with different handlers.
// This is due to the Osmosis SDK emitting different Events (chain upgrades).
type WrapperMsgSwapExactAmountIn3 struct {
	WrapperMsgSwapExactAmountIn
}

// Same as WrapperMsgSwapExactAmountIn but with different handlers.
// This is due to the Osmosis SDK emitting different Events (chain upgrades).
type WrapperMsgSwapExactAmountIn4 struct {
	WrapperMsgSwapExactAmountIn
}

// Same as WrapperMsgExitPool but with different handlers.
// This is due to the Osmosis SDK emitting different Events (chain upgrades).
type WrapperMsgExitPool2 struct {
	WrapperMsgExitPool
}

type WrapperMsgSwapExactAmountOut struct {
	txModule.Message
	OsmosisMsgSwapExactAmountOut *gammTypes.MsgSwapExactAmountOut
	Address                      string
	TokenOut                     sdk.Coin
	TokenIn                      sdk.Coin
}

type WrapperMsgJoinSwapExternAmountIn struct {
	txModule.Message
	OsmosisMsgJoinSwapExternAmountIn *gammTypes.MsgJoinSwapExternAmountIn
	Address                          string
	TokenOut                         sdk.Coin
	TokenIn                          sdk.Coin
}

type WrapperMsgJoinSwapExternAmountIn2 struct {
	WrapperMsgJoinSwapExternAmountIn
}

type WrapperMsgJoinSwapShareAmountOut struct {
	txModule.Message
	OsmosisMsgJoinSwapShareAmountOut *gammTypes.MsgJoinSwapShareAmountOut
	Address                          string
	TokenOut                         sdk.Coin
	TokenIn                          sdk.Coin
}

// Same as WrapperMsgJoinSwapShareAmountOut but with different handlers.
// This is due to the Osmosis SDK emitting different Events (chain upgrades).
type WrapperMsgJoinSwapShareAmountOut2 struct {
	WrapperMsgJoinSwapShareAmountOut
}

type WrapperMsgJoinPool struct {
	txModule.Message
	OsmosisMsgJoinPool *gammTypes.MsgJoinPool
	Address            string
	TokenOut           sdk.Coin
	TokensIn           []sdk.Coin // joins can be done with multiple tokens in
	Claim              *sdk.Coin  // option claim
}

type WrapperMsgExitSwapShareAmountIn struct {
	txModule.Message
	OsmosisMsgExitSwapShareAmountIn *gammTypes.MsgExitSwapShareAmountIn
	Address                         string
	TokenOut                        sdk.Coin
	TokenIn                         sdk.Coin
}

// Same as WrapperMsgExitSwapShareAmountIn but with different handlers.
// This is due to the Osmosis SDK emitting different Events (chain upgrades).
type WrapperMsgExitSwapShareAmountIn2 struct {
	txModule.Message
	OsmosisMsgExitSwapShareAmountIn *gammTypes.MsgExitSwapShareAmountIn
	Address                         string
	TokensOut                       sdk.Coins
	TokenSwaps                      []tokenSwap
	TokenIn                         sdk.Coin
}

type tokenSwap struct {
	TokenSwappedIn  sdk.Coin
	TokenSwappedOut sdk.Coin
}

type WrapperMsgExitSwapExternAmountOut struct {
	txModule.Message
	OsmosisMsgExitSwapExternAmountOut *gammTypes.MsgExitSwapExternAmountOut
	Address                           string
	TokenOut                          sdk.Coin
	TokenIn                           sdk.Coin
}

type WrapperMsgExitPool struct {
	txModule.Message
	OsmosisMsgExitPool *gammTypes.MsgExitPool
	Address            string
	TokensOutOfPool    []sdk.Coin // exits can received multiple tokens out
	TokenIntoPool      sdk.Coin
}

type WrapperMsgCreatePool struct {
	txModule.Message
	OsmosisMsgCreatePool *osmosisOldTypes.MsgCreatePool
	CoinsSpent           []sdk.Coin
	GammCoinsReceived    sdk.Coin
	OtherCoinsReceived   []coinReceived // e.g. from claims module (airdrops)
}

type WrapperMsgCreateBalancerPool struct {
	txModule.Message
	OsmosisMsgCreateBalancerPool *osmosisOldTypes.MsgCreateBalancerPool
	CoinsSpent                   []sdk.Coin
	GammCoinsReceived            sdk.Coin
	OtherCoinsReceived           []coinReceived // e.g. from claims module (airdrops)
}

type coinReceived struct {
	sender       string
	coinReceived sdk.Coin
}

type WrapperMsgCreatePool2 struct {
	WrapperMsgCreatePool
}

type WrapperOldMsgCreateBalancerPool struct {
	txModule.Message
	OsmosisMsgCreateBalancerPool *gammBalancerPoolModelsTypes.MsgCreateBalancerPool
	CoinsSpent                   []sdk.Coin
	GammCoinsReceived            sdk.Coin
	OtherCoinsReceived           []coinReceived // e.g. from claims module (airdrops)
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

	return fmt.Sprintf("MsgSwapExactAmountIn: %s swapped in %s and received %s",
		sf.Address, tokenSwappedIn, tokenSwappedOut)
}

func (sf *WrapperMsgSwapExactAmountIn2) String() string {
	return sf.WrapperMsgSwapExactAmountIn.String()
}

func (sf *WrapperMsgSwapExactAmountIn3) String() string {
	return sf.WrapperMsgSwapExactAmountIn.String()
}

func (sf *WrapperMsgSwapExactAmountIn4) String() string {
	return sf.WrapperMsgSwapExactAmountIn.String()
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
	return fmt.Sprintf("MsgSwapExactAmountOut: %s swapped in %s and received %s",
		sf.Address, tokenSwappedIn, tokenSwappedOut)
}

func (sf *WrapperMsgJoinSwapExternAmountIn) String() string {
	var tokenSwappedOut string
	var tokenSwappedIn string
	if !sf.TokenOut.IsNil() {
		tokenSwappedOut = sf.TokenOut.String()
	}
	if !sf.TokenIn.IsNil() {
		tokenSwappedIn = sf.TokenIn.String()
	}
	return fmt.Sprintf("MsgJoinSwapExternAmountIn: %s joined with %s and received %s",
		sf.Address, tokenSwappedIn, tokenSwappedOut)
}

func (sf *WrapperMsgJoinSwapExternAmountIn2) String() string {
	return sf.WrapperMsgJoinSwapExternAmountIn.String()
}

func (sf *WrapperMsgJoinSwapShareAmountOut) String() string {
	var tokenSwappedOut string
	var tokenSwappedIn string
	if !sf.TokenOut.IsNil() {
		tokenSwappedOut = sf.TokenOut.String()
	}
	if !sf.TokenIn.IsNil() {
		tokenSwappedIn = sf.TokenIn.String()
	}
	return fmt.Sprintf("MsgJoinSwapShareAmountOut: %s joined with %s and received %s",
		sf.Address, tokenSwappedIn, tokenSwappedOut)
}

func (sf *WrapperMsgJoinPool) String() string {
	var tokenOut string
	var tokensIn []string
	if !(len(sf.TokensIn) == 0) {
		for _, v := range sf.TokensIn {
			tokensIn = append(tokensIn, v.String())
		}
	}
	if !sf.TokenOut.IsNil() {
		tokenOut = sf.TokenOut.String()
	}
	return fmt.Sprintf("MsgJoinPool: %s joined pool with %s and received %s",
		sf.Address, strings.Join(tokensIn, ", "), tokenOut)
}

func (sf *WrapperMsgCreatePool) String() string {
	var tokensIn []string
	if !(len(sf.OsmosisMsgCreatePool.PoolAssets) == 0) {
		for _, v := range sf.OsmosisMsgCreatePool.PoolAssets {
			tokensIn = append(tokensIn, v.Token.String())
		}
	}
	return fmt.Sprintf("MsgCreatePool: %s created pool with %s",
		sf.OsmosisMsgCreatePool.Sender, strings.Join(tokensIn, ", "))
}

func (sf *WrapperMsgCreatePool2) String() string {
	return sf.WrapperMsgCreatePool.String()
}

func (sf *WrapperMsgCreateBalancerPool) String() string {
	var tokensIn []string
	if !(len(sf.OsmosisMsgCreateBalancerPool.PoolAssets) == 0) {
		for _, v := range sf.OsmosisMsgCreateBalancerPool.PoolAssets {
			tokensIn = append(tokensIn, v.Token.String())
		}
	}
	return fmt.Sprintf("MsgCreateBalancerPool: %s created pool with %s",
		sf.OsmosisMsgCreateBalancerPool.Sender, strings.Join(tokensIn, ", "))
}

func (sf *WrapperOldMsgCreateBalancerPool) String() string {
	var tokensIn []string
	if !(len(sf.OsmosisMsgCreateBalancerPool.PoolAssets) == 0) {
		for _, v := range sf.OsmosisMsgCreateBalancerPool.PoolAssets {
			tokensIn = append(tokensIn, v.Token.String())
		}
	}
	return fmt.Sprintf("MsgCreateBalancerPool: %s created pool with %s",
		sf.OsmosisMsgCreateBalancerPool.Sender, strings.Join(tokensIn, ", "))
}

func (sf *WrapperMsgExitSwapShareAmountIn) String() string {
	var tokenSwappedOut string
	var tokenSwappedIn string
	if !sf.TokenOut.IsNil() {
		tokenSwappedOut = sf.TokenOut.String()
	}
	if !sf.TokenIn.IsNil() {
		tokenSwappedIn = sf.TokenIn.String()
	}
	return fmt.Sprintf("MsgMsgExitSwapShareAmountIn: %s exited with %s and received %s",
		sf.Address, tokenSwappedIn, tokenSwappedOut)
}

func (sf *WrapperMsgExitSwapShareAmountIn2) String() string {
	var tokenSwappedOut string
	var tokenSwappedIn string

	var postExitTokenSwaps []string
	var postExitTokenSwapsRepr string

	if !sf.TokensOut.Empty() {
		tokenSwappedOut = sf.TokensOut.String()
	}
	if !sf.TokenIn.IsNil() {
		tokenSwappedIn = sf.TokenIn.String()
	}

	if !(len(sf.TokenSwaps) == 0) {
		for _, swap := range sf.TokenSwaps {
			postExitTokenSwaps = append(postExitTokenSwaps, fmt.Sprintf("%s for %s", swap.TokenSwappedIn.String(), swap.TokenSwappedOut.String()))
		}

		postExitTokenSwapsRepr = strings.Join(postExitTokenSwaps, ", ")
	}

	return fmt.Sprintf("MsgMsgExitSwapShareAmountIn: %s exited with %s and received %s, then swapped %s",
		sf.Address, tokenSwappedIn, tokenSwappedOut, postExitTokenSwapsRepr)
}

func (sf *WrapperMsgExitSwapExternAmountOut) String() string {
	var tokenSwappedOut string
	var tokenSwappedIn string
	if !sf.TokenOut.IsNil() {
		tokenSwappedOut = sf.TokenOut.String()
	}
	if !sf.TokenIn.IsNil() {
		tokenSwappedIn = sf.TokenIn.String()
	}
	return fmt.Sprintf("WrapperMsgExitSwapExternAmountOut: %s exited with %s and received %s",
		sf.Address, tokenSwappedIn, tokenSwappedOut)
}

func (sf *WrapperMsgExitPool) String() string {
	var tokensOut []string
	var tokenIn string
	if !(len(sf.TokensOutOfPool) == 0) {
		for _, v := range sf.TokensOutOfPool {
			tokensOut = append(tokensOut, v.String())
		}
	}
	if !sf.TokenIntoPool.IsNil() {
		tokenIn = sf.TokenIntoPool.String()
	}
	return fmt.Sprintf("MsgExitPool: %s exited pool with %s and received %s",
		sf.Address, tokenIn, strings.Join(tokensOut, ", "))
}

func (sf *WrapperMsgExitPool2) String() string {
	return sf.WrapperMsgExitPool.String()
}

func (sf *WrapperMsgSwapExactAmountIn) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgSwapExactAmountIn = msg.(*gammTypes.MsgSwapExactAmountIn)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// The attribute in the log message that shows you the tokens swapped
	tokensSwappedEvt := txModule.GetEventWithType(gammTypes.TypeEvtTokenSwapped, log)
	if tokensSwappedEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// Address of whoever initiated the swap. Will be both sender/receiver.
	senderReceiver, err := txModule.GetValueForAttribute("sender", tokensSwappedEvt)
	if err != nil {
		return err
	}

	if senderReceiver == "" {
		fmt.Println("Error getting sender.")
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderReceiver

	// This gets the first token swapped in (if there are multiple pools we do not care about intermediates)
	tokenInStr, err := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensIn, tokensSwappedEvt)
	if err != nil {
		return err
	}

	tokenIn, err := sdk.ParseCoinNormalized(tokenInStr)
	if err != nil {
		fmt.Println("Error parsing coins in. Err: ", err)
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenIn = tokenIn

	// This gets the last token swapped out (if there are multiple pools we do not care about intermediates)
	tokenOutStr := txModule.GetLastValueForAttribute(gammTypes.AttributeKeyTokensOut, tokensSwappedEvt)
	tokenOut, err := sdk.ParseCoinNormalized(tokenOutStr)
	if err != nil {
		fmt.Println("Error parsing coins out. Err: ", err)
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenOut = tokenOut

	return err
}

// Handles an OLDER (now defunct) swap on Osmosis mainnet (osmosis-1).
// Example TX hash: EA5C6AB8E3084D933F3E005A952A362DFD13DC79003DC2BC9E247920FCDFDD34
func (sf *WrapperMsgSwapExactAmountIn2) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgSwapExactAmountIn = msg.(*gammTypes.MsgSwapExactAmountIn)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// The attribute in the log message that shows you the tokens swapped
	tokensSwappedEvt := txModule.GetEventWithType(EventTypeClaim, log)
	if tokensSwappedEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// Address of whoever initiated the swap. Will be both sender/receiver.
	senderReceiver, err := txModule.GetValueForAttribute("sender", tokensSwappedEvt)
	if err != nil {
		return err
	}

	if senderReceiver == "" {
		fmt.Println("Error getting sender.")
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderReceiver

	// First token swapped in (if there are multiple pools we do not care about intermediates)
	sf.TokenIn = sf.OsmosisMsgSwapExactAmountIn.TokenIn

	// This gets the last token swapped out (if there are multiple pools we do not care about intermediates)
	tokenOutStr := txModule.GetLastValueForAttribute(EventAttributeAmount, tokensSwappedEvt)
	tokenOut, err := sdk.ParseCoinNormalized(tokenOutStr)
	if err != nil {
		fmt.Println("Error parsing coins out. Err: ", err)
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenOut = tokenOut

	return err
}

// Handles an OLDER (now defunct) swap on Osmosis mainnet (osmosis-1).
// Example TX hash: BC8384F767F48EDDF65646EC136518DE00B59A8E2793AABFE7563C62B39A59AE
func (sf *WrapperMsgSwapExactAmountIn3) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgSwapExactAmountIn = msg.(*gammTypes.MsgSwapExactAmountIn)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// The attribute in the log message that shows you the tokens transferred
	tokensTransferredEvt := txModule.GetEventWithType(EventTypeTransfer, log)
	if tokensTransferredEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	msgSender := sf.OsmosisMsgSwapExactAmountIn.Sender
	msgTokensIn := sf.OsmosisMsgSwapExactAmountIn.TokenIn

	// First sender should be the address that conducted the swap
	firstSender := txModule.GetNthValueForAttribute("sender", 1, tokensTransferredEvt)
	firstAmount := txModule.GetNthValueForAttribute(EventAttributeAmount, 1, tokensTransferredEvt)

	if firstSender != msgSender {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	} else if firstAmount != msgTokensIn.String() {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	sf.Address = msgSender

	secondReceiver := txModule.GetNthValueForAttribute("recipient", 2, tokensTransferredEvt)
	secondAmount := txModule.GetNthValueForAttribute(EventAttributeAmount, 2, tokensTransferredEvt)

	if secondReceiver != msgSender {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	amountReceived, err := sdk.ParseCoinNormalized(secondAmount)
	if err != nil {
		return err
	}

	outDenom := sf.OsmosisMsgSwapExactAmountIn.Routes[len(sf.OsmosisMsgSwapExactAmountIn.Routes)-1].TokenOutDenom
	if amountReceived.Denom != outDenom {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("amountReceived.Denom != outDenom. Log: %+v", log)}
	}

	// Address of whoever initiated the swap. Will be both sender/receiver.
	senderReceiver, err := txModule.GetValueForAttribute("sender", tokensTransferredEvt)
	if err != nil {
		return err
	}

	if senderReceiver == "" {
		fmt.Println("Error getting sender.")
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// First token swapped in (if there are multiple pools we do not care about intermediates)
	sf.TokenIn = sf.OsmosisMsgSwapExactAmountIn.TokenIn
	sf.TokenOut = amountReceived

	return err
}

// Handles an OLDER (now defunct) swap on Osmosis mainnet (osmosis-1).
// Example TX hash: BB954377AB50F8EF204123DC8B101B7CB597153C0B8372166BC28ABDAA262516
func (sf *WrapperMsgSwapExactAmountIn4) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgSwapExactAmountIn = msg.(*gammTypes.MsgSwapExactAmountIn)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// The attribute in the log message that shows you the tokens transferred
	tokensTransferredEvt := txModule.GetEventWithType(EventTypeTransfer, log)
	if tokensTransferredEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	msgSender := sf.OsmosisMsgSwapExactAmountIn.Sender
	msgTokensIn := sf.OsmosisMsgSwapExactAmountIn.TokenIn

	// First sender should be the address that conducted the swap
	firstSender := txModule.GetNthValueForAttribute("sender", 1, tokensTransferredEvt)
	firstAmount := txModule.GetNthValueForAttribute(EventAttributeAmount, 1, tokensTransferredEvt)

	if firstSender != msgSender {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	} else if firstAmount != msgTokensIn.String() {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	sf.Address = msgSender

	lastReceiver := txModule.GetLastValueForAttribute("recipient", tokensTransferredEvt)
	lastAmount := txModule.GetLastValueForAttribute(EventAttributeAmount, tokensTransferredEvt)

	if lastReceiver != msgSender {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	amountReceived, err := sdk.ParseCoinNormalized(lastAmount)
	if err != nil {
		return err
	}

	outDenom := sf.OsmosisMsgSwapExactAmountIn.Routes[len(sf.OsmosisMsgSwapExactAmountIn.Routes)-1].TokenOutDenom
	if amountReceived.Denom != outDenom {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("amountReceived.Denom != outDenom. Log: %+v", log)}
	}

	// Address of whoever initiated the swap. Will be both sender/receiver.
	senderReceiver, err := txModule.GetValueForAttribute("sender", tokensTransferredEvt)
	if err != nil {
		return err
	}

	if senderReceiver == "" {
		fmt.Println("Error getting sender.")
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// First token swapped in (if there are multiple pools we do not care about intermediates)
	sf.TokenIn = sf.OsmosisMsgSwapExactAmountIn.TokenIn
	sf.TokenOut = amountReceived

	return err
}

func (sf *WrapperMsgSwapExactAmountOut) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgSwapExactAmountOut = msg.(*gammTypes.MsgSwapExactAmountOut)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// The attribute in the log message that shows you the tokens swapped
	tokensSwappedEvt := txModule.GetEventWithType(gammTypes.TypeEvtTokenSwapped, log)

	if tokensSwappedEvt == nil {
		transferEvt := txModule.GetEventWithType("transfer", log)

		tokenInDenom := sf.OsmosisMsgSwapExactAmountOut.TokenInDenom()

		for _, evt := range transferEvt.Attributes {
			// Get the first amount that matches the token in denom
			if evt.Key == EventAttributeAmount {
				tokenIn, err := sdk.ParseCoinNormalized(evt.Value)
				if err != nil {
					return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
				}

				if tokenIn.Denom == tokenInDenom {
					sf.TokenIn = tokenIn
					break
				}

			}
		}

		senderReceiver, err := txModule.GetValueForAttribute("sender", transferEvt)
		if err != nil {
			return err
		}

		if senderReceiver == "" {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
		}
		sf.Address = senderReceiver

		tokenOutStr := txModule.GetLastValueForAttribute(EventAttributeAmount, transferEvt)
		tokenOut, err := sdk.ParseCoinNormalized(tokenOutStr)
		if err != nil {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
		}

		sf.TokenOut = tokenOut

		if sf.TokenIn.IsNil() {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
		}
	} else {
		// This gets the first token swapped in (if there are multiple pools we do not care about intermediates)
		tokenInStr, err := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensIn, tokensSwappedEvt)
		if err != nil {
			return err
		}

		tokenIn, err := sdk.ParseCoinNormalized(tokenInStr)
		if err != nil {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
		}
		sf.TokenIn = tokenIn

		// Address of whoever initiated the swap. Will be both sender/receiver.
		senderReceiver, err := txModule.GetValueForAttribute("sender", tokensSwappedEvt)
		if err != nil {
			return err
		}

		if senderReceiver == "" {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
		}
		sf.Address = senderReceiver

		// This gets the last token swapped out (if there are multiple pools we do not care about intermediates)
		tokenOutStr := txModule.GetLastValueForAttribute(gammTypes.AttributeKeyTokensOut, tokensSwappedEvt)
		tokenOut, err := sdk.ParseCoinNormalized(tokenOutStr)
		if err != nil {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
		}
		sf.TokenOut = tokenOut

	}

	return nil
}

func (sf *WrapperMsgJoinSwapExternAmountIn) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgJoinSwapExternAmountIn = msg.(*gammTypes.MsgJoinSwapExternAmountIn)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// The attribute in the log message that shows you the received GAMM tokens from the pool
	coinbaseEvt := txModule.GetEventWithType("coinbase", log)
	if coinbaseEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// This gets the amount of GAMM tokens received
	gammTokenInStr, err := txModule.GetValueForAttribute(EventAttributeAmount, coinbaseEvt)
	if err != nil {
		return err
	}

	if !strings.Contains(gammTokenInStr, "gamm") {
		fmt.Println("Gamm token in string must contain gamm")
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	gammTokenIn, err := sdk.ParseCoinNormalized(gammTokenInStr)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenOut = gammTokenIn

	// we can pull the token in directly from the Osmosis Message
	sf.TokenIn = sf.OsmosisMsgJoinSwapExternAmountIn.TokenIn

	// Address of whoever initiated the join
	poolJoinedEvent := txModule.GetEventWithType(gammTypes.TypeEvtPoolJoined, log)
	if poolJoinedEvent == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// Address of whoever initiated the join.
	senderAddress, err := txModule.GetValueForAttribute("sender", poolJoinedEvent)
	if err != nil {
		return err
	}

	if senderAddress == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderAddress

	return err
}

func (sf *WrapperMsgJoinSwapExternAmountIn2) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgJoinSwapExternAmountIn = msg.(*gammTypes.MsgJoinSwapExternAmountIn)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// we can pull the token and sender in directly from the Osmosis Message
	sf.TokenIn = sf.OsmosisMsgJoinSwapExternAmountIn.TokenIn
	sf.Address = sf.OsmosisMsgJoinSwapExternAmountIn.Sender

	transferEvt := txModule.GetEventWithType("transfer", log)
	gammOutString := ""
	// Loop backwards to find the GAMM out string
	for i := len(transferEvt.Attributes) - 1; i >= 0; i-- {
		attr := transferEvt.Attributes[i]
		if attr.Key == EventAttributeAmount && strings.Contains(attr.Value, "gamm") && strings.HasSuffix(attr.Value, fmt.Sprintf("/%d", sf.OsmosisMsgJoinSwapExternAmountIn.PoolId)) {
			// Verify the recipient of the gamm output is the sender of the message
			if i-2 > -1 && transferEvt.Attributes[i-2].Key == "recipient" && transferEvt.Attributes[i-2].Value == sf.Address {
				gammOutString = attr.Value
			}
		}
	}

	if gammOutString == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	gammTokenOut, err := sdk.ParseCoinNormalized(gammOutString)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenOut = gammTokenOut

	return err
}

func (sf *WrapperMsgJoinSwapShareAmountOut) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgJoinSwapShareAmountOut = msg.(*gammTypes.MsgJoinSwapShareAmountOut)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// The attribute in the log message that shows you the received GAMM tokens from the pool
	coinbaseEvt := txModule.GetEventWithType("coinbase", log)
	if coinbaseEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// This gets the amount of GAMM tokens received
	gammTokenInStr, err := txModule.GetValueForAttribute(EventAttributeAmount, coinbaseEvt)
	if err != nil {
		return err
	}

	if !strings.Contains(gammTokenInStr, "gamm") {
		fmt.Println("Gamm token in string must contain gamm")
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	gammTokenIn, err := sdk.ParseCoinNormalized(gammTokenInStr)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenOut = gammTokenIn

	// Address of whoever initiated the join
	poolJoinedEvent := txModule.GetEventWithType(gammTypes.TypeEvtPoolJoined, log)
	if poolJoinedEvent == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// Address of whoever initiated the join.
	senderAddress, err := txModule.GetValueForAttribute("sender", poolJoinedEvent)
	if err != nil {
		return err
	}

	if senderAddress == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderAddress

	tokenIn, err := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensIn, poolJoinedEvent)
	if err != nil {
		return err
	}

	if tokenIn == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenIn, err = sdk.ParseCoinNormalized(tokenIn)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	return err
}

func (sf *WrapperMsgJoinSwapShareAmountOut2) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgJoinSwapShareAmountOut = msg.(*gammTypes.MsgJoinSwapShareAmountOut)
	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// The attribute in the log message that shows you the received GAMM tokens from the pool
	transferEvt := txModule.GetEventWithType("transfer", log)
	if transferEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	gammOutString := ""
	// Loop backwards to find the GAMM out string
	for i := len(transferEvt.Attributes) - 1; i >= 0; i-- {
		attr := transferEvt.Attributes[i]
		if attr.Key == EventAttributeAmount && strings.Contains(attr.Value, "gamm") && strings.HasSuffix(attr.Value, fmt.Sprintf("/%d", sf.OsmosisMsgJoinSwapShareAmountOut.PoolId)) {
			// Verify the recipient of the gamm output is the sender of the message
			if i-2 > -1 && transferEvt.Attributes[i-2].Key == "recipient" && transferEvt.Attributes[i-2].Value == sf.OsmosisMsgJoinSwapShareAmountOut.Sender {
				gammOutString = attr.Value
			}
		}
	}

	if gammOutString == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// This gets the amount of GAMM tokens received
	gammTokenOut, err := sdk.ParseCoinNormalized(gammOutString)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenOut = gammTokenOut

	// Address of whoever initiated the join
	poolJoinedEvent := txModule.GetEventWithType(gammTypes.TypeEvtPoolJoined, log)
	if poolJoinedEvent == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// Address of whoever initiated the join.
	senderAddress, err := txModule.GetValueForAttribute("sender", poolJoinedEvent)
	if err != nil {
		return err
	}

	if senderAddress == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderAddress

	tokenIn, err := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensIn, poolJoinedEvent)
	if err != nil {
		return err
	}

	if tokenIn == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenIn, err = sdk.ParseCoinNormalized(tokenIn)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	return err
}

func (sf *WrapperMsgJoinPool) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgJoinPool = msg.(*gammTypes.MsgJoinPool)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// The attribute in the log message that shows you the received GAMM tokens from the pool
	transferEvt := txModule.GetEventWithType(bankTypes.EventTypeTransfer, log)
	if transferEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// This gets the amount of GAMM tokens received and claim (if needed)
	var gammTokenOutStr string
	if strings.Contains(fmt.Sprint(log), "claim") {
		// This gets the amount of the claim
		claimStr := txModule.GetLastValueForAttribute(EventAttributeAmount, transferEvt)
		claimTokenOut, err := sdk.ParseCoinNormalized(claimStr)
		if err != nil {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
		}
		sf.Claim = &claimTokenOut

		gammTokenOutStr = txModule.GetNthValueForAttribute(EventAttributeAmount, 2, transferEvt)
	} else {
		gammTokenOutStr = txModule.GetLastValueForAttribute(EventAttributeAmount, transferEvt)
	}
	if !strings.Contains(gammTokenOutStr, "gamm") {
		fmt.Println(gammTokenOutStr)
		fmt.Println("Gamm token out string must contain gamm")
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	gammTokenOut, err := sdk.ParseCoinNormalized(gammTokenOutStr)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenOut = gammTokenOut

	// Address of whoever initiated the join
	poolJoinedEvent := txModule.GetEventWithType(gammTypes.TypeEvtPoolJoined, log)
	if poolJoinedEvent == nil {
		// If the message doesn't have the pool_joined event, we can parse the transaction event to extract the
		// amounts of the tokens they transferred in.

		// find the multi-coin amount that also is not gamms... those must be the coins transferred in
		var tokensIn string
		var sender string
		for i, attr := range transferEvt.Attributes {
			if attr.Key == EventAttributeAmount && strings.Contains(attr.Value, ",") {
				tokensIn = attr.Value
				// If we haven't found the sender yet, it will be the address that sent this non-gamm token
				if i > 0 && transferEvt.Attributes[i-1].Key == "sender" && sf.OsmosisMsgJoinPool.Sender == transferEvt.Attributes[i-1].Value {
					sender = transferEvt.Attributes[i-1].Value
				}
				break
			}
		}
		// if either of these methods failed to get info, give up and return an error
		if len(tokensIn) == 0 || sender == "" {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
		}

		// if we got what we needed, set the correct values and return successfully
		sf.Address = sender
		sf.TokensIn, err = sdk.ParseCoinsNormalized(tokensIn)
		if err != nil {
			fmt.Println("Error parsing coins")
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
		}
		return nil
	}

	// Address of whoever initiated the join.
	senderAddress, err := txModule.GetValueForAttribute("sender", poolJoinedEvent)
	if err != nil {
		return err
	}

	if senderAddress == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderAddress

	// String value for the tokens in, which can be multiple
	tokensInString, err := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensIn, poolJoinedEvent)
	if err != nil {
		return err
	}

	if tokensInString == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokensIn, err = sdk.ParseCoinsNormalized(tokensInString)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	return err
}

func (sf *WrapperMsgCreatePool) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgCreatePool = msg.(*osmosisOldTypes.MsgCreatePool)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	coinSpentEvents := txModule.GetEventsWithType(bankTypes.EventTypeCoinSpent, log)
	if len(coinSpentEvents) == 0 {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	coinsSpent := txModule.GetCoinsSpent(sf.OsmosisMsgCreatePool.Sender, coinSpentEvents)

	if len(coinsSpent) < 2 {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("invalid number of coins spent: %+v", log)}
	}

	sf.CoinsSpent = []sdk.Coin{}
	for _, coin := range coinsSpent {
		t, err := sdk.ParseCoinNormalized(coin)
		if err != nil {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
		}

		sf.CoinsSpent = append(sf.CoinsSpent, t)
	}

	coinReceivedEvents := txModule.GetEventsWithType(bankTypes.EventTypeCoinReceived, log)
	if len(coinReceivedEvents) == 0 {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	coinsReceived := txModule.GetCoinsReceived(sf.OsmosisMsgCreatePool.Sender, coinReceivedEvents)

	gammCoinsReceived := []string{}

	for _, coin := range coinsReceived {
		if strings.Contains(coin, "gamm/pool") {
			gammCoinsReceived = append(gammCoinsReceived, coin)
		} else {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("unexpected non-gamm/pool coin received: %+v", log)}
		}
	}

	if len(gammCoinsReceived) != 1 {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("invalid number of coins received: %+v", log)}
	}

	var err error
	sf.GammCoinsReceived, err = sdk.ParseCoinNormalized(gammCoinsReceived[0])
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	return err
}

func (sf *WrapperMsgCreatePool2) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgCreatePool = msg.(*osmosisOldTypes.MsgCreatePool)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	transferEvents := txModule.GetEventsWithType(bankTypes.EventTypeTransfer, log)
	if len(transferEvents) == 0 {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	coinsSpent := []string{}
	coinsReceived := []coinReceived{}

	for _, transfer := range transferEvents {
		parsedTransfer, err := txModule.ParseTransferEvent(transfer)
		if err != nil {
			return err
		}

		for _, curr := range parsedTransfer {
			coins := strings.Split(curr.Amount, ",")
			if curr.Recipient == sf.OsmosisMsgCreatePool.Sender {
				for _, coin := range coins {
					t, err := sdk.ParseCoinNormalized(coin)
					if err != nil {
						return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
					}

					coinsReceived = append(coinsReceived, coinReceived{
						sender:       curr.Sender,
						coinReceived: t,
					})
				}
			} else if curr.Sender == sf.OsmosisMsgCreatePool.Sender {
				coinsSpent = append(coinsSpent, coins...)
			}
		}
	}

	if len(coinsSpent) < 2 {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("invalid number of coins spent: %+v", log)}
	}

	sf.CoinsSpent = []sdk.Coin{}
	for _, coin := range coinsSpent {
		t, err := sdk.ParseCoinNormalized(coin)
		if err != nil {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
		}

		sf.CoinsSpent = append(sf.CoinsSpent, t)
	}

	gammCoinsReceived := []sdk.Coin{}
	sf.OtherCoinsReceived = []coinReceived{}

	for _, coin := range coinsReceived {
		if strings.Contains(coin.coinReceived.Denom, "gamm/pool") {
			gammCoinsReceived = append(gammCoinsReceived, coin.coinReceived)
		} else {
			sf.OtherCoinsReceived = append(sf.OtherCoinsReceived, coin)
		}
	}

	if len(gammCoinsReceived) != 1 {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("invalid number of coins received: %+v", log)}
	}

	sf.GammCoinsReceived = gammCoinsReceived[0]

	return nil
}

func (sf *WrapperMsgCreateBalancerPool) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgCreateBalancerPool = msg.(*osmosisOldTypes.MsgCreateBalancerPool)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	coinSpentEvents := txModule.GetEventsWithType(bankTypes.EventTypeCoinSpent, log)
	if len(coinSpentEvents) == 0 {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	coinsSpent := txModule.GetCoinsSpent(sf.OsmosisMsgCreateBalancerPool.Sender, coinSpentEvents)

	if len(coinsSpent) < 2 {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("invalid number of coins spent: %+v", log)}
	}

	sf.CoinsSpent = []sdk.Coin{}
	for _, coin := range coinsSpent {
		t, err := sdk.ParseCoinNormalized(coin)
		if err != nil {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
		}

		sf.CoinsSpent = append(sf.CoinsSpent, t)
	}

	coinReceivedEvents := txModule.GetEventsWithType(bankTypes.EventTypeCoinReceived, log)
	if len(coinReceivedEvents) == 0 {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	coinsReceived := txModule.GetCoinsReceived(sf.OsmosisMsgCreateBalancerPool.Sender, coinReceivedEvents)

	gammCoinsReceived := []string{}

	for _, coin := range coinsReceived {
		if strings.Contains(coin, "gamm/pool") {
			gammCoinsReceived = append(gammCoinsReceived, coin)
		} else {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("unexpected non-gamm/pool coin received: %+v", log)}
		}
	}

	if len(gammCoinsReceived) != 1 {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("invalid number of coins received: %+v", log)}
	}

	var err error
	sf.GammCoinsReceived, err = sdk.ParseCoinNormalized(gammCoinsReceived[0])
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	return err
}

func (sf *WrapperOldMsgCreateBalancerPool) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgCreateBalancerPool = msg.(*gammBalancerPoolModelsTypes.MsgCreateBalancerPool)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	coinSpentEvents := txModule.GetEventsWithType(bankTypes.EventTypeCoinSpent, log)
	if len(coinSpentEvents) == 0 {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	coinsSpent := txModule.GetCoinsSpent(sf.OsmosisMsgCreateBalancerPool.Sender, coinSpentEvents)

	if len(coinsSpent) < 2 {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("invalid number of coins spent: %+v", log)}
	}

	sf.CoinsSpent = []sdk.Coin{}
	for _, coin := range coinsSpent {
		t, err := sdk.ParseCoinNormalized(coin)
		if err != nil {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
		}

		sf.CoinsSpent = append(sf.CoinsSpent, t)
	}

	coinReceivedEvents := txModule.GetEventsWithType(bankTypes.EventTypeCoinReceived, log)
	if len(coinReceivedEvents) == 0 {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	coinsReceived := txModule.GetCoinsReceived(sf.OsmosisMsgCreateBalancerPool.Sender, coinReceivedEvents)

	gammCoinsReceived := []string{}

	for _, coin := range coinsReceived {
		if strings.Contains(coin, "gamm/pool") {
			gammCoinsReceived = append(gammCoinsReceived, coin)
		} else {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("unexpected non-gamm/pool coin received: %+v", log)}
		}
	}

	if len(gammCoinsReceived) != 1 {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("invalid number of coins received: %+v", log)}
	}

	var err error
	sf.GammCoinsReceived, err = sdk.ParseCoinNormalized(gammCoinsReceived[0])
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	return nil
}

func (sf *WrapperMsgExitSwapShareAmountIn) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgExitSwapShareAmountIn = msg.(*gammTypes.MsgExitSwapShareAmountIn)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// The attribute in the log message that shows you the burned GAMM tokens sent to the pool
	burnEvt := txModule.GetEventWithType("burn", log)
	if burnEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// This gets the amount of GAMM exited with
	gammTokenInStr, err := txModule.GetValueForAttribute(EventAttributeAmount, burnEvt)
	if err != nil {
		return err
	}

	if !strings.Contains(gammTokenInStr, "gamm") {
		fmt.Println("Gamm token in string must contain gamm")
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	gammTokenIn, err := sdk.ParseCoinNormalized(gammTokenInStr)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenIn = gammTokenIn

	// Address of whoever initiated the exit
	poolExitedEvent := txModule.GetEventWithType(gammTypes.TypeEvtPoolExited, log)
	if poolExitedEvent == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// Address of whoever initiated the exit.
	senderAddress, err := txModule.GetValueForAttribute("sender", poolExitedEvent)
	if err != nil {
		return err
	}

	if senderAddress == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderAddress

	tokenOut, err := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensOut, poolExitedEvent)
	if err != nil {
		return err
	}

	if tokenOut == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenOut, err = sdk.ParseCoinNormalized(tokenOut)

	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	return err
}

func (sf *WrapperMsgExitSwapShareAmountIn2) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgExitSwapShareAmountIn = msg.(*gammTypes.MsgExitSwapShareAmountIn)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// The attribute in the log message that shows you the burned GAMM tokens sent to the pool
	burnEvt := txModule.GetEventWithType("burn", log)
	if burnEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// This gets the amount of GAMM exited with
	gammTokenInStr, err := txModule.GetValueForAttribute(EventAttributeAmount, burnEvt)
	if err != nil {
		return err
	}

	if !strings.Contains(gammTokenInStr, "gamm") {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	gammTokenIn, err := sdk.ParseCoinNormalized(gammTokenInStr)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenIn = gammTokenIn

	// Address of whoever initiated the exit
	poolExitedEvent := txModule.GetEventWithType(gammTypes.TypeEvtPoolExited, log)
	if poolExitedEvent == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// Address of whoever initiated the exit.
	senderAddress, err := txModule.GetValueForAttribute("sender", poolExitedEvent)
	if err != nil {
		return err
	}

	if senderAddress == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderAddress

	tokensOut, err := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensOut, poolExitedEvent)
	if err != nil {
		return err
	}

	if tokensOut == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	multiTokensOut, err := sdk.ParseCoinsNormalized(tokensOut)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	sf.TokensOut = multiTokensOut

	// The token swapped events contain the final amount of tokens out in this tx
	tokenSwappedEvents := txModule.GetAllEventsWithType(gammTypes.TypeEvtTokenSwapped, log)

	// This is to handle multi-token pool exit swaps
	for i := range tokenSwappedEvents {
		tokenSwappedIn, err := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensIn, &tokenSwappedEvents[i])
		if err != nil {
			return err
		}

		tokenSwappedOut, err := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensOut, &tokenSwappedEvents[i])
		if err != nil {
			return err
		}

		parsedTokensSwappedIn, err := sdk.ParseCoinNormalized(tokenSwappedIn)
		if err != nil {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
		}

		parsedTokensSwappedOut, err := sdk.ParseCoinNormalized(tokenSwappedOut)
		if err != nil {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
		}

		tokenSwap := tokenSwap{TokenSwappedIn: parsedTokensSwappedIn, TokenSwappedOut: parsedTokensSwappedOut}

		sf.TokenSwaps = append(sf.TokenSwaps, tokenSwap)
	}

	return err
}

func (sf *WrapperMsgExitSwapExternAmountOut) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgExitSwapExternAmountOut = msg.(*gammTypes.MsgExitSwapExternAmountOut)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// The attribute in the log message that shows you the burned GAMM tokens sent to the pool
	burnEvt := txModule.GetEventWithType("burn", log)
	if burnEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// This gets the amount of GAMM exited with
	gammTokenInStr, err := txModule.GetValueForAttribute(EventAttributeAmount, burnEvt)
	if err != nil {
		return err
	}

	if !strings.Contains(gammTokenInStr, "gamm") {
		fmt.Println("Gamm token in string must contain gamm")
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	gammTokenIn, err := sdk.ParseCoinNormalized(gammTokenInStr)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenIn = gammTokenIn

	// Address of whoever initiated the exit
	poolExitedEvent := txModule.GetEventWithType(gammTypes.TypeEvtPoolExited, log)
	if poolExitedEvent == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// Address of whoever initiated the exit.
	senderAddress, err := txModule.GetValueForAttribute("sender", poolExitedEvent)
	if err != nil {
		return err
	}

	if senderAddress == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderAddress

	tokenOut, err := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensOut, poolExitedEvent)
	if err != nil {
		return err
	}

	if tokenOut == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenOut, err = sdk.ParseCoinNormalized(tokenOut)

	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	return err
}

func (sf *WrapperMsgExitPool2) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgExitPool = msg.(*gammTypes.MsgExitPool)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// The attribute in the log message that shows you the sent GAMM tokens during the exit
	transferEvt := txModule.GetEventWithType(bankTypes.EventTypeTransfer, log)
	if transferEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// This gets the amount of GAMM tokens sent
	gammTokenOutStr := txModule.GetLastValueForAttribute(EventAttributeAmount, transferEvt)
	if !strings.Contains(gammTokenOutStr, "gamm") {
		fmt.Println("Gamm token out string must contain gamm")
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	gammTokenOut, err := sdk.ParseCoinNormalized(gammTokenOutStr)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenIntoPool = gammTokenOut

	if sf.OsmosisMsgExitPool.Sender == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = sf.OsmosisMsgExitPool.Sender

	// The first attribute in the event should have a key 'recipient', and a value with the Msg sender's address (whoever is exiting the pool)
	senderAddr := txModule.GetNthValueForAttribute("recipient", 1, transferEvt)
	if senderAddr != sf.Address {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v, senderAddr != sf.Address", log)}
	}

	// String value for the tokens out, which can be multiple
	tokensOutString := txModule.GetNthValueForAttribute(EventAttributeAmount, 1, transferEvt)
	if tokensOutString == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	sf.TokensOutOfPool, err = sdk.ParseCoinsNormalized(tokensOutString)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	return err
}

func (sf *WrapperMsgExitPool) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgExitPool = msg.(*gammTypes.MsgExitPool)

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// The attribute in the log message that shows you the sent GAMM tokens during the exit
	transverEvt := txModule.GetEventWithType(bankTypes.EventTypeTransfer, log)
	if transverEvt == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// This gets the amount of GAMM tokens sent
	gammTokenOutStr := txModule.GetLastValueForAttribute(EventAttributeAmount, transverEvt)
	if !strings.Contains(gammTokenOutStr, "gamm") {
		fmt.Println("Gamm token out string must contain gamm")
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	gammTokenOut, err := sdk.ParseCoinNormalized(gammTokenOutStr)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenIntoPool = gammTokenOut

	// Address of whoever initiated the exit
	poolExitedEvent := txModule.GetEventWithType(gammTypes.TypeEvtPoolExited, log)
	if poolExitedEvent == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// Address of whoever initiated the exit.
	senderAddress, err := txModule.GetValueForAttribute("sender", poolExitedEvent)
	if err != nil {
		return err
	}

	if senderAddress == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderAddress

	// String value for the tokens in, which can be multiple
	tokensOutString, err := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensOut, poolExitedEvent)
	if err != nil {
		return err
	}

	if tokensOutString == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	sf.TokensOutOfPool, err = sdk.ParseCoinsNormalized(tokensOutString)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	return err
}

type ArbitrageTx struct {
	TokenIn   sdk.Coin
	TokenOut  sdk.Coin
	BlockTime time.Time
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

func (sf *WrapperMsgSwapExactAmountIn2) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	return sf.WrapperMsgSwapExactAmountIn.ParseRelevantData()
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

func (sf *WrapperMsgJoinSwapExternAmountIn) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
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

func (sf *WrapperMsgJoinSwapShareAmountOut) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
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

func (sf *WrapperMsgJoinPool) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	// need to make a relevant data block for all Tokens sent to the pool since JoinPool can use 1 or both tokens used in the pool
	relevantData := make([]parsingTypes.MessageRelevantInformation, len(sf.TokensIn))

	// figure out how many gams per token
	nthGamms, remainderGamms := calcNthGams(sf.TokenOut.Amount.BigInt(), len(sf.TokensIn))
	for i, v := range sf.TokensIn {
		// split received tokens across entry so we receive GAMM tokens for both exchanges
		// each swap will get 1 nth of the gams until the last one which will get the remainder
		if i != len(sf.TokensIn)-1 {
			relevantData[i] = parsingTypes.MessageRelevantInformation{
				AmountSent:           v.Amount.BigInt(),
				DenominationSent:     v.Denom,
				AmountReceived:       nthGamms,
				DenominationReceived: sf.TokenOut.Denom,
				SenderAddress:        sf.Address,
				ReceiverAddress:      sf.Address,
			}
		} else {
			relevantData[i] = parsingTypes.MessageRelevantInformation{
				AmountSent:           v.Amount.BigInt(),
				DenominationSent:     v.Denom,
				AmountReceived:       remainderGamms,
				DenominationReceived: sf.TokenOut.Denom,
				SenderAddress:        sf.Address,
				ReceiverAddress:      sf.Address,
			}
		}
	}

	// handle claim if there is one
	if sf.Claim != nil {
		relevantData = append(relevantData, parsingTypes.MessageRelevantInformation{
			ReceiverAddress:      sf.Address,
			AmountReceived:       sf.Claim.Amount.BigInt(),
			DenominationReceived: sf.Claim.Denom,
		})
	}

	return relevantData
}

func (sf *WrapperMsgCreatePool2) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	return sf.WrapperMsgCreatePool.ParseRelevantData()
}

func (sf *WrapperMsgCreatePool) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	// need to make a relevant data block for all Tokens sent to the pool on creation
	relevantData := make([]parsingTypes.MessageRelevantInformation, len(sf.CoinsSpent)+len(sf.OtherCoinsReceived))

	// figure out how many gams per token
	nthGamms, remainderGamms := calcNthGams(sf.GammCoinsReceived.Amount.BigInt(), len(sf.CoinsSpent))
	for i, v := range sf.CoinsSpent {
		// split received tokens across entry so we receive GAMM tokens for both exchanges
		// each swap will get 1 nth of the gams until the last one which will get the remainder
		if i != len(sf.CoinsSpent)-1 {
			relevantData[i] = parsingTypes.MessageRelevantInformation{
				AmountSent:           v.Amount.BigInt(),
				DenominationSent:     v.Denom,
				AmountReceived:       nthGamms,
				DenominationReceived: sf.GammCoinsReceived.Denom,
				SenderAddress:        sf.OsmosisMsgCreatePool.Sender,
				ReceiverAddress:      sf.OsmosisMsgCreatePool.Sender,
			}
		} else {
			relevantData[i] = parsingTypes.MessageRelevantInformation{
				AmountSent:           v.Amount.BigInt(),
				DenominationSent:     v.Denom,
				AmountReceived:       remainderGamms,
				DenominationReceived: sf.GammCoinsReceived.Denom,
				SenderAddress:        sf.OsmosisMsgCreatePool.Sender,
				ReceiverAddress:      sf.OsmosisMsgCreatePool.Sender,
			}
		}
	}

	i := len(sf.CoinsSpent)
	for _, c := range sf.OtherCoinsReceived {
		relevantData[i] = parsingTypes.MessageRelevantInformation{
			AmountSent:           c.coinReceived.Amount.BigInt(),
			DenominationSent:     c.coinReceived.Denom,
			AmountReceived:       c.coinReceived.Amount.BigInt(),
			DenominationReceived: c.coinReceived.Denom,
			SenderAddress:        c.sender,
			ReceiverAddress:      sf.OsmosisMsgCreatePool.Sender,
		}
		i++
	}

	return relevantData
}

func (sf *WrapperMsgCreateBalancerPool) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	// need to make a relevant data block for all Tokens sent to the pool on creation
	relevantData := make([]parsingTypes.MessageRelevantInformation, len(sf.CoinsSpent)+len(sf.OtherCoinsReceived))

	// figure out how many gams per token
	nthGamms, remainderGamms := calcNthGams(sf.GammCoinsReceived.Amount.BigInt(), len(sf.CoinsSpent))
	for i, v := range sf.CoinsSpent {
		// split received tokens across entry so we receive GAMM tokens for both exchanges
		// each swap will get 1 nth of the gams until the last one which will get the remainder
		if i != len(sf.CoinsSpent)-1 {
			relevantData[i] = parsingTypes.MessageRelevantInformation{
				AmountSent:           v.Amount.BigInt(),
				DenominationSent:     v.Denom,
				AmountReceived:       nthGamms,
				DenominationReceived: sf.GammCoinsReceived.Denom,
				SenderAddress:        sf.OsmosisMsgCreateBalancerPool.Sender,
				ReceiverAddress:      sf.OsmosisMsgCreateBalancerPool.Sender,
			}
		} else {
			relevantData[i] = parsingTypes.MessageRelevantInformation{
				AmountSent:           v.Amount.BigInt(),
				DenominationSent:     v.Denom,
				AmountReceived:       remainderGamms,
				DenominationReceived: sf.GammCoinsReceived.Denom,
				SenderAddress:        sf.OsmosisMsgCreateBalancerPool.Sender,
				ReceiverAddress:      sf.OsmosisMsgCreateBalancerPool.Sender,
			}
		}
	}

	i := len(sf.CoinsSpent)
	for _, c := range sf.OtherCoinsReceived {
		relevantData[i] = parsingTypes.MessageRelevantInformation{
			AmountSent:           c.coinReceived.Amount.BigInt(),
			DenominationSent:     c.coinReceived.Denom,
			AmountReceived:       c.coinReceived.Amount.BigInt(),
			DenominationReceived: c.coinReceived.Denom,
			SenderAddress:        c.sender,
			ReceiverAddress:      sf.OsmosisMsgCreateBalancerPool.Sender,
		}
		i++
	}

	return relevantData
}

func (sf *WrapperOldMsgCreateBalancerPool) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	// need to make a relevant data block for all Tokens sent to the pool on creation
	relevantData := make([]parsingTypes.MessageRelevantInformation, len(sf.CoinsSpent)+len(sf.OtherCoinsReceived))

	// figure out how many gams per token
	nthGamms, remainderGamms := calcNthGams(sf.GammCoinsReceived.Amount.BigInt(), len(sf.CoinsSpent))
	for i, v := range sf.CoinsSpent {
		// split received tokens across entry so we receive GAMM tokens for both exchanges
		// each swap will get 1 nth of the gams until the last one which will get the remainder
		if i != len(sf.CoinsSpent)-1 {
			relevantData[i] = parsingTypes.MessageRelevantInformation{
				AmountSent:           v.Amount.BigInt(),
				DenominationSent:     v.Denom,
				AmountReceived:       nthGamms,
				DenominationReceived: sf.GammCoinsReceived.Denom,
				SenderAddress:        sf.OsmosisMsgCreateBalancerPool.Sender,
				ReceiverAddress:      sf.OsmosisMsgCreateBalancerPool.Sender,
			}
		} else {
			relevantData[i] = parsingTypes.MessageRelevantInformation{
				AmountSent:           v.Amount.BigInt(),
				DenominationSent:     v.Denom,
				AmountReceived:       remainderGamms,
				DenominationReceived: sf.GammCoinsReceived.Denom,
				SenderAddress:        sf.OsmosisMsgCreateBalancerPool.Sender,
				ReceiverAddress:      sf.OsmosisMsgCreateBalancerPool.Sender,
			}
		}
	}

	i := len(sf.CoinsSpent)
	for _, c := range sf.OtherCoinsReceived {
		relevantData[i] = parsingTypes.MessageRelevantInformation{
			AmountSent:           c.coinReceived.Amount.BigInt(),
			DenominationSent:     c.coinReceived.Denom,
			AmountReceived:       c.coinReceived.Amount.BigInt(),
			DenominationReceived: c.coinReceived.Denom,
			SenderAddress:        c.sender,
			ReceiverAddress:      sf.OsmosisMsgCreateBalancerPool.Sender,
		}
		i++
	}
	return relevantData
}

func (sf *WrapperMsgExitSwapShareAmountIn) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
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

func (sf *WrapperMsgExitSwapShareAmountIn2) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	relevantData := make([]parsingTypes.MessageRelevantInformation, len(sf.TokensOut))

	// figure out how many gams per token
	nthGamms, remainderGamms := calcNthGams(sf.TokenIn.Amount.BigInt(), len(sf.TokensOut))

	// Handle the pool exit
	for i, v := range sf.TokensOut {
		// split received tokens across entry so we receive GAMM tokens for both exchanges
		// each swap will get 1 nth of the gams until the last one which will get the remainder
		if i != len(sf.TokensOut)-1 {
			relevantData[i] = parsingTypes.MessageRelevantInformation{
				AmountSent:           nthGamms,
				DenominationSent:     sf.TokenIn.Denom,
				AmountReceived:       v.Amount.BigInt(),
				DenominationReceived: v.Denom,
				SenderAddress:        sf.Address,
				ReceiverAddress:      sf.Address,
			}
		} else {
			relevantData[i] = parsingTypes.MessageRelevantInformation{
				AmountSent:           remainderGamms,
				DenominationSent:     sf.TokenIn.Denom,
				AmountReceived:       v.Amount.BigInt(),
				DenominationReceived: v.Denom,
				SenderAddress:        sf.Address,
				ReceiverAddress:      sf.Address,
			}
		}
	}

	// Handle the post exit swap event
	for _, tokensSwapped := range sf.TokenSwaps {
		relevantData = append(relevantData, parsingTypes.MessageRelevantInformation{
			AmountSent:           tokensSwapped.TokenSwappedIn.Amount.BigInt(),
			DenominationSent:     tokensSwapped.TokenSwappedIn.Denom,
			AmountReceived:       tokensSwapped.TokenSwappedOut.Amount.BigInt(),
			DenominationReceived: tokensSwapped.TokenSwappedOut.Denom,
			SenderAddress:        sf.Address,
			ReceiverAddress:      sf.Address,
		})
	}

	return relevantData
}

func (sf *WrapperMsgExitSwapExternAmountOut) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
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

func (sf *WrapperMsgExitPool) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	// need to make a relevant data block for all Tokens received from the pool since ExitPool can receive 1 or both tokens used in the pool
	relevantData := make([]parsingTypes.MessageRelevantInformation, len(sf.TokensOutOfPool))

	// figure out how many gams per token
	nthGamms, remainderGamms := calcNthGams(sf.TokenIntoPool.Amount.BigInt(), len(sf.TokensOutOfPool))
	for i, v := range sf.TokensOutOfPool {
		// only add received tokens to the first entry so we dont duplicate received GAMM tokens
		if i != len(sf.TokensOutOfPool)-1 {
			relevantData[i] = parsingTypes.MessageRelevantInformation{
				AmountSent:           nthGamms,
				DenominationSent:     sf.TokenIntoPool.Denom,
				AmountReceived:       v.Amount.BigInt(),
				DenominationReceived: v.Denom,
				SenderAddress:        sf.Address,
				ReceiverAddress:      sf.Address,
			}
		} else {
			relevantData[i] = parsingTypes.MessageRelevantInformation{
				AmountSent:           remainderGamms,
				DenominationSent:     sf.TokenIntoPool.Denom,
				AmountReceived:       v.Amount.BigInt(),
				DenominationReceived: v.Denom,
				SenderAddress:        sf.Address,
				ReceiverAddress:      sf.Address,
			}
		}
	}

	return relevantData
}

func (sf *WrapperMsgExitPool2) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	return sf.WrapperMsgExitPool.ParseRelevantData()
}

func calcNthGams(totalGamms *big.Int, numSwaps int) (*big.Int, *big.Int) {
	// figure out how many gamms per token
	var nthGamms big.Int
	nthGamms.Div(totalGamms, big.NewInt(int64(numSwaps)))

	// figure out how many gamms will remain for the last swap
	var remainderGamms big.Int
	remainderGamms.Mod(totalGamms, &nthGamms)
	remainderGamms.Add(&nthGamms, &remainderGamms)
	return &nthGamms, &remainderGamms
}
