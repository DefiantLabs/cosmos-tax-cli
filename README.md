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
