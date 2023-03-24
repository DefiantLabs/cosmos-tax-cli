package db

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis"
	"github.com/DefiantLabs/cosmos-tax-cli/util"

	"gorm.io/gorm"
)

func GetHighestTaxableEventBlock(db *gorm.DB, chainID string) (Block, error) {
	var block Block

	result := db.Joins("JOIN taxable_event ON blocks.id = taxable_event.block_id").
		Joins("JOIN chains ON blocks.blockchain_id = chains.id AND chains.chain_id = ?", chainID).Order("height desc").First(&block)

	return block, result.Error
}

func createTaxableEvents(db *gorm.DB, events []TaxableEvent) error {
	// Ordering matters due to foreign key constraints. Call Create() first to get right foreign key ID
	return db.Transaction(func(dbTransaction *gorm.DB) error {
		if len(events) == 0 {
			return errors.New("no events to insert")
		}

		var chainPrev Chain
		var blockPrev Block

		for _, event := range events {
			//whereCond := Chain{ChainID: event.Block.Chain.ChainID, Name: event.Block.Chain.Name}
			if chainPrev.ChainID != event.Block.Chain.ChainID || event.Block.Chain.Name != chainPrev.Name {
				if chainErr := dbTransaction.Where(&event.Block.Chain).FirstOrCreate(&event.Block.Chain).Error; chainErr != nil {
					fmt.Printf("Error %s creating chain DB object.\n", chainErr)
					return chainErr
				}

				chainPrev = event.Block.Chain
			}

			event.Block.Chain = chainPrev

			if blockPrev.Height != event.Block.Height {
				whereCond := Block{Chain: event.Block.Chain, Height: event.Block.Height}
				if blockErr := dbTransaction.Where(whereCond).FirstOrCreate(&event.Block).Error; blockErr != nil {
					fmt.Printf("Error %s creating block DB object.\n", blockErr)
					return blockErr
				}

				blockPrev = event.Block
			}

			event.Block = blockPrev

			if event.EventAddress.Address != "" {
				// viewing gorm logs shows this gets translated into a single ON CONFLICT DO NOTHING RETURNING "id"
				if err := dbTransaction.Where(&event.EventAddress).FirstOrCreate(&event.EventAddress).Error; err != nil {
					fmt.Printf("Error %s creating address for TaxableEvent.\n", err)
					return err
				}
			}

			if event.Denomination.Base == "" || event.Denomination.Symbol == "" {
				return fmt.Errorf("denom not cached for base %s and symbol %s", event.Denomination.Base, event.Denomination.Symbol)
			}

			thisEvent := event // This is redundant but required for the picky gosec linter
			if err := dbTransaction.Where(TaxableEvent{EventHash: event.EventHash}).FirstOrCreate(&thisEvent).Error; err != nil {
				fmt.Printf("Error %s creating tx.\n", err)
				return err
			}
		}

		return nil
	})
}

func IndexOsmoRewards(db *gorm.DB, dryRun bool, chainID string, chainName string, rewards *osmosis.RewardsInfo) error {
	dbEvents := []TaxableEvent{}

	for _, curr := range rewards.Rewards {
		for _, coin := range curr.Coins {
			denom, err := GetDenomForBase(coin.Denom)
			if err != nil {
				// attempt to add missing denoms to the database
				config.Log.Warnf("Denom lookup failed. Will be inserted as UNKNOWN. Denom Received: %v. Err: %v", coin.Denom, err)
				denom, err = AddUnknownDenom(db, coin.Denom)
				if err != nil {
					config.Log.Error(fmt.Sprintf("There was an error adding a missing denom. Denom Received: %v", coin.Denom), err)
					return err
				}
			}

			hash := sha256.New()
			hash.Write([]byte(fmt.Sprint(curr.Address, rewards.EpochBlockHeight, coin)))

			evt := TaxableEvent{
				Source:       OsmosisRewardDistribution,
				Amount:       util.ToNumeric(coin.Amount.BigInt()),
				EventHash:    fmt.Sprintf("%x", hash.Sum(nil)),
				Denomination: denom,
				Block:        Block{Height: rewards.EpochBlockHeight, TimeStamp: rewards.EpochBlockTime, Chain: Chain{ChainID: chainID, Name: chainName}},
				EventAddress: Address{Address: curr.Address},
			}
			dbEvents = append(dbEvents, evt)
		}
	}

	// sort by hash
	sort.SliceStable(dbEvents, func(i, j int) bool {
		return dbEvents[i].EventHash < dbEvents[j].EventHash
	})

	// insert rewards into DB in batches of batchSize
	batchSize := 10000
	for i := 0; i < len(dbEvents); i += batchSize {
		batchEnd := i + batchSize
		if batchEnd > len(dbEvents) {
			batchEnd = len(dbEvents) - 1
		}

		if !dryRun {
			err := createTaxableEvents(db, dbEvents[i:batchEnd])
			if err != nil {
				config.Log.Error("Error storing DB events.", err)
				return err
			}
		}
	}

	return nil
}
