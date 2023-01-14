package csv

import (
	"sort"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
	"github.com/DefiantLabs/cosmos-tax-cli-private/csv/parsers"
	"github.com/DefiantLabs/cosmos-tax-cli-private/csv/parsers/accointing"
	"github.com/DefiantLabs/cosmos-tax-cli-private/csv/parsers/cointracker"
	"github.com/DefiantLabs/cosmos-tax-cli-private/csv/parsers/koinly"
	"github.com/DefiantLabs/cosmos-tax-cli-private/csv/parsers/taxbit"
	"github.com/DefiantLabs/cosmos-tax-cli-private/db"

	"gorm.io/gorm"
)

// Register new parsers by adding them to this list
var supportedParsers = []string{accointing.ParserKey, koinly.ParserKey, cointracker.ParserKey, taxbit.ParserKey}

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
	case cointracker.ParserKey:
		parser := cointracker.Parser{}
		return &parser
	case taxbit.ParserKey:
		parser := taxbit.Parser{}
		return &parser
	}
	return nil
}

func ParseForAddress(addresses []string, startDate, endDate *time.Time, pgSQL *gorm.DB, parserKey string, cfg config.Config) ([]parsers.CsvRow, []string, error) {
	parser := GetParser(parserKey)
	parser.InitializeParsingGroups()

	// Get data for each address
	var headers []string
	var csvRows []parsers.CsvRow
	for _, address := range addresses {
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

		csvRows = append(csvRows, rows...)
		headers = parser.GetHeaders()
	}
	// re-sort rows if needed
	if len(addresses) > 1 {
		SortRows(csvRows, parser.TimeLayout())
	}
	return csvRows, headers, nil
}

func SortRows(csvRows []parsers.CsvRow, timeLayout string) {
	// Sort by date
	sort.Slice(csvRows, func(i int, j int) bool {
		leftDate, err := time.Parse(timeLayout, csvRows[i].GetDate())
		if err != nil {
			config.Log.Error("Error sorting left date.", err)
			return false
		}
		rightDate, err := time.Parse(timeLayout, csvRows[j].GetDate())
		if err != nil {
			config.Log.Error("Error sorting right date.", err)
			return false
		}
		return leftDate.Before(rightDate)
	})
}
