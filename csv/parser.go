package csv

import (
	"go.uber.org/zap"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/csv/parsers"
	"github.com/DefiantLabs/cosmos-tax-cli/csv/parsers/accointing"
	"github.com/DefiantLabs/cosmos-tax-cli/db"

	"gorm.io/gorm"
)

// Theres got to be a better way to do this
func init() {
	parsers.RegisterParser(accointing.ParserKey)
}

func GetParser(parserKey string) parsers.Parser {
	if parserKey == "accointing" {
		parser := accointing.AccointingParser{}
		return &parser
	}
	return nil
}

func ParseForAddress(address string, pgSql *gorm.DB, parserKey string, cfg config.Config) ([]parsers.CsvRow, []string, error) {
	parser := GetParser(parserKey)
	parser.InitializeParsingGroups(cfg)

	//TODO: need to pass in chain and date range
	taxableTxs, err := db.GetTaxableTransactions(address, pgSql)
	if err != nil {
		config.Log.Error("Error getting taxable transaction.", zap.Error(err))
		return nil, nil, err
	}

	err = parser.ProcessTaxableTx(address, taxableTxs)
	if err != nil {
		config.Log.Error("Error processing taxable transaction.", zap.Error(err))
		return nil, nil, err
	}

	taxableEvents, err := db.GetTaxableEvents(address, pgSql)
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
