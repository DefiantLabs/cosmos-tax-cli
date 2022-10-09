package csv

import (
	"fmt"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/csv/parsers"
	"github.com/DefiantLabs/cosmos-tax-cli/csv/parsers/accointing"
	"github.com/DefiantLabs/cosmos-tax-cli/db"

	"gorm.io/gorm"
)

//Theres got to be a better way to do this
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

func ParseForAddress(address string, pgSql *gorm.DB, parserKey string, config config.Config) ([]parsers.CsvRow, []string, error) {

	parser := GetParser(parserKey)
	parser.InitializeParsingGroups(config)

	//TODO: need to pass in chain and date range
	taxableTxs, err := db.GetTaxableTransactions(address, pgSql)

	if err != nil {
		fmt.Println(err)
		return nil, nil, err
	}

	err = parser.ProcessTaxableTx(address, taxableTxs)

	if err != nil {
		fmt.Println(err)
		return nil, nil, err
	}

	taxableEvents, err := db.GetTaxableEvents(address, pgSql)

	if err != nil {
		fmt.Println(err)
		return nil, nil, err
	}

	parser.ProcessTaxableEvent(address, taxableEvents)

	//Get rows once right at the end
	rows := parser.GetRows()

	return rows, parser.GetHeaders(), nil
}
