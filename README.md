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

## Quick Start
Use our docker-compose file to see a quick example of how to run the indexer, db, web client, and ui.
Edit start-block and end-block to be the osmosis block where you did a swap, or earned LP rewards.

Launch docker compose

```
docker compose up
```

Watch the output for the index.

Click the link to the web server



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
      --base.api string                 node api endpoint
      --base.block-timer int            print out how long it takes to process this many blocks (default 10000)
      --base.dry                        index the chain but don't insert data in the DB.
      --base.end-block int              block to stop indexing at (use -1 to index indefinitely (default -1)
      --base.exit-when-caught-up        mainly used for Osmosis rewards indexing (default true)
      --base.index-chain                enable chain indexing? (default true)
      --base.index-rewards              enable osmosis reward indexing? (default true)
      --base.prevent-reattempts         prevent reattempts of failed blocks.
      --base.reindex                    if true, this will re-attempt to index blocks we have already indexed (defaults to false)
      --base.rewards-end-block int      block to stop indexing rewards at (use -1 to index indefinitely
      --base.rewards-start-block int    block to start indexing rewards at
      --base.rpc-workers int            rpc workers (default 1)
      --base.start-block int            block to start indexing at (use -1 to resume from highest block indexed)
      --base.throttling float           throttle delay (default 0.5)
      --base.wait-for-chain             wait for chain to be in sync?
      --base.wait-for-chain-delay int   seconds to wait between each check for node to catch up to the chain (default 10)
      --config string                   config file (default is $HOME/.cosmos-tax-cli-private/config.yaml)
      --database.database string        database name
      --database.host string            database host
      --database.log-level string       database loglevel
      --database.password string        database password
      --database.port string            database port (default "5432")
      --database.user string            database user
      --lens.account-prefix string      lens account prefix
      --lens.chain-id string            lens chain ID
      --lens.chain-name string          lens chain name
      --lens.rpc string                 node rpc endpoint
      --log.level string                log level (default "info")
      --log.path string                 log path (default is $HOME/.cosmos-tax-cli-private/logs.txt
      --log.pretty                      pretty logs

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
#### level
This setting is used to determine which level of logs will be included. The available levels include
- `Debug`
- `Info` (default)
- `Warn`
- `Error`
- `Fatal`
- `Panic`

#### path
Logs will always be written to standard out, but if desired they can also be written to a file.
To do this, simply provide a path to a file.

#### pretty
We use a logging package called [ZeroLog](https://github.com/rs/zerolog). To take advantage of their "pretty" logging
you can set `pretty` to true.

### Database
The config options for the database are largely self-explanatory.

#### host
The address needed to connect to the DB.

#### port
The port needed to connect to the DB.

#### database
The name of the database to connect to.

#### user
The DB username

#### password
The password for the DB user.

#### log-level
This is a feature built into [gorm](https://gorm.io) to allow for logging of query information. This can be helpful for troubleshooting
performance issues.

Available log levels include:
- `silent` (default)
- `info`
- `warn`
- `error`

### Base
These are the core settings for the tool

#### api
Node API endpoint for querying chain information.

#### start-block
The block height to start indexing at.

#### end-block
The block height to stop indexing at. If set to '-1' indexing will keep running and keep pace with the chain.

#### reindex
If true, this will re-attempt to index blocks we have already indexed (defaults to false)

#### prevent-reattempts
If true, failed blocks are not reattempted. Defaults to false.

#### throttling
The minimum number of seconds per block. Higher number, will be slower. A value of 1 will result in approximately
1 block per second being indexed.

#### rpc-workers
The number of RPC workers. This should typically be a similar order of magnitude to the number of cpu cores available
to the indexer.

#### block-timer
The indexer will track how long it takes ot process this number of blocks.

#### wait-for-chain
When true, the indexer will not start until the defined RPC server's catching up status is false.

#### wait-for-chain-delay
Seconds to wait between each check for node to catch up to the chain.

#### index-chain
If false, the indexer won't actually index the chain. This may be desirable if your goal is only to index rewards.

#### exit-when-caught-up
When true, the indexer will exit when it catches up the the RPC server's latest_block_height

#### index-rewards
If true, the indexer will attempt to index osmosis rewards.

#### rewards-start-block
The block height to start indexing rewards. (will default to start block if not set)

#### rewards-end-block
The block height to stop indexing rewards. (will default to end block if not set)

#### dry
If true, the indexer will read the chain but won't actually write data to the database.

### Lens
This tool uses a [fork of Lens](https://github.com/DefiantLabs/lens) to read data from the blockchain. This is built
into the application and does not need to be installed separately.

#### rpc
The node RPC endpoint

#### account-prefix
Lens account prefix

#### chain-id
The ID of the chain being indexed

#### chain-name
The name of the chain being indexed


## Supported message types
When indexing the chain, we need to parse individual messages in order to determine their significance.
Some messages (transfers for example) have tax implication, whereas other (bonding/unbonding funds) do not. Furthermore, some messages types only have partial support or are still in progress. This allows the indexer to not error when processing this message type. While we are constantly adding to our list of supported messages, we do not currently support every possible message on every chain. If a message type is missing, or not being handled properly, please open an issue, or submit a PR.

To view the always up-to-date complete list of supported messages, please consult the code [here](https://github.com/DefiantLabs/cosmos-tax-cli-private/blob/main/core/tx.go)

Here is a list of what we support / handle currently.


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
	MsgDeposit        = "/cosmos.gov.v1beta1.MsgDeposit"
	MsgSubmitProposal = "/cosmos.gov.v1beta1.MsgSubmitProposal"
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

#### Vesting
```go
	MsgCreateVestingAccount = "/cosmos.vesting.v1beta1.MsgCreateVestingAccount"
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

### CosmWasm Modules
#### Wasm
```go
    MsgExecuteContract = "/cosmwasm.wasm.v1.MsgExecuteContract"
```
