package core

import (
	"fmt"

	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
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

// Log error to stdout. Not much else we can do to handle right now.
func HandleFailedBlock(height int64, code BlockProcessingFailure, err error) {
	reason := "{unknown error}"
	switch code {
	case NodeMissingBlockTxs:
		reason = "node has no TX history for block"
	case BlockQueryError:
		reason = "failed to query block result for block"
	case OsmosisNodeRewardLookupError:
		reason = "Failed Osmosis rewards lookup for block"
	case OsmosisNodeRewardIndexError:
		reason = "Failed Osmosis rewards indexing for block"
	}

	config.Log.Error(fmt.Sprintf("Block %v failed. Reason: %v", height, reason), err)
}
