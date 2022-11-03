package core

import (
	"fmt"

	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
	"go.uber.org/zap"
)

type BlockProcessingFailure int

const (
	NodeMissingBlockTxs BlockProcessingFailure = iota
	BlockQueryError
	UnprocessableTxError
	OsmosisNodeRewardLookupError
	OsmosisNodeRewardIndexError
)

type FailedBlockHandler func(height int64, code BlockProcessingFailure, err error)

// FIXME: this should be renamed, it isn't really handling failed blocks, that is happening elsewhere, this is a failure within a block.

// Log error to stdout. Not much else we can do to handle right now.
func HandleFailedBlock(height int64, code BlockProcessingFailure, err error) {
	reason := "{unknown error}"
	if code == NodeMissingBlockTxs {
		reason = "node has no TX history for block"
	} else if code == BlockQueryError {
		reason = "failed to query block result for block"
	} else if code == OsmosisNodeRewardLookupError {
		reason = "Failed Osmosis rewards lookup for block"
	} else if code == OsmosisNodeRewardIndexError {
		reason = "Failed Osmosis rewards indexing for block"
	}

	config.Log.Error(fmt.Sprintf("Block %v failed. Reason: %v", height, reason), zap.Error(err))
}
