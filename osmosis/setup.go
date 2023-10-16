package osmosis

import (
	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/epochs/protorev"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/epochs"
	"github.com/DefiantLabs/cosmos-tax-cli/rpc"
	"github.com/DefiantLabs/lens/client"
)

// SetupOsmosisEpochIndexer sets up the indexer for the osmosis epoch indexing process
func SetupOsmosisEpochIndexer(cl *client.ChainClient, epochIdentifier string) error {
	config.Log.Info("Setting up Osmosis Epoch Indexer")

	switch epochIdentifier {
	case epochs.WeekEpochIdentifier:
		config.Log.Info("Gathering Protorev Developer Account Address")

		resp, err := rpc.GetProtorevDeveloperAccount(cl)
		if err != nil {
			config.Log.Error("Error getting Protorev Developer Account Address", err)
			return err
		}

		protorev.SetDeveloperAddress(resp.DeveloperAccount)
		config.Log.Debugf("Protorev Developer Account Address: %s", resp.DeveloperAccount)
	default:
		config.Log.Infof("Epoch Identifier %s requires no setup, skipping.", epochIdentifier)
	}

	return nil
}
