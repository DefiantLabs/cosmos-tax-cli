package gamm

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	parsingTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules"
	txModule "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-tax-cli/util"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	gammTypes "github.com/osmosis-labs/osmosis/v9/x/gamm/types"
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
)

type WrapperMsgSwapExactAmountIn struct {
	txModule.Message
	OsmosisMsgSwapExactAmountIn *gammTypes.MsgSwapExactAmountIn
	Address                     string
	TokenOut                    sdk.Coin
	TokenIn                     sdk.Coin
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

type WrapperMsgJoinSwapShareAmountOut struct {
	txModule.Message
	OsmosisMsgJoinSwapShareAmountOut *gammTypes.MsgJoinSwapShareAmountOut
	Address                          string
	TokenOut                         sdk.Coin
	TokenIn                          sdk.Coin
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

func (sf *WrapperMsgSwapExactAmountOut) String() string {
	var tokenSwappedOut string
	var tokenSwappedIn string
	if !sf.TokenOut.IsNil() {
		tokenSwappedOut = sf.TokenOut.String()
	}
	if !sf.TokenIn.IsNil() {
		tokenSwappedIn = sf.TokenIn.String()
	}
	return fmt.Sprintf("MsgSwapExactAmountOut: %s swapped in %s and received %s\n",
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
	return fmt.Sprintf("MsgJoinSwapExternAmountIn: %s joined with %s and received %s\n",
		sf.Address, tokenSwappedIn, tokenSwappedOut)
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
	return fmt.Sprintf("MsgJoinSwapShareAmountOut: %s joined with %s and received %s\n",
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
	return fmt.Sprintf("MsgJoinPool: %s joined pool with %s and received %s\n",
		sf.Address, strings.Join(tokensIn, ", "), tokenOut)
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
	return fmt.Sprintf("MsgMsgExitSwapShareAmountIn: %s exited with %s and received %s\n",
		sf.Address, tokenSwappedIn, tokenSwappedOut)
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
	return fmt.Sprintf("WrapperMsgExitSwapExternAmountOut: %s exited with %s and received %s\n",
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
	return fmt.Sprintf("MsgExitPool: %s exited pool with %s and received %s\n",
		sf.Address, tokenIn, strings.Join(tokensOut, ", "))
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
		fmt.Println("Error getting event type.")
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// Address of whoever initiated the swap. Will be both sender/receiver.
	senderReceiver := txModule.GetValueForAttribute("sender", tokensSwappedEvt)
	if senderReceiver == "" {
		fmt.Println("Error getting sender.")
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderReceiver

	// This gets the first token swapped in (if there are multiple pools we do not care about intermediates)
	tokenInStr := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensIn, tokensSwappedEvt)
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
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	// Address of whoever initiated the swap. Will be both sender/receiver.
	senderReceiver := txModule.GetValueForAttribute("sender", tokensSwappedEvt)
	if senderReceiver == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderReceiver

	// This gets the first token swapped in (if there are multiple pools we do not care about intermediates)
	tokenInStr := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensIn, tokensSwappedEvt)
	tokenIn, err := sdk.ParseCoinNormalized(tokenInStr)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenIn = tokenIn

	// This gets the last token swapped out (if there are multiple pools we do not care about intermediates)
	tokenOutStr := txModule.GetLastValueForAttribute(gammTypes.AttributeKeyTokensOut, tokensSwappedEvt)
	tokenOut, err := sdk.ParseCoinNormalized(tokenOutStr)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenOut = tokenOut

	return err
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
	gammTokenInStr := txModule.GetValueForAttribute("amount", coinbaseEvt)
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
	senderAddress := txModule.GetValueForAttribute("sender", poolJoinedEvent)
	if senderAddress == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderAddress

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
	gammTokenInStr := txModule.GetValueForAttribute("amount", coinbaseEvt)
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
	senderAddress := txModule.GetValueForAttribute("sender", poolJoinedEvent)
	if senderAddress == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderAddress

	tokenIn := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensIn, poolJoinedEvent)
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
		claimStr := txModule.GetLastValueForAttribute("amount", transferEvt)
		claimTokenOut, err := sdk.ParseCoinNormalized(claimStr)
		if err != nil {
			return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
		}
		sf.Claim = &claimTokenOut

		gammTokenOutStr = txModule.GetNthValueForAttribute("amount", 2, transferEvt)
	} else {
		gammTokenOutStr = txModule.GetLastValueForAttribute("amount", transferEvt)
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
			if attr.Key == "amount" && !strings.Contains(attr.Value, "gamm/pool") && strings.Contains(attr.Value, ",") {
				tokensIn = attr.Value
				// If we haven't found the sender yet, it will be the address that sent this non-gamm token
				if i > 0 && transferEvt.Attributes[i-1].Key == "sender" {
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
	senderAddress := txModule.GetValueForAttribute("sender", poolJoinedEvent)
	if senderAddress == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderAddress

	// String value for the tokens in, which can be multiple
	tokensInString := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensIn, poolJoinedEvent)
	if tokensInString == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokensIn, err = sdk.ParseCoinsNormalized(tokensInString)
	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	return err
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
	gammTokenInStr := txModule.GetValueForAttribute("amount", burnEvt)
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
	senderAddress := txModule.GetValueForAttribute("sender", poolExitedEvent)
	if senderAddress == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderAddress

	tokenOut := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensOut, poolExitedEvent)
	if tokenOut == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenOut, err = sdk.ParseCoinNormalized(tokenOut)

	if err != nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
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
	gammTokenInStr := txModule.GetValueForAttribute("amount", burnEvt)
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
	senderAddress := txModule.GetValueForAttribute("sender", poolExitedEvent)
	if senderAddress == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderAddress

	tokenOut := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensOut, poolExitedEvent)
	if tokenOut == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.TokenOut, err = sdk.ParseCoinNormalized(tokenOut)

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
	gammTokenOutStr := txModule.GetLastValueForAttribute("amount", transverEvt)
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
	senderAddress := txModule.GetValueForAttribute("sender", poolExitedEvent)
	if senderAddress == "" {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}
	sf.Address = senderAddress

	// String value for the tokens in, which can be multiple
	tokensOutString := txModule.GetValueForAttribute(gammTypes.AttributeKeyTokensOut, poolExitedEvent)
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
	var relevantData = make([]parsingTypes.MessageRelevantInformation, 1)
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
	var relevantData = make([]parsingTypes.MessageRelevantInformation, 1)
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
	var relevantData = make([]parsingTypes.MessageRelevantInformation, 1)
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
	var relevantData = make([]parsingTypes.MessageRelevantInformation, 1)
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
	var relevantData = make([]parsingTypes.MessageRelevantInformation, len(sf.TokensIn))

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

func (sf *WrapperMsgExitSwapShareAmountIn) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	var relevantData = make([]parsingTypes.MessageRelevantInformation, 1)
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

func (sf *WrapperMsgExitSwapExternAmountOut) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	var relevantData = make([]parsingTypes.MessageRelevantInformation, 1)
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
	var relevantData = make([]parsingTypes.MessageRelevantInformation, len(sf.TokensOutOfPool))

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
