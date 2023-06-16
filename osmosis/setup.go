package osmosis

import (
	"github.com/DefiantLabs/cosmos-indexer/config"
	"github.com/DefiantLabs/cosmos-indexer/osmosis/epochs/protorev"
	"github.com/DefiantLabs/cosmos-indexer/rpc"
	"github.com/DefiantLabs/probe/client"
)

// SetupOsmosisEpochIndexer sets up the indexer for the osmosis epoch indexing process
func SetupOsmosisEpochIndexer(cl *client.ChainClient) error {
	config.Log.Info("Setting up Osmosis Epoch Indexer")
	config.Log.Info("Gathering Protorev Developer Account Address")

	resp, err := rpc.GetProtorevDeveloperAccount(cl)
	if err != nil {
		config.Log.Error("Error getting Protorev Developer Account Address", err)
		return err
	}

	protorev.SetDeveloperAddress(resp.DeveloperAccount)
	config.Log.Debugf("Protorev Developer Account Address: %s", resp.DeveloperAccount)

	return nil
}
