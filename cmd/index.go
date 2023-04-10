package cmd

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/DefiantLabs/lens/client"
	"github.com/go-co-op/gocron"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/core"
	dbTypes "github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis"
	"github.com/DefiantLabs/cosmos-tax-cli/rpc"
	"github.com/DefiantLabs/cosmos-tax-cli/tasks"
	"github.com/spf13/cobra"

	"gorm.io/gorm"
)

var reindexMsgType string

func init() {
	indexCmd.Flags().StringVar(&reindexMsgType, "re-index-message-type", "", "If specified, the indexer will reindex only the blocks containing the message type provided.")

	rootCmd.AddCommand(indexCmd)
}

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Indexes the blockchain according to the configuration defined.",
	Long: `Indexes the Cosmos-based blockchain according to the configurations found on the command line
	or in the specified config file. Indexes taxable events into a database for easy querying. It is
	highly recommended to keep this command running as a background service to keep your index up to date.`,
	Run: index,
}

// The Indexer struct is used to perform index operations
type Indexer struct {
	cfg       *config.Config
	dryRun    bool
	db        *gorm.DB
	cl        *client.ChainClient
	scheduler *gocron.Scheduler
}

func setupIndexer() *Indexer {
	var idxr Indexer
	var err error
	idxr.cfg, idxr.dryRun, idxr.db, idxr.scheduler, err = setup(conf)
	if err != nil {
		log.Fatalf("Error during application setup. Err: %v", err)
	}

	// Setup chain specific stuff
	core.SetupAddressRegex(idxr.cfg.Lens.AccountPrefix + "(valoper)?1[a-z0-9]{38}")
	core.SetupAddressPrefix(idxr.cfg.Lens.AccountPrefix)
	core.ChainSpecificMessageTypeHandlerBootstrap(idxr.cfg.Lens.ChainID)
	config.SetChainConfig(idxr.cfg.Lens.AccountPrefix)

	// Setup scheduler to periodically update denoms
	if idxr.cfg.Base.API != "" {
		_, err = idxr.scheduler.Every(6).Hours().Do(tasks.DenomUpsertTask, idxr.cfg.Base.API, idxr.db)
		if err != nil {
			config.Log.Error("Error scheduling denom upsert task. Err: ", err)
		}

		_, err = idxr.scheduler.Every(6).Hours().Do(tasks.IBCDenomUpsertTask, idxr.cfg.Base.API, idxr.db)
		if err != nil {
			config.Log.Error("Error scheduling ibc denom upsert task. Err: ", err)
		}

		idxr.scheduler.StartAsync()
	}

	// Some chains do not have the denom metadata URL available on chain, so we do chain specific downloads instead.
	tasks.DoChainSpecificUpsertDenoms(idxr.db, idxr.cfg.Lens.ChainID)
	idxr.cl = config.GetLensClient(idxr.cfg.Lens)

	// Depending on the app configuration, wait for the chain to catch up
	chainCatchingUp, err := rpc.IsCatchingUp(idxr.cl)
	for idxr.cfg.Base.WaitForChain && chainCatchingUp && err == nil {
		// Wait between status checks, don't spam the node with requests
		config.Log.Debug("Chain is still catching up, please wait or disable check in config.")
		time.Sleep(time.Second * time.Duration(idxr.cfg.Base.WaitForChainDelay))
		chainCatchingUp, err = rpc.IsCatchingUp(idxr.cl)

		// This EOF error pops up from time to time and is unpredictable
		// It is most likely an error on the node, we would need to see any error logs on the node side
		// Try one more time
		if err != nil && strings.HasSuffix(err.Error(), "EOF") {
			time.Sleep(time.Second * time.Duration(idxr.cfg.Base.WaitForChainDelay))
			chainCatchingUp, err = rpc.IsCatchingUp(idxr.cl)
		}
	}
	if err != nil {
		config.Log.Fatal("Error querying chain status.", err)
	}

	return &idxr
}

