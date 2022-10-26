package cmd

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/core"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis"
	"github.com/DefiantLabs/cosmos-tax-cli/rpc"
	"github.com/DefiantLabs/cosmos-tax-cli/tasks"
	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/lens/client"
	"go.uber.org/zap"
	"gorm.io/gorm"

	configHelpers "github.com/DefiantLabs/cosmos-tax-cli/config"
	indexerTx "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	dbTypes "github.com/DefiantLabs/cosmos-tax-cli/db"
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
	Run: func(cmd *cobra.Command, args []string) {
		//TODO: split out setup methods and only call necessary ones
		config, db, scheduler, err := setup(conf)
		cobra.CheckErr(err)
		if err != nil {
			log.Fatalf("Error during application setup. Err: %v", err)
		}

		apiHost := config.Lens.Rpc
		dbConn, _ := db.DB()
		defer dbConn.Close()

		core.ChainSpecificMessageTypeHandlerBootstrap(config.Lens.ChainID)

		//TODO may need to run this task in setup() so that we have a cold start functionality before the indexer starts
		_, err = scheduler.Every(6).Hours().Do(tasks.DenomUpsertTask, apiHost, db)
		if err != nil {
			log.Println("Error scheduling denmon upsert task. Err: ", err)
		}
		scheduler.StartAsync()

		//Some chains do not have the denom metadata URL available on chain, so we do chain specific downloads instead.
		tasks.DoChainSpecificUpsertDenoms(db, config.Lens.ChainID)

		cl := configHelpers.GetLensClient(config.Lens)
		configHelpers.SetChainConfig(config.Base.AddressPrefix)

		//Depending on the app configuration, wait for the chain to catch up
		chainCatchingUp, err := rpc.IsCatchingUp(cl)
		for (config.Base.WaitForChain || config.Base.ExitWhenCaughtUp) && chainCatchingUp && err == nil {
			//Wait between status checks, don't spam the node with requests
			time.Sleep(time.Second * time.Duration(config.Base.WaitForChainDelay))
			chainCatchingUp, err = rpc.IsCatchingUp(cl)
		}
		if err != nil {
			log.Fatalf("Error querying chain status. Err: %v", err)
		}

		//Jobs are just the block height; limit max jobs in the queue, otherwise this queue would contain one
		//item (block height) for every block on the entire blockchain we're indexing. Furthermore, once the queue
		//is close to empty, we will spin up a new thread to fill it up with new jobs.
		blockHeightToProcess := make(chan int64, 10000)

		//This channel represents query job results for the RPC queries to Cosmos Nodes. Every time an RPC query
		//completes, the query result will be sent to this channel (for later processing by a different thread).
		//Realistically, I expect that RPC queries will be slower than our relational DB on the local network.
		//If RPC queries are faster than DB inserts this buffer will fill up.
		//We will periodically check the buffer size to monitor performance so we can optimize later.
		jobResultsChannel := make(chan *indexerTx.GetTxsEventResponseWrapper, 10)
		rpcQueryThreads := 4

		//Spin up a (configurable) number of threads to query RPC endpoints for Transactions.
		for i := 0; i < rpcQueryThreads; i++ {
			go QueryRpc(blockHeightToProcess, jobResultsChannel, cl, core.HandleFailedBlock)
		}

		//Start a thread to process transactions after the RPC querier retrieves them.
		go ProcessTxs(jobResultsChannel, config.Base.BlockTimer, config.Base.IndexingEnabled, db, config.Lens.ChainID, config.Lens.ChainName, core.HandleFailedBlock)

		//Start at the last indexed block height (or the block height in the config, if set)
		currBlock := GetIndexerStartingHeight(config.Base.StartBlock, cl, db)
		//Don't index past this block no matter what
		lastBlock := config.Base.EndBlock
		var wg sync.WaitGroup

		//Osmosis specific indexing requirements. Osmosis distributes rewards to LP holders on a daily basis.
		if configHelpers.IsOsmosis(config) {
			rewardsIndexerStartHeight := config.Base.StartBlock
			if rewardsIndexerStartHeight == -1 {
				rewardsIndexerStartHeight = OsmosisGetRewardsStartIndexHeight(db, config.Lens.ChainID)
			}

			latestOsmosisBlock, err := rpc.GetLatestBlockHeight(cl)
			if err != nil {
				log.Fatalf("Error getting blockchain latest height. Err: %v", err)
			}

			rpcClient := osmosis.URIClient{
				Address: cl.Config.RPCAddr,
				Client:  &http.Client{},
			}

			go IndexOsmosisRewards(&wg, db, rpcClient, config.Lens.ChainID, config.Lens.ChainName, rewardsIndexerStartHeight, latestOsmosisBlock, core.HandleFailedBlock)
		}

		wg.Add(1)
		var latestBlock int64 = math.MaxInt64

		//Add jobs to the queue to be processed
		for !config.Base.OsmosisRewardsOnly {
			//The program is configured to stop running after a set block height.
			//Generally this will only be done while debugging or if a particular block was incorrectly processed.
			if lastBlock != -1 && currBlock >= lastBlock {
				fmt.Println("Hit the last block we're allowed to index, exiting.")
				break
			} else if config.Base.ExitWhenCaughtUp && currBlock >= latestBlock {
				fmt.Println("Hit the last block we're allowed to index, exiting.")
				break
			}

			//The job queue is running out of jobs to process, see if the blockchain has produced any new blocks we haven't indexed yet.
			if len(blockHeightToProcess) <= cap(blockHeightToProcess)/4 {
				//fmt.Println("Filling jobs queue")

				//This is the latest block height available on the Node.
				latestBlock, err := rpc.GetLatestBlockHeight(cl)
				if err != nil {
					log.Fatalf("Error getting blockchain latest height. Err: %v", err)
				}

				//Throttling in case of hitting public APIs
				//TODO: track tx/s downloaded from each RPC endpoint and implement throttling limits per endpoint.
				if config.Base.Throttling != 0 {
					time.Sleep(time.Second * time.Duration(config.Base.Throttling))
				}

				//Already at the latest block, wait for the next block to be available.
				for currBlock <= latestBlock && currBlock <= lastBlock && len(blockHeightToProcess) != cap(blockHeightToProcess) {

					if config.Base.Throttling != 0 {
						time.Sleep(time.Second * time.Duration(config.Base.Throttling))
					}

					//Add the new block to the queue
					//fmt.Printf("Added block %d to the queue\n", currBlock)
					blockHeightToProcess <- currBlock
					currBlock++
				}
			}
		}

		//if len(ch) == cap(ch) {

		//If we error out in the main loop, this will block. Meaning we may not know of an error for 6 hours until last scheduled task stops
		scheduler.Stop()
		wg.Wait()
	},
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

