package distribution

import (
	"fmt"

	txModule "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"

	parsingTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules"

	stdTypes "github.com/cosmos/cosmos-sdk/types"
	distTypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

var IsMsgFundCommunityPool = map[string]bool{
	"/cosmos.distribution.v1beta1.MsgFundCommunityPool": true,
}

type WrapperMsgFundCommunityPool struct {
	txModule.Message
	CosmosMsgFundCommunityPool *distTypes.MsgFundCommunityPool
	Depositor                  string
	Funds                      stdTypes.Coins
}

func (sf *WrapperMsgFundCommunityPool) String() string {
	coinsReceivedString := sf.CosmosMsgFundCommunityPool.Amount.String()
	depositorAddress := sf.CosmosMsgFundCommunityPool.Depositor

	return fmt.Sprintf("MsgFundCommunityPool: Depositor %s gave %s\n",
		depositorAddress, coinsReceivedString)
}

//HandleMsg: Handle type checking for MsgFundCommunityPool
func (sf *WrapperMsgFundCommunityPool) HandleMsg(msgType string, msg stdTypes.Msg, log *txModule.TxLogMessage) error {
	sf.Type = msgType
	sf.CosmosMsgFundCommunityPool = msg.(*distTypes.MsgFundCommunityPool)

	//Confirm that the action listed in the message log matches the Message type
	valid_log := txModule.IsMessageActionEquals(sf.GetType(), log)
	if !valid_log {
		return &txModule.MessageLogFormatError{MessageType: msgType, Log: fmt.Sprintf("%+v", log)}
	}

	//Funds sent and sender address are pulled from the parsed Cosmos Msg
	sf.Depositor = sf.CosmosMsgFundCommunityPool.Depositor
	sf.Funds = sf.CosmosMsgFundCommunityPool.Amount

	return nil
}

func (sf *WrapperMsgFundCommunityPool) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	var relevantData []parsingTypes.MessageRelevantInformation = make([]parsingTypes.MessageRelevantInformation, len(sf.Funds))

	for i, v := range sf.Funds {
		relevantData[i].AmountSent = v.Amount.BigInt()
		relevantData[i].DenominationSent = v.Denom
		relevantData[i].SenderAddress = sf.Depositor
	}

	return relevantData
}
