package liquidity

import (
	"fmt"
	"strings"

	parsingTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules"
	txModule "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	liquidityTypes "github.com/DefiantLabs/lens/tendermint/x/liquidity/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	MsgCreatePool          = "/tendermint.liquidity.v1beta1.MsgCreatePool"
	MsgDepositWithinBatch  = "/tendermint.liquidity.v1beta1.MsgDepositWithinBatch"
	MsgWithdrawWithinBatch = "/tendermint.liquidity.v1beta1.MsgWithdrawWithinBatch"
	MsgSwapWithinBatch     = "/tendermint.liquidity.v1beta1.MsgSwapWithinBatch" // nolint:gosec
)

type WrapperMsgDepositWithinBatch struct {
	txModule.Message
	TendermintMsgDepositWithinBatch *liquidityTypes.MsgDepositWithinBatch
	Address                         string
	TokensOut                       sdk.Coins
}

type WrapperMsgWithdrawWithinBatch struct {
	txModule.Message
	TendermintMsgDepositWithinBatch *liquidityTypes.MsgWithdrawWithinBatch
	Address                         string
	TokenIn                         sdk.Coin
}

func (sf *WrapperMsgDepositWithinBatch) String() string {
	var tokensOut []string

	for _, v := range sf.TokensOut {
		tokensOut = append(tokensOut, v.String())
	}

	return fmt.Sprintf("MsgDepositWithinBatch: %s deposited %s\n",
		sf.Address, strings.Join(tokensOut, ", "))
}

func (sf *WrapperMsgWithdrawWithinBatch) String() string {
	return fmt.Sprintf("MsgWithdrawWithinBatch: %s withdrew %s\n",
		sf.Address, sf.TokenIn.String())
}

func (sf *WrapperMsgDepositWithinBatch) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.TendermintMsgDepositWithinBatch = msg.(*liquidityTypes.MsgDepositWithinBatch)

	sf.Address = sf.TendermintMsgDepositWithinBatch.DepositorAddress
	sf.TokensOut = sf.TendermintMsgDepositWithinBatch.DepositCoins
	return nil
}

func (sf *WrapperMsgWithdrawWithinBatch) HandleMsg(msgType string, msg sdk.Msg, log *txModule.LogMessage) error {
	sf.Type = msgType
	sf.TendermintMsgDepositWithinBatch = msg.(*liquidityTypes.MsgWithdrawWithinBatch)

	sf.Address = sf.TendermintMsgDepositWithinBatch.WithdrawerAddress
	sf.TokenIn = sf.TendermintMsgDepositWithinBatch.PoolCoin

	return nil
}

func (sf *WrapperMsgDepositWithinBatch) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	var relevantData = make([]parsingTypes.MessageRelevantInformation, len(sf.TokensOut))
	for i, v := range sf.TokensOut {
		relevantData[i] = parsingTypes.MessageRelevantInformation{
			AmountSent:       v.Amount.BigInt(),
			DenominationSent: v.Denom,
			SenderAddress:    sf.Address,
		}
	}
	return relevantData
}

func (sf *WrapperMsgWithdrawWithinBatch) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	var relevantData = make([]parsingTypes.MessageRelevantInformation, 1)
	relevantData[0] = parsingTypes.MessageRelevantInformation{
		AmountReceived:       sf.TokenIn.Amount.BigInt(),
		DenominationReceived: sf.TokenIn.Denom,
		SenderAddress:        sf.Address,
	}
	return relevantData
}
