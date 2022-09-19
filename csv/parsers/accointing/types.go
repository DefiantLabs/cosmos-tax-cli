package accointing

import "github.com/DefiantLabs/cosmos-tax-cli/csv/parsers"

var ParserKey string = "accointing"

type AccointingParser struct {
	Rows          []AccointingRow
	ParsingGroups []parsers.ParsingGroup
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

type AccointingTransaction int

const (
	Deposit AccointingTransaction = iota
	Withdraw
	Order
)

func (at AccointingTransaction) String() string {
	return [...]string{"deposit", "withdraw", "order"}[at]
}

type AccointingClassification int

const (
	None AccointingClassification = iota
	Staked
	Airdrop
	Payment
	Fee
	LiquidityPool
	RemoveFunds //Used for GAMM module exits, is this correct?
)

func (ac AccointingClassification) String() string {
	//Note that "None" returns empty string since we're using this for CSV parsing.
	//Accointing considers 'Classification' an optional field, so empty is a valid value.
	return [...]string{"", "staked", "airdrop", "payment", "fee", "liquidity_pool", "remove_funds"}[ac]
}
