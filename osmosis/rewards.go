package osmosis

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

//IndexEpochsBetween figures out which blocks (in the given range, start height to end height)
//contain rewards distribution info, and indexes those blocks. Other blocks will be skipped.
//If an error is encountered while processing, the indexer will retry the block. The indexer
//will return an error if it cannot get a list of epochs to process or fails too many retries.
//See Osmosis repo x/incentives/keeper/distribute.go, doDistributionSends for more info.
func (client *URIClient) IndexEpochsBetween(startHeight int64, endHeight int64) error {
	rewardEpochs, epochLookupErr := client.getRewardEpochs(startHeight, endHeight)
	if epochLookupErr != nil {
		return epochLookupErr
	}

	retryList := []int64{}
	for _, epoch := range rewardEpochs {
		indexErr := client.IndexEpoch(epoch)
		if indexErr != nil {
			retryList = append(retryList, epoch)
		}
	}

	//Now try again but return an error if there's a second failure
	for _, epoch := range retryList {
		indexErr := client.IndexEpoch(epoch)
		if indexErr != nil {
			return indexErr
		}
	}

	return nil
}

//IndexEpoch indexes any reward distribution at the given block height.
//If a block does not contain a reward distribution, it gets skipped.
//Skipped blocks do not cause an error. Therefore, an error indicates
//a problem with the RPC search or with the DB indexer.
func (client *URIClient) IndexEpoch(height int64) error {
	rewards, epochErr := client.getRewards(height)
	if epochErr != nil {
		fmt.Printf("Error %s processing epoch %d\n", epochErr.Error(), height)
		return epochErr
	} else {
		dbIndexEpoch(rewards)
	}

	return nil
}

func dbIndexEpoch(rewards []*OsmosisRewards) {

}

//GetRewardEpochs (RPC) Get a list of the block heights where Osmosis distributed rewards.
//Rewards are distributed daily, the block height is time based and not known in advance.
//The Osmosis SDK emits ABCI events (to tendermint) when rewards are distributed. This function
//queries the node via RPC and figures out what blocks contain the reward distribution info.
//The events are emitted under the key "distribution.receiver" so that is what we search for.
func (client *URIClient) getRewardEpochs(startHeight int64, endHeight int64) ([]int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	osmosisRewardsQuery := "distribution.receiver EXISTS"
	rewardBlocks := []int64{}

	//We search for the EXACT block height because I could not make the BlockSearch
	//pagination work. This is a slow process, but for our indexer it doesn't matter.
	for i := startHeight; i <= endHeight; i++ {
		query := fmt.Sprintf("block.height = %d AND %s", i, osmosisRewardsQuery)
		page := 1
		per_page := 30
		blockSearch, blockSearchErr := client.DoBlockSearch(ctx, query, &page, &per_page, "desc")

		if blockSearchErr != nil {
			return nil, blockSearchErr
		}

		for _, block := range blockSearch.Blocks {
			if block != nil {
				rewardBlocks = append(rewardBlocks, block.Block.Header.Height)
			}
		}
	}

	return rewardBlocks, nil
}

//GetRewards Gets the total rewards distributed to each address
//during the given epoch (block height). If any errors are encountered
//during processing of this block height, an error will be returned
//and no reward information will be returned. This forces reprocessing
//of failed blocks.
func (client *URIClient) getRewards(height int64) ([]*OsmosisRewards, error) {
	rewards := []*OsmosisRewards{}

	//Nodes are very slow at responding to queries for reward distribution blocks.
	//I believe you must set the Node's timeout_broadcast_tx_commit higher than 10s
	//or these queries will time out
	brctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	bresults, err := client.DoBlockResults(brctx, &height)
	if err != nil {
		return nil, err
	}

	//Osmosis emits reward distribution events during the BeginBlocker,
	//which means they show up in the BeginBlockEvents section
	for _, event := range bresults.BeginBlockEvents {
		if event.Type == "distribution" {
			receiver_addr := ""
			receiver_amount := ""

			for _, attr := range event.Attributes {
				if string(attr.Key) == "receiver" {
					receiver_addr = string(attr.Value)
				}
				if string(attr.Key) == "amount" {
					receiver_amount = string(attr.Value)
				}
			}

			if receiver_addr != "" && receiver_amount != "" {
				coins, parseErr := sdk.ParseCoinsNormalized(receiver_amount)
				if parseErr == nil {
					rewards = append(rewards, &OsmosisRewards{Address: receiver_addr, Coins: coins})
				} else {
					return nil, parseErr
				}
			}
		}
	}

	return rewards, nil
}
