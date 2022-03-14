package rest

import (
	"fmt"
	"os"
	"strconv"

	configHelpers "cosmos-exporter/config"
	dbTypes "cosmos-exporter/db"

	"gorm.io/gorm"
)

func GetLatestBlockHeight(host string) uint64 {
	var latestBlock uint64 = 1
	resp, err := GetLatestBlock(host)

	if err != nil {
		fmt.Println("Error getting latest block", err)
		os.Exit(1)
	}

	latestBlock, err = strconv.ParseUint(resp.Block.BlockHeader.Height, 10, 64)

	if err != nil {
		fmt.Println("Error getting latest block", err)
		os.Exit(1)
	}

	return latestBlock
}

func GetBlockStartHeight(config *configHelpers.Config, db *gorm.DB) uint64 {

	//Block MUST be a valid height > 0
	if config.Base.StartBlock != -1 {
		return uint64(config.Base.StartBlock)
	}

	latestBlock := GetLatestBlockHeight(config.Api.Host)
	fmt.Println("Found latest block", latestBlock)
	highestIndexedBlock := dbTypes.GetHighestIndexedBlock(db)
	if highestIndexedBlock.Height < latestBlock {
		return highestIndexedBlock.Height + 1
	}

	return latestBlock
}
