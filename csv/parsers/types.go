package parsers

import (
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
	"github.com/DefiantLabs/cosmos-tax-cli-private/db"
)

type Parser interface {
	InitializeParsingGroups(config config.Config)
	ProcessTaxableTx(address string, taxableTxs []db.TaxableTransaction) error
	ProcessTaxableEvent(taxableEvents []db.TaxableEvent) error
	GetHeaders() []string
	GetRows(address string, startDate, endDate *time.Time) []CsvRow
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
	GetDate() string
}
