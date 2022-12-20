package csv

import (
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
	"github.com/DefiantLabs/cosmos-tax-cli-private/csv/parsers"
	"github.com/DefiantLabs/cosmos-tax-cli-private/csv/parsers/accointing"
	"github.com/DefiantLabs/cosmos-tax-cli-private/csv/parsers/koinly"
	"github.com/DefiantLabs/cosmos-tax-cli-private/db"

	"gorm.io/gorm"
)

// Register new parsers by adding them to this list
var supportedParsers = []string{accointing.ParserKey, koinly.ParserKey}

func init() {
	parsers.RegisterParsers(supportedParsers)
}

func GetParser(parserKey string) parsers.Parser {
	switch parserKey {
	case accointing.ParserKey:
		parser := accointing.Parser{}
		return &parser
	case koinly.ParserKey:
		parser := koinly.Parser{}
		return &parser
	}
	return nil
}

func ParseForAddress(address string, startDate, endDate *time.Time, pgSQL *gorm.DB, parserKey string, cfg config.Config) ([]parsers.CsvRow, []string, error) {
	parser := GetParser(parserKey)
	parser.InitializeParsingGroups(cfg)

	// TODO: need to pass in chain and date range
	taxableTxs, err := db.GetTaxableTransactions(address, pgSQL)
	if err != nil {
		config.Log.Error("Error getting taxable transaction.", err)
		return nil, nil, err
	}

	err = parser.ProcessTaxableTx(address, taxableTxs)
	if err != nil {
		config.Log.Error("Error processing taxable transaction.", err)
		return nil, nil, err
	}

	taxableEvents, err := db.GetTaxableEvents(address, pgSQL)
	if err != nil {
		config.Log.Error("Error getting taxable events.", err)
		return nil, nil, err
	}

	err = parser.ProcessTaxableEvent(taxableEvents)
	if err != nil {
		config.Log.Error("Error processing taxable events.", err)
		return nil, nil, err
	}

	// Get rows once right at the end, also filter them by date
	rows := parser.GetRows(address, startDate, endDate)

	return rows, parser.GetHeaders(), nil
}
