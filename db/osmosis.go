package db

import (
	"fmt"

	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
	"github.com/DefiantLabs/cosmos-tax-cli-private/osmosis"
	"github.com/DefiantLabs/cosmos-tax-cli-private/util"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func GetHighestTaxableEventBlock(db *gorm.DB, chainID string) (Block, error) {
	var block Block

	result := db.Joins("JOIN taxable_event ON blocks.id = taxable_event.block_id").
		Joins("JOIN chains ON blocks.blockchain_id = chains.id AND chains.chain_id = ?", chainID).Order("height desc").First(&block)

	return block, result.Error
}

func createTaxableEvents(db *gorm.DB, events []TaxableEvent) error {
	//Ordering matters due to foreign key constraints. Call Create() first to get right foreign key ID
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
				//viewing gorm logs shows this gets translated into a single ON CONFLICT DO NOTHING RETURNING "id"
				if err := dbTransaction.Where(&event.EventAddress).FirstOrCreate(&event.EventAddress).Error; err != nil {
					fmt.Printf("Error %s creating address for TaxableEvent.\n", err)
					return err
				}
			}

			if event.Denomination.Base == "" || event.Denomination.Symbol == "" {
				return fmt.Errorf("denom not cached for base %s and symbol %s", event.Denomination.Base, event.Denomination.Symbol)
			}

			if err := dbTransaction.Create(&event).Error; err != nil {
				fmt.Printf("Error %s creating tx.\n", err)
				return err
			}
		}

		return nil
	})
}

func IndexOsmoRewards(db *gorm.DB, chainID string, chainName string, rewards []*osmosis.Rewards) error {
	dbEvents := []TaxableEvent{}

	for _, curr := range rewards {
		for _, coin := range curr.Coins {
			denom, err := GetDenomForBase(coin.Denom)
			if err != nil {
				//attempt to add missing denoms to the database
				config.Log.Error("Denom lookup", zap.Error(err), zap.String("denom received", coin.Denom))
				denom, err = AddUnknownDenom(db, coin.Denom)
				if err != nil {
					config.Log.Error("There was an error adding a missing denom", zap.Error(err), zap.String("denom received", coin.Denom))
					return err
				}
			}

			evt := TaxableEvent{
				Source:       OsmosisRewardDistribution,
				Amount:       util.ToNumeric(coin.Amount.BigInt()),
				Denomination: denom,
				// FIXME: will this block have the correct time if it hasn't been indexed yet?
				Block:        Block{Height: curr.EpochBlockHeight, Chain: Chain{ChainID: chainID, Name: chainName}},
				EventAddress: Address{Address: curr.Address},
			}
			dbEvents = append(dbEvents, evt)
		}
	}
	config.Log.Debug("Rewards ready to insert in DB")

	return createTaxableEvents(db, dbEvents)
}
