package rpc

import (
	"fmt"
	"math"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	lensClient "github.com/DefiantLabs/lens/client"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
)

type CustomBlockResults struct {
	Height                int64                     `json:"height"`
	TxsResults            []*abci.ExecTxResult      `json:"txs_results"`
	BeginBlockEvents      []abci.Event              `json:"begin_block_events"`
	EndBlockEvents        []abci.Event              `json:"end_block_events"`
	ValidatorUpdates      []abci.ValidatorUpdate    `json:"validator_updates"`
	ConsensusParamUpdates *cmtproto.ConsensusParams `json:"consensus_param_updates"`
	FinalizeBlockEvents   []abci.Event              `json:"finalize_block_events"`
}

func NormalizeCustomBlockResults(blockResults *coretypes.ResultBlockResults) (*CustomBlockResults, error) {
	customBlockResults := &CustomBlockResults{
		Height:                blockResults.Height,
		TxsResults:            blockResults.TxsResults,
		ValidatorUpdates:      blockResults.ValidatorUpdates,
		ConsensusParamUpdates: blockResults.ConsensusParamUpdates,
		FinalizeBlockEvents:   blockResults.FinalizeBlockEvents,
	}

	if len(blockResults.FinalizeBlockEvents) != 0 {
		beginBlockEvents := []abci.Event{}
		endBlockEvents := []abci.Event{}

		for _, event := range blockResults.FinalizeBlockEvents {
			eventAttrs := []abci.EventAttribute{}
			isBeginBlock := false
			isEndBlock := false
			for _, attr := range event.Attributes {
				if attr.Key == "mode" {
					if attr.Value == "BeginBlock" {
						isBeginBlock = true
					} else if attr.Value == "EndBlock" {
						isEndBlock = true
					}
				} else {
					eventAttrs = append(eventAttrs, attr)
				}
			}

			switch {
			case isBeginBlock && isEndBlock:
				return nil, fmt.Errorf("finalize block event has both BeginBlock and EndBlock mode")
			case !isBeginBlock && !isEndBlock:
				return nil, fmt.Errorf("finalize block event has neither BeginBlock nor EndBlock mode")
			case isBeginBlock:
				beginBlockEvents = append(beginBlockEvents, abci.Event{Type: event.Type, Attributes: eventAttrs})
			case isEndBlock:
				endBlockEvents = append(endBlockEvents, abci.Event{Type: event.Type, Attributes: eventAttrs})
			}
		}

		customBlockResults.BeginBlockEvents = append(customBlockResults.BeginBlockEvents, beginBlockEvents...)
		customBlockResults.EndBlockEvents = append(customBlockResults.EndBlockEvents, endBlockEvents...)
	}

	return customBlockResults, nil
}

func GetBlockResultWithRetry(cl *lensClient.ChainClient, height int64, retryMaxAttempts int64, retryMaxWaitSeconds uint64) (*CustomBlockResults, error) {
	if retryMaxAttempts == 0 {
		return GetBlockResultRPC(cl, height)
	}

	if retryMaxWaitSeconds < 2 {
		retryMaxWaitSeconds = 2
	}

	var attempts int64
	maxRetryTime := time.Duration(retryMaxWaitSeconds) * time.Second
	if maxRetryTime < 0 {
		config.Log.Warn("Detected maxRetryTime overflow, setting time to sane maximum of 30s")
		maxRetryTime = 30 * time.Second
	}

	currentBackoffDuration, maxReached := GetBackoffDurationForAttempts(attempts, maxRetryTime)

	for {
		resp, err := GetBlockResultRPC(cl, height)
		attempts++
		if err != nil && (retryMaxAttempts < 0 || (attempts <= retryMaxAttempts)) {
			config.Log.Error("Error getting RPC response, backing off and trying again", err)
			config.Log.Debugf("Attempt %d with wait time %+v", attempts, currentBackoffDuration)
			time.Sleep(currentBackoffDuration)

			// guard against overflow
			if !maxReached {
				currentBackoffDuration, maxReached = GetBackoffDurationForAttempts(attempts, maxRetryTime)
			}

		} else {
			if err != nil {
				config.Log.Error("Error getting RPC response, reached max retry attempts")
			}
			return resp, err
		}
	}
}

func GetBackoffDurationForAttempts(numAttempts int64, maxRetryTime time.Duration) (time.Duration, bool) {
	backoffBase := 1.5
	backoffDuration := time.Duration(math.Pow(backoffBase, float64(numAttempts)) * float64(time.Second))

	maxReached := false
	if backoffDuration > maxRetryTime || backoffDuration < 0 {
		maxReached = true
		backoffDuration = maxRetryTime
	}

	return backoffDuration, maxReached
}
