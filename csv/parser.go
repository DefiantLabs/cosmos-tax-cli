package csv

import (
	"go.uber.org/zap"

	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
	"github.com/DefiantLabs/cosmos-tax-cli-private/csv/parsers"
	"github.com/DefiantLabs/cosmos-tax-cli-private/csv/parsers/accointing"
	"github.com/DefiantLabs/cosmos-tax-cli-private/db"

	"gorm.io/gorm"
)

// Register new parsers by adding them to this list
var supportedParsers = []string{accointing.ParserKey}

func init() {
	parsers.RegisterParsers(supportedParsers)
}

func GetParser(parserKey string) parsers.Parser {
	if parserKey == accointing.ParserKey {
		parser := accointing.Parser{}
		return &parser
	}
	return nil
}

func ParseForAddress(address string, pgSQL *gorm.DB, parserKey string, cfg config.Config) ([]parsers.CsvRow, []string, error) {
	parser := GetParser(parserKey)
	parser.InitializeParsingGroups(cfg)

	//TODO: need to pass in chain and date range
	taxableTxs, err := db.GetTaxableTransactions(address, pgSQL)
	if err != nil {
		config.Log.Error("Error getting taxable transaction.", zap.Error(err))
		return nil, nil, err
	}

	err = parser.ProcessTaxableTx(address, taxableTxs)
	if err != nil {
		config.Log.Error("Error processing taxable transaction.", zap.Error(err))
		return nil, nil, err
	}

	taxableEvents, err := db.GetTaxableEvents(address, pgSQL)
	if err != nil {
		config.Log.Error("Error getting taxable events.", zap.Error(err))
		return nil, nil, err
	}

	err = parser.ProcessTaxableEvent(address, taxableEvents)
	if err != nil {
		config.Log.Error("Error processing taxable events.", zap.Error(err))
		return nil, nil, err
	}

	//Get rows once right at the end
	rows := parser.GetRows()

	return rows, parser.GetHeaders(), nil
}
