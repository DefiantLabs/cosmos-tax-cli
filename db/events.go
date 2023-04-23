package db

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/events"
	"github.com/DefiantLabs/cosmos-tax-cli/util"
	"gorm.io/gorm"
)

func IndexBlockEvents(db *gorm.DB, dryRun bool, blockHeight int64, blockTime time.Time, blockEvents []events.EventRelevantInformation, dbChainID string, dbChainName string) error {
	dbEvents := []TaxableEvent{}

	for _, blockEvent := range blockEvents {

		denom, err := GetDenomForBase(blockEvent.Denomination)
		if err != nil {
			// attempt to add missing denoms to the database
			config.Log.Warnf("Denom lookup failed. Will be inserted as UNKNOWN. Denom Received: %v. Err: %v", blockEvent.Denomination, err)
			denom, err = AddUnknownDenom(db, blockEvent.Denomination)
			if err != nil {
				config.Log.Error(fmt.Sprintf("There was an error adding a missing denom. Denom Received: %v", blockEvent.Denomination), err)
				return err
			}
		}

		// Create unique hash for each event to ensure idempotency
		hash := sha256.New()
		hash.Write([]byte(fmt.Sprint(blockEvent.Address, blockHeight, fmt.Sprintf("%v%s", blockEvent.Amount, blockEvent.Denomination), blockEvent.EventSource)))

		evt := TaxableEvent{
			Source:       blockEvent.EventSource,
			Amount:       util.ToNumeric(blockEvent.Amount),
			EventHash:    fmt.Sprintf("%x", hash.Sum(nil)),
			Denomination: denom,
			Block:        Block{Height: blockHeight, TimeStamp: blockTime, Chain: Chain{ChainID: dbChainID, Name: dbChainName}},
			EventAddress: Address{Address: blockEvent.Address},
		}
		dbEvents = append(dbEvents, evt)

	}

	// sort by hash
	sort.SliceStable(dbEvents, func(i, j int) bool {
		return dbEvents[i].EventHash < dbEvents[j].EventHash
	})

	// insert events into DB in batches of batchSize
	batchSize := 1000
	for i := 0; i < len(dbEvents); i += batchSize {
		batchEnd := i + batchSize
		if batchEnd > len(dbEvents) {
			batchEnd = len(dbEvents)
		}

		awaitingInsert := dbEvents[i:batchEnd]

		//Only way this can happen is if i == batchEnd
		if len(awaitingInsert) == 0 {
			awaitingInsert = []TaxableEvent{dbEvents[i]}
		}

		if !dryRun {
			err := createTaxableEvents(db, awaitingInsert)
			if err != nil {
				config.Log.Error("Error storing DB events.", err)
				return err
			}
		}
	}

	return nil
}
