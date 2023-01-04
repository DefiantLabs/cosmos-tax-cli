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
in a database to allow multiple users to query for their data. This section will cover the end to end process of
indexing and querying data.

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
in-depth [indexer](#indexer) section below.

### Querying
Once the chain has been indexed, data can be queried using the `query` command. As with indexing, a config file is provided
to configure the application. In addition, the addresses you wish to query can be provided as a comma separated list:

`go run main.go query --address "address1,address2" --config {{PATH_TO_CONFIG}}`

For more information about the query function and its optional arguments, please refer to the more in-depth
[query](#query) section below.

## Indexer
// TODO, add help text and a breakdown of all the config fields/flags.

## Query
// TODO, add help text and a breakdown of all the config fields/flags.
