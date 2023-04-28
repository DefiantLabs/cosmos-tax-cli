package cosmoshub

import tendermintHandlers "github.com/DefiantLabs/cosmos-tax-cli/tendermint"

// Extend these using an init func to setup CosmosHub end blocker handlers if we want more functionality
var EndBlockerEventTypeHandlers = tendermintHandlers.EndBlockerEventTypeHandlers