func index(cmd *cobra.Command, args []string) {
	// Setup the indexer with config, db, and cl
	idxr := setupIndexer()
	dbConn, err := idxr.db.DB()
	if err != nil {
		config.Log.Fatal("Failed to connect to DB", err)
	}
	defer dbConn.Close()

	// blockChan are just the block heights; limit max jobs in the queue, otherwise this queue would contain one
	// item (block height) for every block on the entire blockchain we're indexing. Furthermore, once the queue
	// is close to empty, we will spin up a new thread to fill it up with new jobs.
	blockChan := make(chan int64, 10000)

	// This channel represents query job results for the RPC queries to Cosmos Nodes. Every time an RPC query
	// completes, the query result will be sent to this channel (for later processing by a different thread).
	// Realistically, I expect that RPC queries will be slower than our relational DB on the local network.
	// If RPC queries are faster than DB inserts this buffer will fill up.
	// We will periodically check the buffer size to monitor performance so we can optimize later.
	rpcQueryThreads := int(idxr.cfg.Base.RPCWorkers)
	if rpcQueryThreads == 0 {
		rpcQueryThreads = 4
	} else if rpcQueryThreads > 64 {
		rpcQueryThreads = 64
	}
	txDataChan := make(chan *dbData, 4*rpcQueryThreads)
	var txChanWaitGroup sync.WaitGroup // This group is to ensure we are done getting transactions before we close the TX channel
	// Spin up a (configurable) number of threads to query RPC endpoints for Transactions.
	// this is assumed to be the slowest process that allows concurrency and thus has the most dedicated go routines.
	if idxr.cfg.Base.ChainIndexingEnabled {
		for i := 0; i < rpcQueryThreads; i++ {
			txChanWaitGroup.Add(1)
			go func() {
				idxr.queryRPC(blockChan, txDataChan, core.HandleFailedBlock)
				txChanWaitGroup.Done()
			}()
		}
	}

	// close the transaction chan once all transactions have been written to it
	go func() {
		txChanWaitGroup.Wait()
		close(txDataChan)
	}()

	var wg sync.WaitGroup // This group is to ensure we are done processing transactions (as well as osmo rewards) before returning

	// Osmosis specific indexing requirements. Osmosis distributes rewards to LP holders on a daily basis.
	rewardsDataChan := make(chan *osmosis.RewardsInfo, 4*rpcQueryThreads)
	if config.IsOsmosis(idxr.cfg) && idxr.cfg.Base.RewardIndexingEnabled {
		wg.Add(1)
		go idxr.indexOsmosisRewards(&wg, core.HandleFailedBlock, rewardsDataChan)
	} else {
		close(rewardsDataChan)
	}

	chain := dbTypes.Chain{
		ChainID: idxr.cfg.Lens.ChainID,
		Name:    idxr.cfg.Lens.ChainName,
	}
	dbChainID, err := dbTypes.GetDBChainID(idxr.db, chain)
	if err != nil {
		config.Log.Fatal("Failed to add/create chain in DB", err)
	}

	// Start a thread to index the data queried from the chain.
	if idxr.cfg.Base.ChainIndexingEnabled || idxr.cfg.Base.RewardIndexingEnabled {
		wg.Add(1)
		go idxr.doDBUpdates(&wg, txDataChan, rewardsDataChan, dbChainID)
	}

	// Add jobs to the queue to be processed
	if idxr.cfg.Base.ChainIndexingEnabled {
		if reindexMsgType != "" {
			idxr.enqueueBlocksToProcessByMsgType(blockChan, dbChainID, reindexMsgType)
		} else {
			idxr.enqueueBlocksToProcess(blockChan, dbChainID)
		}
		// close the block chan once all blocks have been written to it
		close(blockChan)
	}

	// If we error out in the main loop, this will block. Meaning we may not know of an error for 6 hours until last scheduled task stops
	idxr.scheduler.Stop()
	wg.Wait()
}

