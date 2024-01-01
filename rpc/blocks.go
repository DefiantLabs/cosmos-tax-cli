package rpc

import (
	"math"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	lensClient "github.com/DefiantLabs/lens/client"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
)

func GetBlockResultWithRetry(cl *lensClient.ChainClient, height int64, retryMaxAttempts int64, retryMaxWaitSeconds uint64) (*ctypes.ResultBlockResults, error) {
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
