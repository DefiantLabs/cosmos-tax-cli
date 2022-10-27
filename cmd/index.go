package cmd

import (
	"errors"
	"fmt"
	"github.com/go-co-op/gocron"
	"log"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/core"
	indexerTx "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	dbTypes "github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis"
	"github.com/DefiantLabs/cosmos-tax-cli/rpc"
	"github.com/DefiantLabs/cosmos-tax-cli/tasks"

	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/lens/client"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func init() {
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
	db        *gorm.DB
	cl        *client.ChainClient
	scheduler *gocron.Scheduler
}

func setupIndexer() *Indexer {
	var idxr Indexer
	var err error
	//TODO: split out setup methods and only call necessary ones
	idxr.cfg, idxr.db, idxr.scheduler, err = setup(conf)
	if err != nil {
		log.Fatalf("Error during application setup. Err: %v", err)
	}

	core.ChainSpecificMessageTypeHandlerBootstrap(idxr.cfg.Lens.ChainID)

	//TODO may need to run this task in setup() so that we have a cold start functionality before the indexer starts
	_, err = idxr.scheduler.Every(6).Hours().Do(tasks.DenomUpsertTask, idxr.cfg.Lens.Rpc, idxr.db)
	if err != nil {
		config.Log.Error("Error scheduling denmon upsert task. Err: ", zap.Error(err))
	}
	idxr.scheduler.StartAsync()

	//Some chains do not have the denom metadata URL available on chain, so we do chain specific downloads instead.
	tasks.DoChainSpecificUpsertDenoms(idxr.db, idxr.cfg.Lens.ChainID)

	idxr.cl = config.GetLensClient(idxr.cfg.Lens)
	config.SetChainConfig(idxr.cfg.Base.AddressPrefix)

	//Depending on the app configuration, wait for the chain to catch up
	chainCatchingUp, err := rpc.IsCatchingUp(idxr.cl)
	for (idxr.cfg.Base.WaitForChain || idxr.cfg.Base.ExitWhenCaughtUp) && chainCatchingUp && err == nil {
		//Wait between status checks, don't spam the node with requests
		time.Sleep(time.Second * time.Duration(idxr.cfg.Base.WaitForChainDelay))
		chainCatchingUp, err = rpc.IsCatchingUp(idxr.cl)
	}
	if err != nil {
		config.Log.Fatal("Error querying chain status.", zap.Error(err))
	}

	return &idxr
}

func index(cmd *cobra.Command, args []string) {
	// Setup the indexer with config, db, and cl
	idxr := setupIndexer()
	dbConn, err := idxr.db.DB()
	if err != nil {
		config.Log.Fatal("Failed to connect to DB", zap.Error(err))
	}
	defer dbConn.Close()

	//blockChan are just the block heights; limit max jobs in the queue, otherwise this queue would contain one
	//item (block height) for every block on the entire blockchain we're indexing. Furthermore, once the queue
	//is close to empty, we will spin up a new thread to fill it up with new jobs.
	blockChan := make(chan int64, 10000)

	//This channel represents query job results for the RPC queries to Cosmos Nodes. Every time an RPC query
	//completes, the query result will be sent to this channel (for later processing by a different thread).
	//Realistically, I expect that RPC queries will be slower than our relational DB on the local network.
	//If RPC queries are faster than DB inserts this buffer will fill up.
	//We will periodically check the buffer size to monitor performance so we can optimize later.
	rpcQueryThreads := 4 //TODO: set this from the cfg
	blockTXsChan := make(chan *indexerTx.GetTxsEventResponseWrapper, 4*rpcQueryThreads)

	var txChanWaitGroup sync.WaitGroup // This group is to ensure we are done getting transactions before we close the TX channel
	//Spin up a (configurable) number of threads to query RPC endpoints for Transactions.
	//this is assumed to be the slowest process that allows concurrency and thus has the most dedicated go routines.
	for i := 0; i < rpcQueryThreads; i++ {
		txChanWaitGroup.Add(1)
		go func() {
			idxr.queryRpc(blockChan, blockTXsChan, core.HandleFailedBlock)
			txChanWaitGroup.Done()
		}()
	}

	// close the transaction chan once all transactions have been written to it
	go func() {
		txChanWaitGroup.Wait()
		close(blockTXsChan)
	}()

	var wg sync.WaitGroup // This group is to ensure we are done processing transactions (as well as osmo rewards) before returning

	//Start a thread to process transactions after the RPC querier retrieves them.
	wg.Add(1)
	go idxr.processTxs(&wg, blockTXsChan, core.HandleFailedBlock)

	//Osmosis specific indexing requirements. Osmosis distributes rewards to LP holders on a daily basis.
	if config.IsOsmosis(idxr.cfg) {
		wg.Add(1)
		go idxr.indexOsmosisRewards(&wg, core.HandleFailedBlock)
	}

	//Add jobs to the queue to be processed
	if !idxr.cfg.Base.OsmosisRewardsOnly {
		idxr.enqueueBlocksToProcess(blockChan)
		// close the block chan once all blocks have been written to it
		close(blockChan)
	}

	//If we error out in the main loop, this will block. Meaning we may not know of an error for 6 hours until last scheduled task stops
	idxr.scheduler.Stop()
	wg.Wait()
}

