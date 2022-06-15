package test

import (
	"testing"

	dbUtils "github.com/DefiantLabs/cosmos-exporter/db"
	"github.com/stretchr/testify/assert"
)

func TestGetOsmosisRewardIndex(t *testing.T) {
	addressRegex := "osmo(valoper)?1[a-z0-9]{38}"
	addressPrefix := "osmo"
	gorm, err := db_setup(addressRegex, addressPrefix)
	if err != nil {
		t.Fail()
	}

	setupOsmosisTestModels(gorm)
	createOsmosisTaxableEvent(gorm, 100)

	block, err := dbUtils.GetHighestTaxableEventBlock(gorm, "osmosis-1")
	if err != nil {
		t.Fail()
	}

	assert.Equal(t, block.Height, int64(100))
}

func TestInsertOsmosisRewards(t *testing.T) {
	addressRegex := "osmo(valoper)?1[a-z0-9]{38}"
	addressPrefix := "osmo"
	gorm, err := db_setup(addressRegex, addressPrefix)
	if err != nil {
		t.Fail()
	}

	setupOsmosisTestModels(gorm)
	createOsmosisTaxableEvent(gorm, 1111111111)

	block, err := dbUtils.GetHighestTaxableEventBlock(gorm, "osmosis-1")
	if err != nil {
		t.Fail()
	}

	assert.Equal(t, block.Height, int64(100))
}
