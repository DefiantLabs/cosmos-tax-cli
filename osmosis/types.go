package osmosis

import (
	"net/http"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type RewardEpoch struct {
	EpochBlockHeight int64
	Indexed          bool
	Error            error
}

type OsmosisRewards struct {
	EpochBlockHeight int64
	Address          string
	Coins            sdk.Coins
}

type Result struct {
	Data Data
}

type Data struct {
	Value EventDataNewBlockHeader
}

type EventDataNewBlockHeader struct {
	Header Header `json:"header"`
	NumTxs string `json:"num_txs"` // Number of txs in a block
}

type Header struct {
	// basic block info
	ChainID string    `json:"chain_id"`
	Height  string    `json:"height"`
	Time    time.Time `json:"time"`
}

type TendermintNewBlockHeader struct {
	Result Result
}

type URIClient struct {
	Address    string
	Client     *http.Client
	AuthHeader string
}
