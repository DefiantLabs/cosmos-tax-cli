# Cosmos Tax CLI

This application indexes a Cosmos chain to a standardized DB schema with the following goals in mind:
* A generalized DB schema that could potentially work with all Cosmos SDK Chains
* A focus on making it easy to correlate transactions with addresses and store relevant data

In addition to indexing a chain, this tool can also query the indexed data to find all transactions associated with
one or more addresses on a given chain. This data can be returned in one of multiple formatted CSVs designed
to integrate with tools for determining an individuals tax liability as a result of these transactions.

This CLI tool for indexing and querying the chain is also accompanied by a webserver (found in the `client` directory)
which can allow a frontend UI to request a CSV. Defiant has created its own version of this frontend which can be found
[here](https://github.com/DefiantLabs/sycamore)

## Getting Started
The typical workflow for using the tax CLI is to index the chain for a given period of time and persist this data
in a database to allow multiple users to query for their data. This section will cover the end-to-end process of
indexing and querying data.

Note: While this readme includes up-to-date information about using the tool, the tool itself also contains some
internal documentation. Running `go run main.go` without any arguments should display the help text for the application
as well as a list of the flags that can be used. Additionally, calling any of the commands with the `--help` flag
will display their help text.

### Prerequisites
Before you can begin indexing a chain, you must first configure the applications dependencies.

#### PostgreSQL
The app requires a postgresql server with an established database and an owner user/role with password login.
A simple example for setting up a containerized database locally can be found [here](https://towardsdatascience.com/local-development-set-up-of-postgresql-with-docker-c022632f13ea).

#### Go
The app is written and Go and you will need to build from source. This requires a system install of Go 1.19.
Instruction for installing and configuring Go can be found [here](https://go.dev/doc/install).

### Indexing
At this point you should be ready to run the indexer. The indexer is what adds records to the Postgres DB and required
in order to index all the data that one might want to query.

To run the indexer, simply use the `index` command `go run main.go index --config {{PATH_TO_CONFIG}}` where `{{PATH_TO_CONFIG}}`
is replaced with the path to a local config file used to configure the application. An example config file can be found
[here](https://github.com/DefiantLabs/cosmos-tax-cli-private/blob/main/config.toml.example). For more information about
the config file, as well as the CLI flags which can be used to override config settings, please refer to the more
in-depth [config](#config) section below.

**CMD help text**
```
$ go run main.go index --help
Indexes the Cosmos-based blockchain according to the configurations found on the command line
        or in the specified config file. Indexes taxable events into a database for easy querying. It is
        highly recommended to keep this command running as a background service to keep your index up to date.

Usage:
  cosmos-tax-cli-private index [flags]

Flags:
  -h, --help                           help for index
      --re-index-message-type string   If specified, the indexer will reindex only the blocks containing the message type provided.

Global Flags:
      --base.api string             node api endpoint
      --base.dry                    index the chain but don't insert data in the DB.
      --base.endBlock int           block to stop indexing at (use -1 to index indefinitely (default -1)
      --base.index                  enable indexing? (default true)
      --base.indexRewards           enable osmosis reward indexing? (default true)
      --base.preventReattempts      prevent reattempts of failed blocks.
      --base.rewardEndBlock int     block to stop indexing rewards at (use -1 to index indefinitely
      --base.rewardStartBlock int   block to start indexing rewards at
      --base.rpcworkers int         rpc workers (default 1)
      --base.startBlock int         block to start indexing at (use -1 to resume from highest block indexed)
      --base.throttling float       throttle delay (default 0.5)
      --base.waitforchain           wait for chain to be in sync?
      --config string               config file (default is $HOME/.cosmos-tax-cli-private/config.yaml)
      --db.database string          database name
      --db.host string              database host
      --db.loglevel string          database loglevel
      --db.password string          database password
      --db.port string              database port (default "5432")
      --db.user string              database user
      --lens.accountPrefix string   lens account prefix
      --lens.chainID string         lens chain ID
      --lens.chainName string       lens chain name
      --lens.rpc string             node rpc endpoint
      --log.level string            log level (default "info")
      --log.path string             log path (default is $HOME/.cosmos-tax-cli-private/logs.txt
      --log.pretty                  pretty logs

```

### Querying
Once the chain has been indexed, data can be queried using the `query` command. As with indexing, a config file is provided
to configure the application. In addition, the addresses you wish to query can be provided as a comma separated list:

`go run main.go query --address "address1,address2" --config {{PATH_TO_CONFIG}}`

For more information about the config, please refer to the more in-depth
[config](#config) section below.

## Config
A config file can be used to configure the tool. The config is broken into 4 sections:
- [Log](#log)
- [Database](#database)
- [Base](#base)
- [Lens](#lens)

**Note: Ultimately all the settings available in the config will be available via CLI flags.
To see a list of the currently supported flags, simply display the application help text:
`go run main.go`**

### Log
#### Level
This setting is used to determine which level of logs will be included. The available levels include
- `Debug`
- `Info` (default)
- `Warn`
- `Error`
- `Fatal`
- `Panic`

#### Path
Logs will always be written to standard out, but if desired they can also be written to a file.
To do this, simply provide a path to a file.

#### Pretty
We use a logging package called [ZeroLog](https://github.com/rs/zerolog). To take advantage of their "pretty" logging
you can set `Pretty` to true.

### Database
The config options for the database are largely self-explanatory.
`Host`: The address needed to connect to the DB.
`Port`: The port needed to connect to the DB.
`Database`: The name of the database to connect to.
`User`: The DB username
`Password`: The password for the DB user.

#### LogLevel
This is a feature built into [gorm](https://gorm.io) to allow for logging of query information. This can be helpful for troubleshooting
performance issues.

Available log levels include:
- `silent` (default)
- `info`
- `warn`
- `error`

### Base
These are the core settings for the tool

#### API
Node API endpoint for querying chain information.

#### StartBlock
The block height to start indexing at.

#### EndBlock
The block height to stop indexing at. If set to '-1' indexing will keep running and keep pace with the chain.

#### Throttling
The minimum number of seconds per block. Higher number, will be slower. A value of 1 will result in approximately
1 block per second being indexed.

#### RPCWorkers
The number of RPC workers. This should typically be a similar order of magnitude to the number of cpu cores available
to the indexer.

#### BlockTimer
The indexer will track how long it takes ot process this number of blocks.

#### WaitForChain
// TODO: add details but also improve code behaior here.

#### WaitForChainDelay
// TODO: add details but also improve code behaior here.

#### IndexingEnabled
If false, the indexer won't actually index the chain. This may be desirable if your goal is only to index rewards.

#### ExitWhenCaughtUp
// TODO: add details but also improve code behaior here.

#### RewardIndexingEnabled
If true, the indexer will attempt to index osmosis rewards.

#### RewardStartBlock
The block height to start indexing rewards. (will default to start block if not set)

#### RewardEndBlock
The block height to stop indexing rewards. (will default to end block if not set)

#### Dry
If true, the indexer will read the chain but won't actually write data to the database.

#### CreateCSVFile
Defaults to true. If false, queried data will be printed to standard out instead of creating a CSV.

#### CSVFile
Configures the name of the CSV file generated by the query cmd.

### Lens
This tool uses a [fork of Lens](https://github.com/DefiantLabs/lens) to read data from the blockchain. This is built
into the application and does not need to be installed separately.

#### RPC
The node RPC endpoint

#### AccountPrefix
Lens account prefix

#### ChainID
The ID of the chain being indexed

#### ChainName
The name of the chain being indexed


## Supported message types
When indexing the chain, we need to parse individual messages in order to determine their significance.
Some messages (transfers for example) have tax implication, whereas other (bonding/unbonding funds) do not.
While we are constantly adding to our list of supported messages, we do not currently support every possible
message on every chain.

To view the always up-to-date complete list of supported messages, please consult the code [here](https://github.com/DefiantLabs/cosmos-tax-cli-private/blob/main/core/tx.go)

Here is a list of what we support at the time of this writing.

### Cosmos Modules
#### Authz
```go
	MsgExec   = "/cosmos.authz.v1beta1.MsgExec"
	MsgGrant  = "/cosmos.authz.v1beta1.MsgGrant"
	MsgRevoke = "/cosmos.authz.v1beta1.MsgRevoke"
```

#### Bank
```go
	MsgSendV0 = "MsgSend"
	MsgSend   = "/cosmos.bank.v1beta1.MsgSend"
	MsgMultiSendV0 = "MsgMultiSend"
	MsgMultiSend   = "/cosmos.bank.v1beta1.MsgMultiSend"
```

#### Distribution
```go
	MsgFundCommunityPool           = "/cosmos.distribution.v1beta1.MsgFundCommunityPool"
	MsgWithdrawValidatorCommission = "/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission"
	MsgWithdrawDelegatorReward     = "/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward"
	MsgWithdrawRewards             = "withdraw-rewards"
	MsgSetWithdrawAddress          = "/cosmos.distribution.v1beta1.MsgSetWithdrawAddress"
```

#### Gov
```go
	MsgVote           = "/cosmos.gov.v1beta1.MsgVote"
	MsgDeposit        = "/cosmos.gov.v1beta1.MsgDeposit"        // handle additional deposits to the given proposal
	MsgSubmitProposal = "/cosmos.gov.v1beta1.MsgSubmitProposal" // handle the initial deposit for the proposer
	MsgVoteWeighted   = "/cosmos.gov.v1beta1.MsgVoteWeighted"
```

#### IBC
```go
	MsgTransfer = "/ibc.applications.transfer.v1.MsgTransfer"
	MsgAcknowledgement    = "/ibc.core.channel.v1.MsgAcknowledgement"
	MsgChannelOpenTry     = "/ibc.core.channel.v1.MsgChannelOpenTry"
	MsgChannelOpenConfirm = "/ibc.core.channel.v1.MsgChannelOpenConfirm"
	MsgChannelOpenInit    = "/ibc.core.channel.v1.MsgChannelOpenInit"
	MsgChannelOpenAck     = "/ibc.core.channel.v1.MsgChannelOpenAck"
	MsgRecvPacket         = "/ibc.core.channel.v1.MsgRecvPacket"
	MsgTimeout            = "/ibc.core.channel.v1.MsgTimeout"
	MsgTimeoutOnClose     = "/ibc.core.channel.v1.MsgTimeoutOnClose"
	MsgConnectionOpenTry     = "/ibc.core.connection.v1.MsgConnectionOpenTry"
	MsgConnectionOpenConfirm = "/ibc.core.connection.v1.MsgConnectionOpenConfirm"
	MsgConnectionOpenInit    = "/ibc.core.connection.v1.MsgConnectionOpenInit"
	MsgConnectionOpenAck     = "/ibc.core.connection.v1.MsgConnectionOpenAck"
	MsgCreateClient = "/ibc.core.client.v1.MsgCreateClient"
	MsgUpdateClient = "/ibc.core.client.v1.MsgUpdateClient"
```

#### Slashing
```go
	MsgUnjail       = "/cosmos.slashing.v1beta1.MsgUnjail"
	MsgUpdateParams = "/cosmos.slashing.v1beta1.MsgUpdateParams"
```

#### Staking
```go
	MsgDelegate        = "/cosmos.staking.v1beta1.MsgDelegate"
	MsgUndelegate      = "/cosmos.staking.v1beta1.MsgUndelegate"
	MsgBeginRedelegate = "/cosmos.staking.v1beta1.MsgBeginRedelegate"
	MsgCreateValidator = "/cosmos.staking.v1beta1.MsgCreateValidator"
	MsgEditValidator   = "/cosmos.staking.v1beta1.MsgEditValidator"
```

### Osmosis Modules
#### Gamm
```go
	MsgSwapExactAmountIn       = "/osmosis.gamm.v1beta1.MsgSwapExactAmountIn"
	MsgSwapExactAmountOut      = "/osmosis.gamm.v1beta1.MsgSwapExactAmountOut"
	MsgJoinSwapExternAmountIn  = "/osmosis.gamm.v1beta1.MsgJoinSwapExternAmountIn"
	MsgJoinSwapShareAmountOut  = "/osmosis.gamm.v1beta1.MsgJoinSwapShareAmountOut"
	MsgJoinPool                = "/osmosis.gamm.v1beta1.MsgJoinPool"
	MsgExitSwapShareAmountIn   = "/osmosis.gamm.v1beta1.MsgExitSwapShareAmountIn"
	MsgExitSwapExternAmountOut = "/osmosis.gamm.v1beta1.MsgExitSwapExternAmountOut"
	MsgExitPool                = "/osmosis.gamm.v1beta1.MsgExitPool"
```

#### Incentives
```go
	MsgCreateGauge = "/osmosis.incentives.MsgCreateGauge"
	MsgAddToGauge  = "/osmosis.incentives.MsgAddToGauge"

```

#### Lockup
```go
	MsgBeginUnlocking    = "/osmosis.lockup.MsgBeginUnlocking"
	MsgLockTokens        = "/osmosis.lockup.MsgLockTokens"
	MsgBeginUnlockingAll = "/osmosis.lockup.MsgBeginUnlockingAll"
```

#### Superfluid
```go
	MsgSuperfluidDelegate        = "/osmosis.superfluid.MsgSuperfluidDelegate"
	MsgSuperfluidUndelegate      = "/osmosis.superfluid.MsgSuperfluidUndelegate"
	MsgSuperfluidUnbondLock      = "/osmosis.superfluid.MsgSuperfluidUnbondLock"
	MsgLockAndSuperfluidDelegate = "/osmosis.superfluid.MsgLockAndSuperfluidDelegate"
	MsgUnPoolWhitelistedPool     = "/osmosis.superfluid.MsgUnPoolWhitelistedPool"
```

### Tendermint Modules
#### Liquidity
```go
	MsgCreatePool          = "/tendermint.liquidity.v1beta1.MsgCreatePool"
	MsgDepositWithinBatch  = "/tendermint.liquidity.v1beta1.MsgDepositWithinBatch"
	MsgWithdrawWithinBatch = "/tendermint.liquidity.v1beta1.MsgWithdrawWithinBatch"
	MsgSwapWithinBatch     = "/tendermint.liquidity.v1beta1.MsgSwapWithinBatch"
```