func IndexOsmosisRewards(
	wg *sync.WaitGroup,
	db *gorm.DB,
	rpcClient osmosis.URIClient,
	chainID string,
	chainName string,
	startHeight int64,
	endHeight int64,
	failedBlockHandler func(height int64, code core.BlockProcessingFailure, err error),
) {
	defer wg.Done()

	for epoch := startHeight; epoch <= endHeight; epoch++ {
		rewards, indexErr := rpcClient.GetEpochRewards(epoch)
		if indexErr != nil {
			failedBlockHandler(epoch, core.OsmosisNodeRewardLookupError, indexErr)
		}

		if len(rewards) > 0 {
			indexErr = dbTypes.IndexOsmoRewards(db, chainID, chainName, rewards)
			if indexErr != nil {
				failedBlockHandler(epoch, core.OsmosisNodeRewardIndexError, indexErr)
			}
		}
	}
}

func QueryRpc(blockHeightToProcess chan int64, results chan *indexerTx.GetTxsEventResponseWrapper, cl *client.ChainClient, failedBlockHandler func(height int64, code core.BlockProcessingFailure, err error)) {
	reprocessBlock := int64(0)

	for {
		blockToProcess := reprocessBlock

		if reprocessBlock == 0 {
			blockToProcess = <-blockHeightToProcess
			reprocessBlock = 0
		}
		//fmt.Printf("Querying RPC transactions for block %d\n", blockToProcess)
		newBlock := dbTypes.Block{Height: blockToProcess}

		//TODO: There is currently no pagination implemented!
		//TODO: Do something smarter than giving up when we encounter an error.
		txsEventResp, err := rpc.GetTxsByBlockHeight(cl, newBlock.Height)
		if err != nil {
			fmt.Println("Error getting transactions by block height", err)
			reprocessBlock = newBlock.Height
			continue
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
		results <- res
	}
}

func ProcessTxs(results chan *indexerTx.GetTxsEventResponseWrapper, numBlocksTimed int64, indexingEnabled bool, db *gorm.DB, chainID string, chainName string, failedBlockHandler func(height int64, code core.BlockProcessingFailure, err error)) {
	blocksProcessed := 0
	timeStart := time.Now()

	for {
		txToProcess := <-results
		txDBWrappers, err := core.ProcessRpcTxs(db, txToProcess.CosmosGetTxsEventResponse)
		if err != nil {
			config.Logger.Error("ProcessRpcTxs: unhandled error", zap.Error(err))
			failedBlockHandler(txToProcess.Height, core.UnprocessableTxError, err)
		}

		//While debugging we'll sometimes want to turn off INSERTS to the DB
		//Note that this does not turn off certain reads or DB connections.
		if indexingEnabled {
			log.Printf("Indexing block %d, threaded.\n", txToProcess.Height)
			err = dbTypes.IndexNewBlock(db, txToProcess.Height, txDBWrappers, chainID, chainName)
			if err != nil {
				if err != nil {
					log.Fatalf("Error indexing block %v. Err: %v", txToProcess.Height, err)
				}
			}
		}

		//Just measuring how many blocks/second we can process
		if numBlocksTimed > 0 {
			blocksProcessed++
			if blocksProcessed%int(numBlocksTimed) == 0 {
				totalTime := time.Since(timeStart)
				fmt.Printf("Processing %d blocks took %f seconds. %d total blocks have been processed.\n", numBlocksTimed, totalTime.Seconds(), blocksProcessed)
				timeStart = time.Now()
			}
		}
	}
}