// enqueueBlocksToProcess will pass the blocks that need to be processed to the blockchannel
func (idxr *Indexer) enqueueBlocksToProcess(blockChan chan int64) {
	//Start at the last indexed block height (or the block height in the config, if set)
	currBlock := GetIndexerStartingHeight(idxr.cfg.Base.StartBlock, idxr.cl, idxr.db)

	//Don't index past this block no matter what
	lastBlock := idxr.cfg.Base.EndBlock
	var latestBlock int64 = math.MaxInt64

	//Add jobs to the queue to be processed
	for {
		//The program is configured to stop running after a set block height.
		//Generally this will only be done while debugging or if a particular block was incorrectly processed.
		if lastBlock != -1 && currBlock >= lastBlock {
			config.Log.Info("Hit the last block we're allowed to index, exiting.")
			return
		} else if idxr.cfg.Base.ExitWhenCaughtUp && currBlock >= latestBlock {
			config.Log.Info("Hit the last block we're allowed to index, exiting.")
			return
		}

		//The job queue is running out of jobs to process, see if the blockchain has produced any new blocks we haven't indexed yet.
		if len(blockChan) <= cap(blockChan)/4 {
			//This is the latest block height available on the Node.
			var err error
			latestBlock, err = rpc.GetLatestBlockHeight(idxr.cl)
			if err != nil {
				config.Log.Fatal("Error getting blockchain latest height. Err: %v", zap.Error(err))
			}

			//Throttling in case of hitting public APIs
			//TODO: track tx/s downloaded from each RPC endpoint and implement throttling limits per endpoint.
			if idxr.cfg.Base.Throttling != 0 {
				time.Sleep(time.Second * time.Duration(idxr.cfg.Base.Throttling))
			}

			//Already at the latest block, wait for the next block to be available.
			for currBlock <= latestBlock && currBlock <= lastBlock && len(blockChan) != cap(blockChan) {
				if idxr.cfg.Base.Throttling != 0 {
					time.Sleep(time.Second * time.Duration(idxr.cfg.Base.Throttling))
				}

				//Add the new block to the queue
				//fmt.Printf("Added block %d to the queue\n", currBlock)
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

func GetIndexerStartingHeight(configStartHeight int64, cl *client.ChainClient, db *gorm.DB) int64 {
	//Start the indexer at the configured value if one has been set. This starting height will be used
	//instead of searching the database to find the last indexed block.
	if configStartHeight != -1 {
		return configStartHeight
	}

	latestBlock, err := rpc.GetLatestBlockHeight(cl)
	if err != nil {
		log.Fatalf("Error getting blockchain latest height. Err: %v", err)
	}

	fmt.Println("Found latest block", latestBlock)
	highestIndexedBlock := dbTypes.GetHighestIndexedBlock(db)
	if highestIndexedBlock.Height < latestBlock {
		return highestIndexedBlock.Height + 1
	}

	return latestBlock
}

func (idxr *Indexer) indexOsmosisRewards(wg *sync.WaitGroup, failedBlockHandler func(height int64, code core.BlockProcessingFailure, err error)) {
	defer wg.Done()

	startHeight := idxr.cfg.Base.StartBlock
	if startHeight == -1 {
		startHeight = OsmosisGetRewardsStartIndexHeight(idxr.db, idxr.cfg.Lens.ChainID)
	}

	endHeight := idxr.cfg.Base.EndBlock
	if endHeight == -1 {
		var err error
		endHeight, err = rpc.GetLatestBlockHeight(idxr.cl)
		if err != nil {
			config.Log.Fatal("Error getting blockchain latest height.", zap.Error(err))
		}
	}

	rpcClient := osmosis.URIClient{
		Address: idxr.cl.Config.RPCAddr,
		Client:  &http.Client{},
	}

	maxAttempts := 5
	for epoch := startHeight; epoch <= endHeight; epoch++ {
		attempts := 1
		_, err := idxr.indexOsmosisReward(rpcClient, epoch)
		for err != nil && attempts < maxAttempts {
			attempts++
			// for some reason these need an exponential backoff....
			time.Sleep(time.Second * time.Duration(math.Pow(2, float64(attempts))))
			code, err := idxr.indexOsmosisReward(rpcClient, epoch)
			if err != nil && attempts == maxAttempts {
				failedBlockHandler(epoch, code, err)
			}
		}
	}
}

func (idxr *Indexer) indexOsmosisReward(rpcClient osmosis.URIClient, epoch int64) (core.BlockProcessingFailure, error) {
	rewards, err := rpcClient.GetEpochRewards(epoch)
	if err != nil {
		return core.OsmosisNodeRewardLookupError, err
	}

	if len(rewards) > 0 {
		err = dbTypes.IndexOsmoRewards(idxr.db, idxr.cfg.Lens.ChainID, idxr.cfg.Lens.ChainName, rewards)
		if err != nil {
			return core.OsmosisNodeRewardIndexError, err
		}
	}
	return 0, nil
}

func (idxr *Indexer) queryRpc(blockChan chan int64, blockTXsChan chan *indexerTx.GetTxsEventResponseWrapper, failedBlockHandler func(height int64, code core.BlockProcessingFailure, err error)) {
	maxAttempts := 5
	for blockToProcess := range blockChan {
		// attempt to process the block 5 times and then give up
		var attemptCount int
		for processBlock(idxr.cl, failedBlockHandler, blockTXsChan, blockToProcess) != nil && attemptCount < maxAttempts {
			attemptCount++
			if attemptCount == maxAttempts {
				config.Log.Error(fmt.Sprintf("Failed to process block %v after %v attempts. Will add to failed blocks table", blockToProcess, maxAttempts))
				err := dbTypes.UpsertFailedBlock(idxr.db, blockToProcess, idxr.cfg.Lens.ChainID, idxr.cfg.Lens.ChainName) // TODO: We could hang this off the DB connection...
				if err != nil {
					config.Log.Fatal(fmt.Sprintf("Failed to store that block %v failed. Not safe to continue.", blockToProcess), zap.Error(err))
				}
			}
		}
	}
}

func processBlock(cl *client.ChainClient, failedBlockHandler func(height int64, code core.BlockProcessingFailure, err error), blockTXsChan chan *indexerTx.GetTxsEventResponseWrapper, blockToProcess int64) error {
	//fmt.Printf("Querying RPC transactions for block %d\n", blockToProcess)
	newBlock := dbTypes.Block{Height: blockToProcess}

	//TODO: There is currently no pagination implemented!
	//TODO: Do something smarter than giving up when we encounter an error.
	txsEventResp, err := rpc.GetTxsByBlockHeight(cl, newBlock.Height)
	if err != nil {
		config.Log.Error("Error getting transactions by block height. Will reattempt", zap.Error(err))
		return err
	}

	if len(txsEventResp.Txs) == 0 {
		//The node might have pruned history resulting in a failed lookup. Recheck to see if the block was supposed to have TX results.
		blockResults, err := rpc.GetBlockByHeight(cl, newBlock.Height)
		if err != nil || blockResults == nil {
			failedBlockHandler(newBlock.Height, core.BlockQueryError, err)
		} else if len(blockResults.TxsResults) > 0 {
			//Two queries for the same block got a diff # of TXs. Though it is not guaranteed,
			//DeliverTx events typically make it into a block so this warrants manual investigation.
			//In this case, we couldn't look up TXs on the node but the Node's block has DeliverTx events,
			//so we should log this and manually review the block on e.g. mintscan or another tool.
			failedBlockHandler(newBlock.Height, core.NodeMissingBlockTxs, errors.New("node has DeliverTx results for block, but querying txs by height failed"))
		}
	}

	res := &indexerTx.GetTxsEventResponseWrapper{
		CosmosGetTxsEventResponse: txsEventResp,
		Height:                    blockToProcess,
	}
	blockTXsChan <- res
	return nil
}

func (idxr *Indexer) processTxs(wg *sync.WaitGroup, blockTXsChan chan *indexerTx.GetTxsEventResponseWrapper, failedBlockHandler func(height int64, code core.BlockProcessingFailure, err error)) {
	blocksProcessed := 0
	timeStart := time.Now()
	defer wg.Done()

	for txToProcess := range blockTXsChan {
		txDBWrappers, err := core.ProcessRpcTxs(idxr.db, txToProcess.CosmosGetTxsEventResponse)
		if err != nil {
			config.Log.Error("ProcessRpcTxs: unhandled error", zap.Error(err))
			failedBlockHandler(txToProcess.Height, core.UnprocessableTxError, err)
		}

		//While debugging we'll sometimes want to turn off INSERTS to the DB
		//Note that this does not turn off certain reads or DB connections.
		if idxr.cfg.Base.IndexingEnabled {
			config.Log.Info(fmt.Sprintf("Indexing block %d, threaded.\n", txToProcess.Height))
			err = dbTypes.IndexNewBlock(idxr.db, txToProcess.Height, txDBWrappers, idxr.cfg.Lens.ChainID, idxr.cfg.Lens.ChainName)
			if err != nil {
				if err != nil {
					log.Fatalf("Error indexing block %v. Err: %v", txToProcess.Height, err)
				}
			}
		}

		//Just measuring how many blocks/second we can process
		if idxr.cfg.Base.BlockTimer > 0 {
			blocksProcessed++
			if blocksProcessed%int(idxr.cfg.Base.BlockTimer) == 0 {
				totalTime := time.Since(timeStart)
				config.Log.Info(fmt.Sprintf("Processing %d blocks took %f seconds. %d total blocks have been processed.\n", idxr.cfg.Base.BlockTimer, totalTime.Seconds(), blocksProcessed))
				timeStart = time.Now()
			}
		}
	}
}
