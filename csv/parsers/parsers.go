package parsers

import "github.com/DefiantLabs/cosmos-tax-cli/db"

type Parser interface {
	GetRowsFromTaxableTx([]db.TaxableTransaction) [][]string
	InitializeParsingGroups(chainId string)
}

var Parsers map[string]Parser

//Check in your parsers here
func init() {
	Parsers = make(map[string]Parser)
}

func AddParserToParsers(key string, val Parser) {
	Parsers[key] = val
}

func GetParserKeys() []string {
	var parserKeys []string

	for i, _ := range Parsers {
		parserKeys = append(parserKeys, i)
	}

	return parserKeys
}

type TxParsingGroup interface {
	BelongsToGroup(db.TaxableTransaction) bool
	String() string
	AddTxToGroup(db.TaxableTransaction)
	GetGroupedTxes() map[uint][]db.TaxableTransaction
	GetRowsForParsingGroup(string) [][]string
}