// enqueueBlocksToProcessByMsgType will pass the blocks containing the specified msg type to the indexer
func (idxr *Indexer) enqueueBlocksToProcessByMsgType(blockChan chan int64, chainID uint, msgType string) {
	// get the block range
	startBlock := idxr.cfg.Base.StartBlock
	endBlock := idxr.cfg.Base.EndBlock
	if endBlock == -1 {
		heighestBlock := dbTypes.GetHighestIndexedBlock(idxr.db, chainID)
		endBlock = heighestBlock.Height
	}

	rows, err := idxr.db.Raw(`SELECT height FROM blocks
							JOIN txes ON txes.block_id = blocks.id
							JOIN messages ON messages.tx_id = txes.id
							JOIN message_types ON message_types.id = messages.message_type_id
							AND message_types.message_type = ?
							WHERE height > ? AND height < ? AND blockchain_id = ?::int;
							`, msgType, startBlock, endBlock, chainID).Rows()
	if err != nil {
		config.Log.Fatalf("Error checking DB for blocks to reindex. Err: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var block int64
		err = idxr.db.ScanRows(rows, &block)
		if err != nil {
			config.Log.Fatal("Error getting block height. Err: %v", err)
		}
		config.Log.Debugf("Sending block %v to be re-indexed.", block)

		if idxr.cfg.Base.Throttling != 0 {
			time.Sleep(time.Second * time.Duration(idxr.cfg.Base.Throttling))
		}

		// Add the new block to the queue
		blockChan <- block
	}
}

func (idxr *Indexer) enqueueFailedBlocks(blockChan chan int64, chainID uint) {
	// Get all failed blocks
	failedBlocks := dbTypes.GetFailedBlocks(idxr.db, chainID)
	if len(failedBlocks) == 0 {
		return
	}
	for _, block := range failedBlocks {
		if idxr.cfg.Base.Throttling != 0 {
			time.Sleep(time.Second * time.Duration(idxr.cfg.Base.Throttling))
		}
		config.Log.Infof("Will re-attempt failed block: %v", block.Height)
		blockChan <- block.Height
	}
	config.Log.Info("All failed blocks have been re-enqueued for processing")
}

// enqueueBlocksToProcess will pass the blocks that need to be processed to the blockchannel
func (idxr *Indexer) enqueueBlocksToProcess(blockChan chan int64, chainID uint) {
	// Unless explicitly prevented, lets attempt to enqueue any failed blocks
	if !idxr.cfg.Base.PreventReattempts {
		idxr.enqueueFailedBlocks(blockChan, chainID)
	}

	// Start at the last indexed block height (or the block height in the config, if set)
	currBlock := idxr.GetIndexerStartingHeight(chainID)
	// Don't index past this block no matter what
	lastBlock := idxr.cfg.Base.EndBlock
	var latestBlock int64 = math.MaxInt64

	// Add jobs to the queue to be processed
	for {
		// The program is configured to stop running after a set block height.
		// Generally this will only be done while debugging or if a particular block was incorrectly processed.
		if lastBlock != -1 && currBlock > lastBlock {
			config.Log.Info("Hit the last block we're allowed to index, exiting enqueue func.")
			return
		} else if idxr.cfg.Base.ExitWhenCaughtUp && currBlock > latestBlock {
			config.Log.Info("Hit the last block we're allowed to index, exiting enqueue func.")
			return
		}

		// The job queue is running out of jobs to process, see if the blockchain has produced any new blocks we haven't indexed yet.
		if len(blockChan) <= cap(blockChan)/4 {
			// This is the latest block height available on the Node.
			var err error
			latestBlock, err = rpc.GetLatestBlockHeight(idxr.cl)
			if err != nil {
				config.Log.Fatal("Error getting blockchain latest height. Err: %v", err)
			}

			// Throttling in case of hitting public APIs
			if idxr.cfg.Base.Throttling != 0 {
				time.Sleep(time.Second * time.Duration(idxr.cfg.Base.Throttling))
			}

			// Already at the latest block, wait for the next block to be available.
			for currBlock <= latestBlock && (currBlock <= lastBlock || lastBlock == -1) && len(blockChan) != cap(blockChan) {
				// if we are not re-indexing, skip curr block if already indexed
				if !idxr.cfg.Base.ReIndex && blockAlreadyIndexed(currBlock, chainID, idxr.db) {
					currBlock++
					continue
				}

				if idxr.cfg.Base.Throttling != 0 {
					time.Sleep(time.Second * time.Duration(idxr.cfg.Base.Throttling))
				}

				// Add the new block to the queue
				blockChan <- currBlock
				currBlock++
			}
		}
	}
}

// If nothing has been indexed yet, the start height should be 0.
func OsmosisGetRewardsStartIndexHeight(db *gorm.DB, chainID string) int64 {
	block, err := dbTypes.GetHighestTaxableEventBlock(db, chainID)
	if err != nil && err.Error() != "record not found" {
		log.Fatalf("Cannot retrieve highest indexed Osmosis rewards block. Err: %v", err)
	}

	return block.Height
}

// blockAlreadyIndexed will return true if the block is already in the DB
func blockAlreadyIndexed(blockHeight int64, chainID uint, db *gorm.DB) bool {
	var exists bool
	err := db.Raw(`SELECT count(*) > 0 FROM blocks WHERE height = ?::int AND blockchain_id = ?::int AND indexed = true AND time_stamp != '0001-01-01T00:00:00.000Z';`, blockHeight, chainID).Row().Scan(&exists)
	if err != nil {
		config.Log.Fatalf("Error checking DB for block. Err: %v", err)
	}
	return exists
}

// GetIndexerStartingHeight will determine which block to start at
// if start block is set to -1, it will start at the highest block indexed
// otherwise, it will start at the first missing block between the start and end height
func (idxr *Indexer) GetIndexerStartingHeight(chainID uint) int64 {
	// If the start height is set to -1, resume from the highest block already indexed
	if idxr.cfg.Base.StartBlock == -1 {
		latestBlock, err := rpc.GetLatestBlockHeight(idxr.cl)
		if err != nil {
			log.Fatalf("Error getting blockchain latest height. Err: %v", err)
		}

		fmt.Println("Found latest block", latestBlock)
		highestIndexedBlock := dbTypes.GetHighestIndexedBlock(idxr.db, chainID)
		if highestIndexedBlock.Height < latestBlock {
			return highestIndexedBlock.Height + 1
		}
	}

	// if we are re-indexing, just start at the configured start block
	if idxr.cfg.Base.ReIndex {
		return idxr.cfg.Base.StartBlock
	}

	maxStart := idxr.cfg.Base.EndBlock
	if maxStart == -1 {
		heighestBlock := dbTypes.GetHighestIndexedBlock(idxr.db, chainID)
		maxStart = heighestBlock.Height
	}

	// Otherwise, start at the first block after the configured start block that we have not yet indexed.
	return dbTypes.GetFirstMissingBlockInRange(idxr.db, idxr.cfg.Base.StartBlock, maxStart, chainID)
}

func (idxr *Indexer) indexOsmosisRewards(wg *sync.WaitGroup, failedBlockHandler core.FailedBlockHandler, rewardsDataChan chan *osmosis.RewardsInfo) {
	defer wg.Done()
	defer close(rewardsDataChan)

	averageOsmosisBlocksPerDay := int64(13362)
	reindex := idxr.cfg.Base.ReIndex
	startHeight := idxr.cfg.Base.RewardStartBlock
	intervalWidth := int64(14000)
	lastKnownRewardsHeight := int64(-1)
	ignoreIntervalWidth := true

	if startHeight <= 0 || !reindex {
		dbLastIndexedReward := OsmosisGetRewardsStartIndexHeight(idxr.db, idxr.cfg.Lens.ChainID)
		if dbLastIndexedReward > 0 {
			//the next plausible block that might contain osmosis rewards is a day later
			startHeight = dbLastIndexedReward + averageOsmosisBlocksPerDay
			ignoreIntervalWidth = false
		}
	}

	// 0 isn't a valid starting block
	if startHeight <= 0 {
		startHeight = 1
	}

	lastKnownBlockHeight, errBh := rpc.GetLatestBlockHeight(idxr.cl)
	if errBh != nil {
		config.Log.Fatal("Error getting blockchain latest height.", errBh)
	}

	endHeight := idxr.cfg.Base.RewardEndBlock
	if endHeight == -1 {
		endHeight = lastKnownBlockHeight
	}

	config.Log.Infof("Indexing Rewards from block: %v to %v", idxr.cfg.Base.RewardStartBlock, endHeight)

	rpcClient := osmosis.URIClient{
		Address: idxr.cl.Config.RPCAddr,
		Client:  &http.Client{},
	}

	delta := int64(0)

	// If we've never found a rewards epoch before, we will search sequentially until one is found.
	// From that point on, we assume the next epoch is roughly the same number of blocks away as the previous one.
	// We will give up if we cannot find an epoch "close enough" to the estimated block height (intervalWidth).
	for (delta <= intervalWidth || ignoreIntervalWidth) &&
		(endHeight == -1 || startHeight+delta <= endHeight) {

		if math.Abs(float64((startHeight+delta)-lastKnownBlockHeight)) <= 100 {
			time.Sleep(1 * time.Hour)

			lastKnownBlockHeight, errBh = rpc.GetLatestBlockHeight(idxr.cl)
			if errBh != nil {
				config.Log.Fatal("Error getting blockchain latest height.", errBh)
			}
		} else if delta%1000 == 0 {
			lastKnownBlockHeight, errBh = rpc.GetLatestBlockHeight(idxr.cl)
			if errBh != nil {
				config.Log.Fatal("Error getting blockchain latest height.", errBh)
			}
		}

		if idxr.cfg.Base.Throttling != 0 {
			time.Sleep(time.Second * time.Duration(idxr.cfg.Base.Throttling))
		}

		// Search in the forwards direction
		hasRewards, err := idxr.processRewardEpoch(startHeight+delta, rewardsDataChan, rpcClient, failedBlockHandler)

		if err != nil {
			config.Log.Fatalf("Error getting rewards info for block %v. Err: %v", startHeight+delta, err)
		}

		if hasRewards {
			ignoreIntervalWidth = false
			blocksBetweenRewards := averageOsmosisBlocksPerDay
			if lastKnownRewardsHeight != -1 {
				blocksBetweenRewards = int64(math.Abs(float64((startHeight + delta) - lastKnownRewardsHeight)))
			}

			lastKnownRewardsHeight = startHeight + delta
			startHeight = lastKnownRewardsHeight + blocksBetweenRewards
			delta = 0
			intervalWidth = 1000
			continue
		}

		// Search in the backwards direction as well, as long as we're not close to the current chain height
		if delta >= 1 && math.Abs(float64(startHeight+delta-lastKnownBlockHeight)) > 100 && startHeight-delta >= 1 {
			hasRewards, err := idxr.processRewardEpoch(startHeight-delta, rewardsDataChan, rpcClient, failedBlockHandler)

			if err != nil {
				config.Log.Fatalf("Error getting rewards info for block %v. Err: %v", startHeight-delta, err)
			}

			if hasRewards {
				ignoreIntervalWidth = false
				blocksBetweenRewards := averageOsmosisBlocksPerDay
				if lastKnownRewardsHeight != -1 {
					blocksBetweenRewards = int64(math.Abs(float64((startHeight - delta) - lastKnownRewardsHeight)))
				}

				lastKnownRewardsHeight = startHeight - delta
				startHeight = lastKnownRewardsHeight + blocksBetweenRewards
				delta = 0
				intervalWidth = 1000
			}

		}

		delta++
	}

	config.Log.Info("Finished rewards processing loop")

}

func (idxr *Indexer) processRewardEpoch(
	epoch int64,
	rewardsDataChan chan *osmosis.RewardsInfo,
	rpcClient osmosis.URIClient,
	failedBlockHandler core.FailedBlockHandler,
) (bool, error) {
	attempts := 1
	maxAttempts := 5

	_, hasRewards, err := idxr.indexOsmosisReward(rpcClient, epoch, rewardsDataChan)
	for err != nil && attempts < maxAttempts {
		attempts++
		// for some reason these need an exponential backoff....
		time.Sleep(time.Second * time.Duration(math.Pow(2, float64(attempts))))
		code, hasRewards, err := idxr.indexOsmosisReward(rpcClient, epoch, rewardsDataChan)
		if err != nil && attempts == maxAttempts {
			failedBlockHandler(epoch, code, err)
			return false, err
		} else if err == nil {
			return hasRewards, nil
		}
	}

	return hasRewards, err
}

// indexOsmosisReward returns true if rewards were found and sent to the channel for processing
func (idxr *Indexer) indexOsmosisReward(rpcClient osmosis.URIClient, epoch int64, rewardsDataChan chan *osmosis.RewardsInfo) (core.BlockProcessingFailure, bool, error) {
	rewards, err := rpcClient.GetEpochRewards(epoch)
	if err != nil {
		config.Log.Error(fmt.Sprintf("Error getting rewards for epoch %d\n", epoch), err)
		return core.OsmosisNodeRewardLookupError, false, err
	}

	if len(rewards) > 0 {
		config.Log.Info(fmt.Sprintf("Found %d Osmosis rewards at epoch %v", len(rewards), epoch))

		// Get the block time
		var blockTime time.Time
		result, err := rpc.GetBlock(idxr.cl, epoch)
		if err != nil {
			config.Log.Errorf("Error getting block info for block %v. Err: %v", epoch, err)
		} else {
			blockTime = result.Block.Time
		}

		batchSize := 10000
		for i := 0; i < len(rewards); i += batchSize {
			batchEnd := i + batchSize
			if batchEnd > len(rewards) {
				batchEnd = len(rewards) - 1
			}

			rewardBatch := rewards[i:batchEnd]

			// Send result to data chan to be inserted
			rewardsDataChan <- &osmosis.RewardsInfo{
				EpochBlockHeight: epoch,
				EpochBlockTime:   blockTime,
				Rewards:          rewardBatch,
			}
		}

		return 0, true, nil
	} else {
		config.Log.Debug(fmt.Sprintf("No Osmosis rewards at block height %v", epoch))
	}
	return 0, false, nil
}

// queryRPC will query the RPC endpoint
// this information will be parsed and converted into the domain objects we use for indexing this data.
// data is then passed to a channel to be consumed and inserted into the DB
func (idxr *Indexer) queryRPC(blockChan chan int64, dbDataChan chan *dbData, failedBlockHandler core.FailedBlockHandler) {
	maxAttempts := 5
	for blockToProcess := range blockChan {
		// attempt to process the block 5 times and then give up
		var attemptCount int
		for processBlock(idxr.cl, idxr.db, failedBlockHandler, dbDataChan, blockToProcess) != nil && attemptCount < maxAttempts {
			attemptCount++
			if attemptCount == maxAttempts {
				config.Log.Error(fmt.Sprintf("Failed to process block %v after %v attempts. Will add to failed blocks table", blockToProcess, maxAttempts))
				err := dbTypes.UpsertFailedBlock(idxr.db, blockToProcess, idxr.cfg.Lens.ChainID, idxr.cfg.Lens.ChainName)
				if err != nil {
					config.Log.Fatal(fmt.Sprintf("Failed to store that block %v failed. Not safe to continue.", blockToProcess), err)
				}
			}
		}
	}
}

func processBlock(cl *client.ChainClient, dbConn *gorm.DB, failedBlockHandler func(height int64, code core.BlockProcessingFailure, err error), dbDataChan chan *dbData, blockToProcess int64) error {
	// fmt.Printf("Querying RPC transactions for block %d\n", blockToProcess)
	newBlock := dbTypes.Block{Height: blockToProcess}
	var txDBWrappers []dbTypes.TxDBWrapper
	var blockTime *time.Time
	var err error
	errTypeURL := false

	txsEventResp, err := rpc.GetTxsByBlockHeight(cl, newBlock.Height)
	if err != nil {
		if strings.Contains(err.Error(), "unable to resolve type URL") {
			errTypeURL = true
		} else {
			config.Log.Errorf("Error getting transactions by block height (%v). Err: %v. Will reattempt", newBlock.Height, err)
			return err
		}
	}

	// There are two reasons this block would be hit
	// 1) The node might have pruned history resulting in a failed lookup. Recheck to see if the block was supposed to have TX results.
	// 2) The RPC endpoint (node we queried) doesn't recognize the type URL anymore, for an older type (e.g. on an archive node).
	if errTypeURL || len(txsEventResp.Txs) == 0 {
		// The node might have pruned history resulting in a failed lookup. Recheck to see if the block was supposed to have TX results.
		resBlockResults, err := rpc.GetBlockByHeight(cl, newBlock.Height)
		if err != nil || resBlockResults == nil {
			if err != nil && strings.Contains(err.Error(), "is not available, lowest height is") {
				failedBlockHandler(newBlock.Height, core.NodeMissingHistoryForBlock, err)
			} else {
				failedBlockHandler(newBlock.Height, core.BlockQueryError, err)
			}
			return nil
		} else if len(resBlockResults.TxsResults) > 0 {
			// The tx.height=X query said there were 0 TXs, but GetBlockByHeight() found some. When this happens
			// it is the same on every RPC node. Thus, we defer to the results from GetBlockByHeight.
			config.Log.Debugf("Falling back to secondary queries for block height %d", newBlock.Height)

			blockResults, err := rpc.GetBlock(cl, newBlock.Height)
			if err != nil {
				config.Log.Fatalf("Secondary RPC query failed, %d, %s", newBlock.Height, err)
			}

			txDBWrappers, blockTime, err = core.ProcessRPCBlockByHeightTXs(dbConn, cl, blockResults, resBlockResults)
			if err != nil {
				config.Log.Fatalf("Second query parser failed (ProcessRPCBlockByHeightTXs), %d, %s", newBlock.Height, err.Error())
				return err
			}
		}
	} else {
		txDBWrappers, blockTime, err = core.ProcessRPCTXs(dbConn, txsEventResp)
		if err != nil {
			config.Log.Error("ProcessRpcTxs: unhandled error", err)
			failedBlockHandler(blockToProcess, core.UnprocessableTxError, err)
		}
	}

	// Get the block time if we don't have TXs
	if blockTime == nil {
		result, err := rpc.GetBlock(cl, newBlock.Height)
		if err != nil {
			config.Log.Errorf("Error getting block info for block %v. Err: %v", newBlock.Height, err)
			return err
		}
		blockTime = &result.Block.Time
	}

	res := &dbData{
		txDBWrappers: txDBWrappers,
		blockTime:    *blockTime,
		blockHeight:  blockToProcess,
	}
	dbDataChan <- res

	return nil
}

type dbData struct {
	txDBWrappers []dbTypes.TxDBWrapper
	blockTime    time.Time
	blockHeight  int64
}

// doDBUpdates will read the data out of the db data chan that had been processed by the workers
// if this is a dry run, we will simply empty the channel and track progress
// otherwise we will index the data in the DB.
// it will also read rewars data and index that.
func (idxr *Indexer) doDBUpdates(wg *sync.WaitGroup, txDataChan chan *dbData, rewardsDataChan chan *osmosis.RewardsInfo, dbChainID uint) {
	blocksProcessed := 0
	dbWrites := 0
	dbReattempts := 0
	timeStart := time.Now()
	defer wg.Done()

	for {
		// break out of loop once both channels are fully consumed
		if rewardsDataChan == nil && txDataChan == nil {
			config.Log.Info("DB updates complete")
			break
		}

		select {
		// read rewards from the reward chan
		case rewardData, ok := <-rewardsDataChan:
			if !ok {
				rewardsDataChan = nil
				continue
			}
			dbWrites++
			config.Log.Info(fmt.Sprintf("Sending %v rewards at epoch %v to DB", len(rewardData.Rewards), rewardData.EpochBlockHeight))
			err := dbTypes.IndexOsmoRewards(idxr.db, idxr.dryRun, idxr.cfg.Lens.ChainID, idxr.cfg.Lens.ChainName, rewardData)
			if err != nil {
				// Do a single reattempt on failure
				dbReattempts++
				err = dbTypes.IndexOsmoRewards(idxr.db, idxr.dryRun, idxr.cfg.Lens.ChainID, idxr.cfg.Lens.ChainName, rewardData)
				if err != nil {
					config.Log.Fatal(fmt.Sprintf("Error storing rewards in DB at epoch %d", rewardData.EpochBlockHeight), err)
				}
			}

		// read tx data from the data chan
		case data, ok := <-txDataChan:
			if !ok {
				txDataChan = nil
				continue
			}
			dbWrites++
			// While debugging we'll sometimes want to turn off INSERTS to the DB
			// Note that this does not turn off certain reads or DB connections.
			if !idxr.dryRun {
				config.Log.Info(fmt.Sprintf("Indexing %v TXs from block %d", len(data.txDBWrappers), data.blockHeight))
				err := dbTypes.IndexNewBlock(idxr.db, data.blockHeight, data.blockTime, data.txDBWrappers, dbChainID)
				if err != nil {
					// Do a single reattempt on failure
					dbReattempts++
					err = dbTypes.IndexNewBlock(idxr.db, data.blockHeight, data.blockTime, data.txDBWrappers, dbChainID)
					if err != nil {
						config.Log.Fatal(fmt.Sprintf("Error indexing block %v.", data.blockHeight), err)
					}
				}
			} else {
				config.Log.Info(fmt.Sprintf("Processing block %d (dry run, block data will not be stored in DB).", data.blockHeight))
			}

			// Just measuring how many blocks/second we can process
			if idxr.cfg.Base.BlockTimer > 0 {
				blocksProcessed++
				if blocksProcessed%int(idxr.cfg.Base.BlockTimer) == 0 {
					totalTime := time.Since(timeStart)
					config.Log.Info(fmt.Sprintf("Processing %d blocks took %f seconds. %d total blocks have been processed.\n", idxr.cfg.Base.BlockTimer, totalTime.Seconds(), blocksProcessed))
					timeStart = time.Now()
				}
				if float64(dbReattempts)/float64(dbWrites) > .1 {
					config.Log.Fatalf("More than 10%% of the last %v DB writes have failed.", dbWrites)
				}
			}
		}
	}
}
