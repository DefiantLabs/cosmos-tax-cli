package gamm

import (
	"math/big"

	txModule "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
