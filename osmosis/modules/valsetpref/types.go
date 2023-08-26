package valsetpref

import (
	"fmt"
	"strings"

	parsingTypes "github.com/DefiantLabs/cosmos-indexer/cosmos/modules"
	txModule "github.com/DefiantLabs/cosmos-indexer/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-indexer/util"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	valsetPrefTypes "github.com/osmosis-labs/osmosis/v16/x/valset-pref/types"
)

const (
	MsgSetValidatorSetPreference  = "/osmosis.valsetpref.v1beta1.MsgSetValidatorSetPreference"
	MsgDelegateToValidatorSet     = "/osmosis.valsetpref.v1beta1.MsgDelegateToValidatorSet"
	MsgUndelegateFromValidatorSet = "/osmosis.valsetpref.v1beta1.MsgUndelegateFromValidatorSet"
	MsgRedelegateValidatorSet     = "/osmosis.valsetpref.v1beta1.MsgRedelegateValidatorSet"
)

type WrapperMsgDelegateToValidatorSet struct {
	txModule.Message
	OsmosisMsgDelegateToValidatorSet *valsetPrefTypes.MsgDelegateToValidatorSet
	DelegatorAddress                 string
	RewardsOut                       sdk.Coins
}

func (sf *WrapperMsgDelegateToValidatorSet) String() string {
	var tokensSent []string
	if !(len(sf.RewardsOut) == 0) {
		for _, v := range sf.RewardsOut {
			tokensSent = append(tokensSent, v.String())
		}
		return fmt.Sprintf("WrapperMsgDelegateToValidatorSet: %s received rewards %s",
			sf.DelegatorAddress, strings.Join(tokensSent, ", "))
	} else {
		return fmt.Sprintf("WrapperMsgDelegateToValidatorSet: %s did not withdraw rewards",
			sf.DelegatorAddress)
	}

}

// HandleMsg: Handle type checking for MsgFundCommunityPool
func (sf *WrapperMsgDelegateToValidatorSet) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.OsmosisMsgDelegateToValidatorSet = msg.(*valsetPrefTypes.MsgDelegateToValidatorSet)
	sf.DelegatorAddress = sf.OsmosisMsgDelegateToValidatorSet.Delegator

	// Confirm that the action listed in the message log matches the Message type
	validLog := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !validLog {
		return util.ReturnInvalidLog(msgType, log)
	}

	// The attribute in the log message that shows you the delegator rewards auto-received
	receiveEvent := txModule.GetEventsWithType(bankTypes.EventTypeCoinReceived, log)
	if receiveEvent != nil {
		delegaterCoinsReceivedStrings := txModule.GetCoinsReceived(sf.DelegatorAddress, receiveEvent)

		for _, coinString := range delegaterCoinsReceivedStrings {
			coins, err := sdk.ParseCoinsNormalized(coinString)
			if err != nil {
				return err
			}
			sf.RewardsOut = append(sf.RewardsOut, coins...)
		}

	}

	return nil
}

func (sf *WrapperMsgDelegateToValidatorSet) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	relevantData := make([]parsingTypes.MessageRelevantInformation, 0)

	for _, token := range sf.RewardsOut {
		if token.Amount.IsPositive() {
			relevantData = append(relevantData, parsingTypes.MessageRelevantInformation{
				AmountReceived:       token.Amount.BigInt(),
				DenominationReceived: token.Denom,
				SenderAddress:        sf.DelegatorAddress,
			})
		}
	}
	return relevantData
}
