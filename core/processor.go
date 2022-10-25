package core

import "fmt"

type BlockProcessingFailure int

const (
	NodeMissingBlockTxs BlockProcessingFailure = iota
	BlockQueryError
	UnprocessableTxError
	OsmosisNodeRewardLookupError
	OsmosisNodeRewardIndexError
)

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

	fmt.Printf("%s %d, details: %s", reason, height, err.Error())
}
