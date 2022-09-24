package parsers

import (
	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/db"
)

type Parser interface {
	InitializeParsingGroups(config config.Config)
	ProcessTaxableTx(address string, taxableTxs []db.TaxableTransaction) error
	ProcessTaxableEvent(address string, taxableEvents []db.TaxableEvent) error
	GetHeaders() []string
	GetRows() []CsvRow
}

type ParsingGroup interface {
	BelongsToGroup(db.TaxableTransaction) bool
	String() string
	AddTxToGroup(db.TaxableTransaction)
	GetGroupedTxes() map[uint][]db.TaxableTransaction
	ParseGroup() error
	GetRowsForParsingGroup() []CsvRow
}

type CsvRow interface {
	GetRowForCsv() []string
}
