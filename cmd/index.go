package cmd

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-co-op/gocron"

	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
	"github.com/DefiantLabs/cosmos-tax-cli-private/core"
	dbTypes "github.com/DefiantLabs/cosmos-tax-cli-private/db"
	"github.com/DefiantLabs/cosmos-tax-cli-private/osmosis"
	"github.com/DefiantLabs/cosmos-tax-cli-private/rpc"
	"github.com/DefiantLabs/cosmos-tax-cli-private/tasks"

	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/lens/client"
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
		idxr.scheduler.StartAsync()
	}

	// Some chains do not have the denom metadata URL available on chain, so we do chain specific downloads instead.
	tasks.DoChainSpecificUpsertDenoms(idxr.db, idxr.cfg.Lens.ChainID)
	idxr.cl = config.GetLensClient(idxr.cfg.Lens)

	// Depending on the app configuration, wait for the chain to catch up
	chainCatchingUp, err := rpc.IsCatchingUp(idxr.cl)
	for (idxr.cfg.Base.WaitForChain || idxr.cfg.Base.ExitWhenCaughtUp) && chainCatchingUp && err == nil {
		// Wait between status checks, don't spam the node with requests
		config.Log.Debug("Chain is still catching up, please wait or disable check in config.")
		time.Sleep(time.Second * time.Duration(idxr.cfg.Base.WaitForChainDelay))
		chainCatchingUp, err = rpc.IsCatchingUp(idxr.cl)
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
	for i := 0; i < rpcQueryThreads; i++ {
		txChanWaitGroup.Add(1)
		go func() {
			idxr.queryRPC(blockChan, txDataChan, core.HandleFailedBlock)
			txChanWaitGroup.Done()
		}()
	}

	// close the transaction chan once all transactions have been written to it
	go func() {
		txChanWaitGroup.Wait()
		close(txDataChan)
	}()

	var wg sync.WaitGroup // This group is to ensure we are done processing transactions (as well as osmo rewards) before returning

	// Osmosis specific indexing requirements. Osmosis distributes rewards to LP holders on a daily basis.
	rewardsDataChan := make(chan []*osmosis.Rewards, 4*rpcQueryThreads)
	if config.IsOsmosis(idxr.cfg) && idxr.cfg.Base.RewardIndexingEnabled {
		wg.Add(1)
		go idxr.indexOsmosisRewards(&wg, core.HandleFailedBlock, rewardsDataChan)
	} else {
		close(rewardsDataChan)
	}

	// Start a thread to index the data queried from the chain.
	if idxr.cfg.Base.IndexingEnabled {
		wg.Add(1)
		go idxr.doDBUpdates(&wg, txDataChan, rewardsDataChan)
	}

	// Add jobs to the queue to be processed
	if idxr.cfg.Base.IndexingEnabled {
		chain := dbTypes.Chain{
			ChainID: idxr.cfg.Lens.ChainID,
			Name:    idxr.cfg.Lens.ChainName,
		}
		chainID, err := dbTypes.GetDBChainID(idxr.db, chain)
		if err != nil {
			config.Log.Fatal("Failed to add/create chain in DB", err)
		}
		if reindexMsgType != "" {
			idxr.enqueueBlocksToProcessByMsgType(blockChan, chainID, reindexMsgType)
		} else {
			idxr.enqueueBlocksToProcess(blockChan, chainID)
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
	err := db.Raw(`SELECT count(*) > 0 FROM blocks WHERE height = ?::int AND blockchain_id = ?::int AND indexed = true;`, blockHeight, chainID).Row().Scan(&exists)
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

func (idxr *Indexer) indexOsmosisRewards(wg *sync.WaitGroup, failedBlockHandler core.FailedBlockHandler, rewardsDataChan chan []*osmosis.Rewards) {
	defer wg.Done()
	defer close(rewardsDataChan)

	startHeight := idxr.cfg.Base.RewardStartBlock
	if startHeight == -1 {
		startHeight = OsmosisGetRewardsStartIndexHeight(idxr.db, idxr.cfg.Lens.ChainID)
	}

	endHeight := idxr.cfg.Base.RewardEndBlock
	if endHeight == -1 {
		var err error
		endHeight, err = rpc.GetLatestBlockHeight(idxr.cl)
		if err != nil {
			config.Log.Fatal("Error getting blockchain latest height.", err)
		}
	}

	config.Log.Infof("Indexing Rewards from block: %v to %v", idxr.cfg.Base.RewardStartBlock, endHeight)

	rpcClient := osmosis.URIClient{
		Address: idxr.cl.Config.RPCAddr,
		Client:  &http.Client{},
	}

	maxAttempts := 5
	for epoch := startHeight; epoch <= endHeight; epoch++ {
		attempts := 1
		_, err := idxr.indexOsmosisReward(rpcClient, epoch, rewardsDataChan)
		for err != nil && attempts < maxAttempts {
			attempts++
			// for some reason these need an exponential backoff....
			time.Sleep(time.Second * time.Duration(math.Pow(2, float64(attempts))))
			code, err := idxr.indexOsmosisReward(rpcClient, epoch, rewardsDataChan)
			if err != nil && attempts == maxAttempts {
				failedBlockHandler(epoch, code, err)
			}
		}
	}
}

func (idxr *Indexer) indexOsmosisReward(rpcClient osmosis.URIClient, epoch int64, rewardsDataChan chan []*osmosis.Rewards) (core.BlockProcessingFailure, error) {
	config.Log.Debug(fmt.Sprintf("Getting rewards for epoch %v", epoch))
	rewards, err := rpcClient.GetEpochRewards(epoch)
	if err != nil {
		config.Log.Error(fmt.Sprintf("Error getting rewards for epoch %d\n", epoch), err)
		return core.OsmosisNodeRewardLookupError, err
	}

	if len(rewards) > 0 {
		rewardsDataChan <- rewards
	}
	return 0, nil
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
				// TODO: When we work on resume functionality, we need to build in a way to clear blocks out of the failure table when they succeed
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

	txsEventResp, err := rpc.GetTxsByBlockHeight(cl, newBlock.Height)
	if err != nil {
		config.Log.Errorf("Error getting transactions by block height (%v). Err: %v. Will reattempt", newBlock.Height, err)
		return err
	}

	if len(txsEventResp.Txs) == 0 {
		// The node might have pruned history resulting in a failed lookup. Recheck to see if the block was supposed to have TX results.
		blockResults, err := rpc.GetBlockByHeight(cl, newBlock.Height)
		if err != nil || blockResults == nil {
			if err != nil && strings.Contains(err.Error(), "is not available, lowest height is") {
				failedBlockHandler(newBlock.Height, core.NodeMissingHistoryForBlock, err)
			} else {
				failedBlockHandler(newBlock.Height, core.BlockQueryError, err)
			}
			return nil
		} else if len(blockResults.TxsResults) > 0 {
			// Two queries for the same block got a diff # of TXs. Though it is not guaranteed,
			// DeliverTx events typically make it into a block so this warrants manual investigation.
			// In this case, we couldn't look up TXs on the node but the Node's block has DeliverTx events,
			// so we should log this and manually review the block on e.g. mintscan or another tool.
			config.Log.Fatalf("Two queries for the same block (%v) got a diff # of TXs.", newBlock.Height)
		}
	}

	txDBWrappers, blockTime, err := core.ProcessRPCTXs(dbConn, txsEventResp)
	if err != nil {
		config.Log.Error("ProcessRpcTxs: unhandled error", err)
		failedBlockHandler(blockToProcess, core.UnprocessableTxError, err)
	}

	res := &dbData{
		txDBWrappers: txDBWrappers,
		blockTime:    blockTime,
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
func (idxr *Indexer) doDBUpdates(wg *sync.WaitGroup, txDataChan chan *dbData, rewardsDataChan chan []*osmosis.Rewards) {
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
			config.Log.Info(fmt.Sprintf("Found %v rewards at epoch %v, sending to DB", len(rewardData), rewardData[0].EpochBlockHeight))
			err := dbTypes.IndexOsmoRewards(idxr.db, idxr.dryRun, idxr.cfg.Lens.ChainID, idxr.cfg.Lens.ChainName, rewardData)
			if err != nil {
				// Do a single reattempt on failure
				dbReattempts++
				err = dbTypes.IndexOsmoRewards(idxr.db, idxr.dryRun, idxr.cfg.Lens.ChainID, idxr.cfg.Lens.ChainName, rewardData)
				if err != nil {
					config.Log.Fatal("Error storing rewards in DB.", err)
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
				config.Log.Info(fmt.Sprintf("Indexing %v TXs from block %d.", len(data.txDBWrappers), data.blockHeight))
				err := dbTypes.IndexNewBlock(idxr.db, data.blockHeight, data.blockTime, data.txDBWrappers, idxr.cfg.Lens.ChainID, idxr.cfg.Lens.ChainName)
				if err != nil {
					// Do a single reattempt on failure
					dbReattempts++
					err = dbTypes.IndexNewBlock(idxr.db, data.blockHeight, data.blockTime, data.txDBWrappers, idxr.cfg.Lens.ChainID, idxr.cfg.Lens.ChainName)
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
