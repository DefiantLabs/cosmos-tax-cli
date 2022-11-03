# Cosmos Tax CLI

This application indexes a Cosmos chain to a standardized DB schema with the following goals in mind:

* A generalized DB schema that could potentially work with all Cosmos SDK Chains
* A focus on making it easy to correlate transactions with addresses and store relevant data

# Requirements

## PostgreSQL

The app requires a postgresql server with an established database and an owner user/role with password login.

## Go

The app is written and Go and you will need to build from source. This requires a system install of Go.

## Getting Started
Lens is a prerequisite to using this tool, so make sure Lens is set up properly. There is a [lens] section in our config.toml.example
and the vars in there are so configuring the Lens SDK will work properly. The code in this repo only queries RPC, it does not create TXs.
So the key you use does not need to be associated with a wallet that has funds in it. Therefore, you can create a random key using lens.
I suggest installing lens (git clone git@github.com:DefiantLabs/lens.git). Run a make build which will put the 'lens' binary in the build folder.
If for some reason it fails, you can use git clone https://github.com/strangelove-ventures/lens.git and do the same thing.

Run lens and look at the help sections, it will tell you what your home directory is. Generally this is ~/.lens. All keys and config files
used by lens will go there. Lens uses a file in that directory called config.yaml to understand how to connect to various blockchains.
I found it easier to manually edit this config file than use the lens command to do it. Here's an example of a valid chain (under 'chains') in config.yaml:

kujira:
    key: kujiman
    chain-id: kaiyo-1
    rpc-addr: https://rpc.kujira.ccvalidators.com:443
    grpc-addr: https://rpc.kujira.ccvalidators.com:443
    account-prefix: kuji
    keyring-backend: test
    gas-adjustment: 1.2
    gas-prices: 0.01ukuji
    key-directory: /home/kyle/.lens/keys/kaiyo-1
    debug: false
    timeout: 20s
    output-format: json
    sign-mode: direct

To get the key kujiman to exist, you'd need to run lens keys add kujiman --chain kujira. Lens will create a new mnemonic for kujiman and store the key in the .lens
directory. You can also specify --keyring-backend test which will ensure the key is not password protected (which makes queries easier). Otherwise password prompts
can interrupt your programs. (Note: this only matters for TXs, not for queries. Keys are not needed for queries since queries are free and do not modify data).

At this point you should be ready to run the indexer. The indexer is what adds records to the Postgres DB and is what runs when you run the main program.
There is also a CSV generator, but right now you can only access it by calling the appropriate functions - there are test cases that do this. Eventually,
a frontend and CLI will be available to run the CSV generator.