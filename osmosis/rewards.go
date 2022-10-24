package osmosis

import (
	"context"
	"fmt"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetRewardsBetween figures out which blocks (in the given range, start height to end height)
// contain Osmosis rewards, and queries the reward info. Blocks without rewards are skipped.
// An error is returned if we cannot query a list of reward epochs. Otherwise []*OsmosisRewards
// is returned, which contains all of the rewards for a given block height and address.
// See Osmosis repo x/incentives/keeper/distribute.go, doDistributionSends for more info.
func (client *URIClient) GetRewardsBetween(startHeight int64, endHeight int64) ([]*OsmosisRewards, error) {
	// rewardEpochs, epochLookupErr := client.getRewardEpochs(startHeight, endHeight)
	// if epochLookupErr != nil {
	// 	return nil, epochLookupErr
	// }

	epochList := []*OsmosisRewards{}
	// for _, epoch := range rewardEpochs {
	for epoch := startHeight; epoch <= endHeight; epoch++ {
		rewards, indexErr := client.GetEpochRewards(epoch)
		if indexErr != nil {
			return nil, indexErr
		}
		epochList = append(epochList, rewards...)
	}

	return epochList, nil
}

// IndexEpoch indexes any reward distribution at the given block height.
// If a block does not contain a reward distribution, it gets skipped.
// An error indicates a problem with the RPC search or the DB indexer.
func (client *URIClient) GetEpochRewards(height int64) ([]*OsmosisRewards, error) {
	rewards, epochErr := client.getRewards(height)
	if epochErr != nil {
		fmt.Printf("Error %s processing epoch %d\n", epochErr.Error(), height)
		return nil, epochErr
	}

	return rewards, nil
}

// GetRewardEpochs (RPC) Get a list of the block heights where Osmosis distributed rewards.
// Rewards are distributed daily, the block height is time based and not known in advance.
// The Osmosis SDK emits ABCI events (to tendermint) when rewards are distributed. This function
// queries the node via RPC and figures out what blocks contain the reward distribution info.
// The events are emitted under the key "distribution.receiver" so that is what we search for.
//
//nolint:unused
func (client *URIClient) getRewardEpochs(startHeight int64, endHeight int64) ([]int64, error) {
	osmosisRewardsQuery := "distribution.receiver EXISTS"
	rewardBlocks := []int64{}

	//We search for the EXACT block height because I could not make the BlockSearch
	//pagination work. This is a slow process, but for our indexer it doesn't matter.
	for i := startHeight; i <= endHeight; i++ {
		query := fmt.Sprintf("block.height = %d AND %s", i, osmosisRewardsQuery)
		page := 1
		per_page := 30
		blockSearch, blockSearchErr := client.DoBlockSearch(context.Background(), query, &page, &per_page, "desc")

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

// GetRewards Gets the total rewards distributed to each address
// during the given epoch (block height). If any errors are encountered
// during processing of this block height, an error will be returned
// and no reward information will be returned. This forces reprocessing
// of failed blocks.
func (client *URIClient) getRewards(height int64) ([]*OsmosisRewards, error) {
	rewards := map[string]*OsmosisRewards{}

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

				if strings.Contains(receiver_amount, ",") {
					fmt.Printf("Fuck VSCode")
				}
			}

			if receiver_addr != "" && receiver_amount != "" {
				coins, parseErr := sdk.ParseCoinsNormalized(receiver_amount)
				if parseErr == nil {
					if prevRewards, ok := rewards[receiver_addr]; ok {
						fmt.Printf("Receiver has more than one entry for Osmosis rewards at block height %d\n", prevRewards.EpochBlockHeight)
						coinsCombined := addCoins(prevRewards.Coins, coins)
						rewards[receiver_addr].Coins = coinsCombined
					} else {
						rewards[receiver_addr] = &OsmosisRewards{Address: receiver_addr, Coins: coins, EpochBlockHeight: height}
					}
				} else {
					return nil, parseErr
				}
			}
		}
	}

	allRewards := []*OsmosisRewards{}
	for _, reward := range rewards {
		allRewards = append(allRewards, reward)
	}

	return allRewards, nil
}

func addCoins(coinList1 []sdk.Coin, coinList2 []sdk.Coin) []sdk.Coin {
	fullList := append(coinList1, coinList2...)
	denomAmountMap := map[string]sdk.Int{} //key = coin denom, value = coin amount

	for _, coin := range fullList {
		if prevAmount, ok := denomAmountMap[coin.Denom]; ok {
			denomAmountMap[coin.Denom] = prevAmount.Add(coin.Amount)
		} else {
			denomAmountMap[coin.Denom] = coin.Amount
		}
	}

	combinedList := []sdk.Coin{}
	for denom, amount := range denomAmountMap {
		combinedList = append(combinedList, sdk.NewCoin(denom, amount))
	}

	return combinedList
}
