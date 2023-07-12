package concentratedliquidity

import (
	"errors"
	"fmt"
	"strings"

	parsingTypes "github.com/DefiantLabs/cosmos-indexer/cosmos/modules"
	txModule "github.com/DefiantLabs/cosmos-indexer/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-indexer/util"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clTypes "github.com/osmosis-labs/osmosis/v16/x/concentrated-liquidity/types"
)

const (
	MsgCreatePosition = "/osmosis.concentratedliquidity.v1beta1.MsgCreatePosition"
)

type WrapperMsgCreatePosition struct {
	txModule.Message
	OsmosisMsgCreatePosition *clTypes.MsgCreatePosition
	TokensSent               sdk.Coins
	Address                  string
}

func (sf *WrapperMsgCreatePosition) String() string {
	var tokensSent []string
	if !(len(sf.TokensSent) == 0) {
		for _, v := range sf.TokensSent {
			tokensSent = append(tokensSent, v.String())
		}
	}
	return fmt.Sprintf("MsgCreatePosition: %s created position by sending %s",
		sf.Address, strings.Join(tokensSent, ", "))
}

func (sf *WrapperMsgCreatePosition) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgCreatePosition = msg.(*clTypes.MsgCreatePosition)

	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	sf.TokensSent = sf.OsmosisMsgCreatePosition.TokensProvided

	// Need to get actual amounts from event emissions
	createPositionEvent := txModule.GetEventWithType("create_position", log)
	if createPositionEvent == nil {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	amount0String := txModule.GetValueForAttribute("amount0", createPositionEvent)
	amount1String := txModule.GetValueForAttribute("amount1", createPositionEvent)

	if amount0String != "" {
		amount0, ok := sdk.NewIntFromString(amount0String)

		if !ok {
			return errors.New("error parsing amount0")
		}
		sf.TokensSent[0].Amount = amount0
	} else {
		sf.TokensSent[0].Amount = sdk.NewIntFromUint64(0)
	}

	if amount1String != "" && len(sf.TokensSent) > 1 {
		amount1, ok := sdk.NewIntFromString(amount1String)

		if !ok {
			return errors.New("error parsing amount1")
		}
		sf.TokensSent[1].Amount = amount1
	} else if len(sf.TokensSent) > 1 {
		sf.TokensSent[1].Amount = sdk.NewIntFromUint64(0)
	}

	sf.Address = sf.OsmosisMsgCreatePosition.Sender

	return nil
}

func (sf *WrapperMsgCreatePosition) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	relevantData := make([]parsingTypes.MessageRelevantInformation, 0)

	for _, token := range sf.TokensSent {
		if token.Amount.IsPositive() {
			relevantData = append(relevantData, parsingTypes.MessageRelevantInformation{
				AmountSent:       token.Amount.BigInt(),
				DenominationSent: token.Denom,
				SenderAddress:    sf.Address,
			})
		}
	}
	return relevantData
}
