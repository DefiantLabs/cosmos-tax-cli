package db

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
	"github.com/DefiantLabs/cosmos-tax-cli-private/osmosis"
	"github.com/DefiantLabs/cosmos-tax-cli-private/util"
	"gorm.io/gorm"
)

func GetHighestTaxableEventBlock(db *gorm.DB, chainID string) (Block, error) {
	var block Block

	result := db.Joins("JOIN taxable_event ON blocks.id = taxable_event.block_id").
		Joins("JOIN chains ON blocks.blockchain_id = chains.id AND chains.chain_id = ?", chainID).Order("height desc").First(&block)

	return block, result.Error
}

func eventExists(db *gorm.DB, event TaxableEvent) bool {
	var count int64
	db.Model(&TaxableEvent{}).Where("event_hash = ?", event.EventHash).Count(&count)
	return count > 0
}

func createTaxableEvents(db *gorm.DB, events []TaxableEvent) error {
	// Ordering matters due to foreign key constraints. Call Create() first to get right foreign key ID
	return db.Transaction(func(dbTransaction *gorm.DB) error {
		for _, event := range events {
			if chainErr := dbTransaction.Where(&event.Block.Chain).FirstOrCreate(&event.Block.Chain).Error; chainErr != nil {
				fmt.Printf("Error %s creating chain DB object.\n", chainErr)
				return chainErr
			}

			if blockErr := dbTransaction.Where(&event.Block).FirstOrCreate(&event.Block).Error; blockErr != nil {
				fmt.Printf("Error %s creating block DB object.\n", blockErr)
				return blockErr
			}

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
			if err := dbTransaction.Create(&thisEvent).Error; err != nil {
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
				if strings.Contains(coin.Denom, "gamm") {
					config.Log.Infof("Denom lookup failed for gamm token. Will be inserted as UNKNOWN. Denom Received: %v. Err: %v", coin.Denom, err)
				} else {
					config.Log.Warnf("Denom lookup failed. Will be inserted as UNKNOWN. Denom Received: %v. Err: %v", coin.Denom, err)
				}
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
	batchSize := 500
	config.Log.Debug(fmt.Sprintf("Rewards ready to insert in DB. Will insert in batches of %v", batchSize))
	for i := 0; i < len(dbEvents); i += batchSize {
		batchEnd := i + batchSize
		if batchEnd > len(dbEvents) {
			batchEnd = len(dbEvents) - 1
		}
		// if this batch has already been inserted, we can skip it
		if eventExists(db, dbEvents[i]) {
			continue
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
