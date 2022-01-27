# Cosmos Exporter

This application indexes a Cosmos chain to a standardized DB schema with the following goals in mind:

* A generalized DB schema that could potentially work with all Cosmos SDK Chains
* A focus on making it easy to correlate transactions with addresses and store relevant data

# Requirements

## PostgreSQL

The app requires a postgresql server with an established database and an owner user/role with password login.

## Go

The app is written and Go and you will need to build from source. This requires a system install of Go.