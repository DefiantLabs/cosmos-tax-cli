package db

import (
	"fmt"

	"github.com/DefiantLabs/cosmos-exporter/osmosis"
	"gorm.io/gorm"
)

func GetHighestTaxableEventBlock(db *gorm.DB, chainID string) (Block, error) {
	var block Block

	result := db.Joins("JOIN block ON block.id = taxable_event.block").
		Where("block.chain.chainid = ?", chainID).Order("height desc").First(&block)

	return block, result.Error
}

func createTaxableEvents(db *gorm.DB, events []TaxableEvent) error {
	//Ordering matters due to foreign key constraints. Call Create() first to get right foreign key ID
	return db.Transaction(func(dbTransaction *gorm.DB) error {
		for _, event := range events {
			//TODO: this should be the same Chain for every TaxableEvent so maybe it could be optimized
			if err := dbTransaction.Where(&event.Block.Chain).FirstOrCreate(&event.Block.Chain).Error; err != nil {
				fmt.Printf("Error %s creating chain for TaxableEvent.\n", err)
				return err
			}

			//TODO: this should be the same Block for every TaxableEvent so maybe it could be optimized
			if err := dbTransaction.Where(&event.Block).FirstOrCreate(&event.Block).Error; err != nil {
				fmt.Printf("Error %s creating block for TaxableEvent.\n", err)
				return err
			}

			if event.EventAddress.Address != "" {
				//viewing gorm logs shows this gets translated into a single ON CONFLICT DO NOTHING RETURNING "id"
				if err := dbTransaction.Where(&event.EventAddress).FirstOrCreate(&event.EventAddress).Error; err != nil {
					fmt.Printf("Error %s creating address for TaxableEvent.\n", err)
					return err
				}
			}

			if event.Denomination.Denom != "" {
				//viewing gorm logs shows this gets translated into a single ON CONFLICT DO NOTHING RETURNING "id"
				if err := dbTransaction.Where(&event.Denomination).FirstOrCreate(&event.Denomination).Error; err != nil {
					fmt.Printf("Error %s creating SimpleDenom for TaxableEvent.\n", err)
					return err
				}
			}

			if err := dbTransaction.Create(&event).Error; err != nil {
				fmt.Printf("Error %s creating tx.\n", err)
				return err
			}

		}

		return nil
	})
}

func IndexOsmoRewards(db *gorm.DB, chainID string, rewards []*osmosis.OsmosisRewards) error {

	dbEvents := []TaxableEvent{}

	for _, curr := range rewards {
		for _, coin := range curr.Coins {
			evt := TaxableEvent{
				Source:       OsmosisRewardDistribution,
				Amount:       coin.Amount.ToDec().MustFloat64(),
				Denomination: SimpleDenom{Denom: coin.Denom},
				Block:        Block{Height: curr.EpochBlockHeight, Chain: Chain{ChainID: chainID, Name: chainID}},
				EventAddress: Address{Address: curr.Address},
			}
			dbEvents = append(dbEvents, evt)
		}
	}

	return createTaxableEvents(db, dbEvents)
}
