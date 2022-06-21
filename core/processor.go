package core

import "fmt"

type BlockProcessingFailure int

const (
	NodeMissingBlockTxs BlockProcessingFailure = iota
	BlockQueryError
)

//Log error to stdout. Not much else we can do to handle right now.
func HandleFailedBlock(height int64, code BlockProcessingFailure) {
	reason := "{unknown error}"
	if code == NodeMissingBlockTxs {
		reason = "node has no TX history for block"
	} else if code == BlockQueryError {
		reason = "failed to query block result for block"
	}

	fmt.Printf("%s %d", reason, height)
}
