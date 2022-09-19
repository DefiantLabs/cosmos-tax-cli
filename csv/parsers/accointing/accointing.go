package accointing

import (
	"github.com/DefiantLabs/cosmos-tax-cli/db"
)

var ParserKey string = "accointing"

type AccointingParser struct {
	Rows []AccointingRow
}

func (parser *AccointingParser) GetRowsFromTaxableTx(taxableTx []db.TaxableTransaction) [][]string {
	var rows [][]string
	for _, v := range parser.Rows {
		newRow := v.GetRowForCsv()
		rows = append(rows, newRow)
	}

	return rows
}

func (parser *AccointingParser) InitializeParsingGroups(chainId string) {

}

type AccointingRow struct {
	Date            string
	InBuyAmount     string
	InBuyAsset      string
	OutSellAmount   string
	OutSellAsset    string
	FeeAmount       string
	FeeAsset        string
	Classification  AccointingClassification
	TransactionType AccointingTransaction
	OperationId     string
	Comments        string
}

func (row AccointingRow) GetRowForCsv() []string {
	return nil
}
