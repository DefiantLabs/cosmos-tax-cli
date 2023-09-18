package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type IndexConfig struct {
	Database           database
	API                api // deprecated in favor of lens.Rpc (at least in this app)
	ConfigFileLocation string
	Base               indexBase
	Log                log
	Lens               lens
	Client             client
}

type reindexingConf struct {
	ReindexMessageType string `mapstructure:"re-index-message-type"`
	PreventReattempts  bool   `mapstructure:"prevent-reattempts"`
}

type indexBase struct {
	reindexingConf
	throttlingBase
	API                       string
	StartBlock                int64  `mapstructure:"start-block"`
	EndBlock                  int64  `mapstructure:"end-block"`
	BlockInputFile            string `mapstructure:"block-input-file"`
	ReIndex                   bool
	RPCWorkers                int64 `mapstructure:"rpc-workers"`
	BlockTimer                int64 `mapstructure:"block-timer"`
	WaitForChain              bool  `mapstructure:"wait-for-chain"`
	WaitForChainDelay         int64 `mapstructure:"wait-for-chain-delay"`
	ChainIndexingEnabled      bool  `mapstructure:"index-chain"`
	ExitWhenCaughtUp          bool  `mapstructure:"exit-when-caught-up"`
	BlockEventIndexingEnabled bool  `mapstructure:"index-block-events"`
	Dry                       bool
	BlockEventsStartBlock     int64  `mapstructure:"block-events-start-block"`
	BlockEventsEndBlock       int64  `mapstructure:"block-events-end-block"`
	EpochEventIndexingEnabled bool   `mapstructure:"index-epoch-events"`
	EpochIndexingIdentifier   string `mapstructure:"epoch-indexing-identifier"`
	EpochEventsStartEpoch     int64  `mapstructure:"epoch-events-start-epoch"`
	EpochEventsEndEpoch       int64  `mapstructure:"epoch-events-end-epoch"`
	RPCRetryAttempts          int64  `mapstructure:"rpc-retry-attempts"`
	RPCRetryMaxWait           uint64 `mapstructure:"rpc-retry-max-wait"`
}

func SetupIndexSpecificFlags(conf *IndexConfig, cmd *cobra.Command) {
	// chain indexing
	cmd.PersistentFlags().BoolVar(&conf.Base.ChainIndexingEnabled, "base.index-chain", true, "enable chain indexing?")
	cmd.PersistentFlags().Int64Var(&conf.Base.StartBlock, "base.start-block", 0, "block to start indexing at (use -1 to resume from highest block indexed)")
	cmd.PersistentFlags().Int64Var(&conf.Base.EndBlock, "base.end-block", -1, "block to stop indexing at (use -1 to index indefinitely")
	cmd.PersistentFlags().StringVar(&conf.Base.BlockInputFile, "base.block-input-file", "", "A file location containing a JSON list of block heights to index. Will override start and end block flags.")
	cmd.PersistentFlags().BoolVar(&conf.Base.ReIndex, "base.reindex", false, "if true, this will re-attempt to index blocks we have already indexed (defaults to false)")
	cmd.PersistentFlags().BoolVar(&conf.Base.PreventReattempts, "base.prevent-reattempts", false, "prevent reattempts of failed blocks.")
	// block event indexing
	cmd.PersistentFlags().BoolVar(&conf.Base.BlockEventIndexingEnabled, "base.index-block-events", true, "enable block beginblocker and endblocker event indexing?")
	cmd.PersistentFlags().Int64Var(&conf.Base.BlockEventsStartBlock, "base.block-events-start-block", 0, "block to start indexing block events at")
	cmd.PersistentFlags().Int64Var(&conf.Base.BlockEventsEndBlock, "base.block-events-end-block", 0, "block to stop indexing block events at (use -1 to index indefinitely")
	// epoch event indexing
	cmd.PersistentFlags().BoolVar(&conf.Base.EpochEventIndexingEnabled, "base.index-epoch-events", false, "enable epoch beginblocker and endblocker event indexing?")
	cmd.PersistentFlags().Int64Var(&conf.Base.EpochEventsStartEpoch, "base.epoch-events-start-epoch", 0, "epoch number to start indexing block events at")
	cmd.PersistentFlags().Int64Var(&conf.Base.EpochEventsEndEpoch, "base.epoch-events-end-epoch", 0, "epoch number to stop indexing block events at")
	cmd.PersistentFlags().StringVar(&conf.Base.EpochIndexingIdentifier, "base.epoch-indexing-identifier", "", "epoch identifier to index")
	// other base setting
	cmd.PersistentFlags().BoolVar(&conf.Base.Dry, "base.dry", false, "index the chain but don't insert data in the DB.")
	cmd.PersistentFlags().StringVar(&conf.Base.API, "base.api", "", "node api endpoint")
	cmd.PersistentFlags().Int64Var(&conf.Base.RPCWorkers, "base.rpc-workers", 1, "rpc workers")
	cmd.PersistentFlags().BoolVar(&conf.Base.WaitForChain, "base.wait-for-chain", false, "wait for chain to be in sync?")
	cmd.PersistentFlags().Int64Var(&conf.Base.WaitForChainDelay, "base.wait-for-chain-delay", 10, "seconds to wait between each check for node to catch up to the chain")
	cmd.PersistentFlags().Int64Var(&conf.Base.BlockTimer, "base.block-timer", 10000, "print out how long it takes to process this many blocks")
	cmd.PersistentFlags().BoolVar(&conf.Base.ExitWhenCaughtUp, "base.exit-when-caught-up", true, "mainly used for Osmosis rewards indexing")
	cmd.PersistentFlags().Int64Var(&conf.Base.RPCRetryAttempts, "base.rpc-retry-attempts", 0, "number of RPC query retries to make")
	cmd.PersistentFlags().Uint64Var(&conf.Base.RPCRetryMaxWait, "base.rpc-retry-max-wait", 30, "max retry incremental backoff wait time in seconds")
}

func (conf *IndexConfig) Validate() error {
	err := validateDatabaseConf(conf.Database)

	if err != nil {
		return err
	}

	lensConf := conf.Lens

	lensConf, err = validateLensConf(lensConf)

	if err != nil {
		return err
	}

	conf.Lens = lensConf

	err = validateThrottlingConf(conf.Base.throttlingBase)

	if err != nil {
		return err
	}

	if conf.Base.StartBlock == 0 {
		return errors.New("base start-block must be set")
	}
	if conf.Base.EndBlock == 0 {
		return errors.New("base end-block must be set")
	}
	// If block event indexes are not valid, error
	if conf.Base.BlockEventsStartBlock < 0 {
		return errors.New("block-events-start-block must be valid")
	}
	if conf.Base.BlockEventsEndBlock < -1 {
		return errors.New("block-events-end-block must be valid")
	}

	// Check if API is provided, and if so, set default ports if not set
	if conf.Base.API != "" {
		if strings.Count(conf.Base.API, ":") != 2 {
			if strings.HasPrefix(conf.Base.API, "https:") {
				conf.Base.API = fmt.Sprintf("%s:443", conf.Base.API)
			} else if strings.HasPrefix(conf.Base.API, "http:") {
				conf.Base.API = fmt.Sprintf("%s:80", conf.Base.API)
			}
		}
	}

	return nil
}
