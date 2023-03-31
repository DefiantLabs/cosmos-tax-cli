package koinly

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli/assetlists"
	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/bank"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/distribution"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/gov"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/ibc"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/staking"
	"github.com/DefiantLabs/cosmos-tax-cli/csv/parsers"
	"github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/gamm"
)

var unsupportedCoins = []string{
	"gamm",
}

var coinReplacementMap = map[string]string{}

var symbolsToKoinlyIds = map[string]string{}

// Probably not the best place for this, but its only used in Koinly for now
// May want to consider adding the asset list URL as a config value and passing the AssetList value down to parsers if needed
func init() {
	assetList, err := assetlists.GetAssetList("https://raw.githubusercontent.com/DefiantLabs/assetlists/main/osmosis-1/osmosis-1.assetlist.json")
	if err != nil {
		fmt.Println(err)
		panic("Could not load Koinly Symbols to IDs map")
	}

	for _, asset := range assetList.Assets {
		// Required format for koinly IDs is ID:<val>
		symbolsToKoinlyIds[asset.Symbol] = fmt.Sprintf("ID:%s", asset.KoinlyId)
	}
}

func (p *Parser) TimeLayout() string {
	return TimeLayout
}

func (p *Parser) ProcessTaxableTx(address string, taxableTxs []db.TaxableTransaction) error {
	// Build a map, so we know which TX go with which messages
	txMap := parsers.MakeTXMap(taxableTxs)

	// Pull messages out of txMap that must be grouped together
	parsers.SeparateParsingGroups(txMap, p.ParsingGroups)

	// Parse all the potentially taxable events (one transaction group at a time)
	for _, txGroup := range txMap {
		// All messages have been removed into a parsing group
		if len(txGroup) != 0 {
			// For the current transaction group, generate the rows for the CSV.
			// Usually (but not always) a transaction will only have a single row in the CSV.
			txRows, err := ParseTx(address, txGroup)
			if err != nil {
				return err
			}
			for _, v := range txRows {
				p.Rows = append(p.Rows, v.(Row))
			}
		}
	}

	// Parse all the TXs found in the Parsing Groups
	for _, txParsingGroup := range p.ParsingGroups {
		err := txParsingGroup.ParseGroup(ParseGroup)
		if err != nil {
			return err
		}
	}

	// Handle fees on all taxableTxs at once, we don't do this in the regular parser or in the parsing groups
	// This requires HandleFees to process the fees into unique mappings of tx -> fees (since we gather Taxable Messages in the taxableTxs)
	// If we move it into the ParseTx function or into the ParseGroup function, we may be able to reduce the logic in the HandleFees func
	feeRows, err := HandleFees(address, taxableTxs)
	if err != nil {
		return err
	}

	p.Rows = append(p.Rows, feeRows...)

	return nil
}

func (p *Parser) ProcessTaxableEvent(taxableEvents []db.TaxableEvent) error {
	// Parse all the potentially taxable events
	for _, event := range taxableEvents {
		// generate the rows for the CSV.
		p.Rows = append(p.Rows, ParseEvent(event)...)
	}

	return nil
}

func (p *Parser) InitializeParsingGroups() {
	p.ParsingGroups = append(p.ParsingGroups, parsers.GetOsmosisTxParsingGroups()...)
}

func (p *Parser) GetRows(address string, startDate, endDate *time.Time) []parsers.CsvRow {
	// Combine all normal rows and parser group rows into 1
	koinlyRows := p.Rows // contains TX rows and fees as well as taxable events
	for _, v := range p.ParsingGroups {
		for _, row := range v.GetRowsForParsingGroup() {
			koinlyRows = append(koinlyRows, row.(Row))
		}
	}

	for i, row := range koinlyRows {

		if row.FeeCurrency != "" {
			if _, ok := symbolsToKoinlyIds[row.FeeCurrency]; ok {
				koinlyRows[i].FeeCurrency = symbolsToKoinlyIds[row.FeeCurrency]
			}
		}
		if row.ReceivedCurrency != "" {
			if _, ok := symbolsToKoinlyIds[row.ReceivedCurrency]; ok {
				koinlyRows[i].ReceivedCurrency = symbolsToKoinlyIds[row.ReceivedCurrency]
			}
		}
		if row.SentCurrency != "" {
			if _, ok := symbolsToKoinlyIds[row.SentCurrency]; ok {
				koinlyRows[i].SentCurrency = symbolsToKoinlyIds[row.SentCurrency]
			}
		}
	}

	// Sort by date
	sort.Slice(koinlyRows, func(i int, j int) bool {
		leftDate, err := time.Parse(TimeLayout, koinlyRows[i].Date)
		if err != nil {
			config.Log.Error("Error sorting left date.", err)
			return false
		}
		rightDate, err := time.Parse(TimeLayout, koinlyRows[j].Date)
		if err != nil {
			config.Log.Error("Error sorting right date.", err)
			return false
		}
		return leftDate.Before(rightDate)
	})

	// Now that we are sorted, if we have a start date, drop everything from before it, if end date is set, drop everything after it
	var firstToKeep *int
	var lastToKeep *int
	for i := range koinlyRows {
		if startDate != nil && firstToKeep == nil {
			rowDate, err := time.Parse(TimeLayout, koinlyRows[i].Date)
			if err != nil {
				config.Log.Fatal("Error parsing row date.", err)
			}
			if rowDate.Before(*startDate) {
				continue
			} else {
				startIdx := i
				firstToKeep = &startIdx
			}
		} else if endDate != nil && lastToKeep == nil {
			rowDate, err := time.Parse(TimeLayout, koinlyRows[i].Date)
			if err != nil {
				config.Log.Fatal("Error parsing row date.", err)
			}
			if rowDate.Before(*endDate) {
				continue
			} else if i > 0 {
				endIdx := i - 1
				lastToKeep = &endIdx
				break
			}
		}
	}
	if firstToKeep != nil && lastToKeep != nil { // nolint:gocritic
		koinlyRows = koinlyRows[*firstToKeep:*lastToKeep]
	} else if firstToKeep != nil {
		koinlyRows = koinlyRows[*firstToKeep:]
	} else if lastToKeep != nil {
		koinlyRows = koinlyRows[:*lastToKeep]
	}

	mapUnsupportedCoints(koinlyRows)

	// Copy AccointingRows into csvRows for return val
	csvRows := make([]parsers.CsvRow, len(koinlyRows))
	for i, v := range koinlyRows {
		if _, isUnsuppored := coinReplacementMap[v.ReceivedCurrency]; isUnsuppored {
			v.ReceivedCurrency = coinReplacementMap[v.ReceivedCurrency]
		}
		if _, isUnsuppored := coinReplacementMap[v.SentCurrency]; isUnsuppored {
			v.SentCurrency = coinReplacementMap[v.SentCurrency]
		}

		csvRows[i] = v
	}
	return csvRows
}

// mapUnsupportedCoints will create a map of unsupported coins to be replaced with NULL
func mapUnsupportedCoints(rows []Row) {
	for _, row := range rows {
		for _, unsupportedCoin := range unsupportedCoins {
			if strings.Contains(row.ReceivedCurrency, unsupportedCoin) {
				if _, ok := coinReplacementMap[row.ReceivedCurrency]; !ok {
					coinReplacementMap[row.ReceivedCurrency] = fmt.Sprintf("NULL%d", len(coinReplacementMap)+1)
				}
			}
			if strings.Contains(row.SentCurrency, unsupportedCoin) {
				if _, ok := coinReplacementMap[row.SentCurrency]; !ok {
					coinReplacementMap[row.SentCurrency] = fmt.Sprintf("NULL%d", len(coinReplacementMap)+1)
				}
			}
		}
	}
}

func (p Parser) GetHeaders() []string {
	return []string{"Date", "Sent Amount", "Sent Currency", "Received Amount", "Received Currency", "Fee Amount", "Fee Currency",
		"Net Worth Amount", "Net Worth Currency", "Label", "Description", "TxHash"}
}

// HandleFees:
// If the transaction lists the same amount of fees as there are rows in the CSV,
// then we spread the fees out one per row. Otherwise we add a line for the fees,
// where each fee has a separate line.
func HandleFees(address string, events []db.TaxableTransaction) (rows []Row, err error) {
	// No events -- This address didn't pay any fees
	if len(events) == 0 {
		return rows, nil
	}

	// We need to gather all unique fees, but we are receiving Messages not Txes
	// Make a map from TX hash to fees array to keep unique
	txToFeesMap := make(map[uint][]db.Fee)
	txIdsToTx := make(map[uint]db.Tx)
	for _, event := range events {
		txID := event.Message.Tx.ID
		feeStore := event.Message.Tx.Fees
		txToFeesMap[txID] = feeStore
		txIdsToTx[txID] = event.Message.Tx
	}

	for id, txFees := range txToFeesMap {
		for _, fee := range txFees {
			if fee.PayerAddress.Address == address {
				newRow := Row{}
				err = newRow.ParseFee(txIdsToTx[id], fee)
				if err != nil {
					return nil, err
				}
				rows = append(rows, newRow)
			}
		}
	}

	return rows, nil
}

// ParseEvent: Parse the potentially taxable event
func ParseEvent(event db.TaxableEvent) (rows []Row) {
	if event.Source == db.OsmosisRewardDistribution {
		row, err := ParseOsmosisReward(event)
		if err != nil {
			config.Log.Fatal("error parsing row. Should be impossible to reach this condition, ideally (once all bugs worked out)", err)
		}
		rows = append(rows, row)
	}

	return rows
}

// ParseTx: Parse the potentially taxable TX and Messages
// This function is used for parsing a single TX that will not need to relate to any others
// Use TX Parsing Groups to parse txes as a group
func ParseTx(address string, events []db.TaxableTransaction) (rows []parsers.CsvRow, err error) {
	for _, event := range events {
		switch event.Message.MessageType.MessageType {
		case bank.MsgSendV0:
			rows = append(rows, ParseMsgSend(address, event))
		case bank.MsgSend:
			rows = append(rows, ParseMsgSend(address, event))
		case bank.MsgMultiSendV0:
			rows = append(rows, ParseMsgMultiSend(address, event))
		case bank.MsgMultiSend:
			rows = append(rows, ParseMsgMultiSend(address, event))
		case distribution.MsgFundCommunityPool:
			rows = append(rows, ParseMsgFundCommunityPool(address, event))
		case distribution.MsgWithdrawValidatorCommission:
			rows = append(rows, ParseMsgWithdrawValidatorCommission(address, event))
		case distribution.MsgWithdrawRewards:
			rows = append(rows, ParseMsgWithdrawDelegatorReward(address, event))
		case distribution.MsgWithdrawDelegatorReward:
			rows = append(rows, ParseMsgWithdrawDelegatorReward(address, event))
		case staking.MsgDelegate:
			rows = append(rows, ParseMsgWithdrawDelegatorReward(address, event))
		case staking.MsgUndelegate:
			rows = append(rows, ParseMsgWithdrawDelegatorReward(address, event))
		case staking.MsgBeginRedelegate:
			rows = append(rows, ParseMsgWithdrawDelegatorReward(address, event))
		case gamm.MsgSwapExactAmountIn:
			rows = append(rows, ParseMsgSwapExactAmountIn(event))
		case gamm.MsgSwapExactAmountOut:
			rows = append(rows, ParseMsgSwapExactAmountOut(event))
		case ibc.MsgTransfer:
			rows = append(rows, ParseMsgTransfer(address, event))
		case gov.MsgSubmitProposal:
			rows = append(rows, ParseMsgSubmitProposal(address, event))
		case gov.MsgDeposit:
			rows = append(rows, ParseMsgDeposit(address, event))
		default:
			return nil, fmt.Errorf("no parser for message type '%v'", event.Message.MessageType.MessageType)
		}
	}
	return rows, nil
}

// ParseMsgValidatorWithdraw:
// This transaction is always a withdrawal.
func ParseMsgWithdrawValidatorCommission(address string, event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgWithdrawValidatorCommission.", err)
	}
	row.Label = Unstake
	return *row
}

// ParseMsgValidatorWithdraw:
// This transaction is always a withdrawal.
func ParseMsgWithdrawDelegatorReward(address string, event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgWithdrawDelegatorReward.", err)
	}
	row.Label = Unstake
	return *row
}

// ParseMsgSend:
// If the address we searched is the receiver, then this transaction is a deposit.
// If the address we searched is the sender, then this transaction is a withdrawal.
func ParseMsgSend(address string, event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgSend.", err)
	}
	return *row
}

func ParseMsgMultiSend(address string, event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgMultiSend.", err)
	}
	return *row
}

func ParseMsgFundCommunityPool(address string, event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgFundCommunityPool.", err)
	}
	return *row
}

func ParseMsgSwapExactAmountIn(event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseSwap(event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgSwapExactAmountIn.", err)
	}
	return *row
}

func ParseMsgSwapExactAmountOut(event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseSwap(event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgSwapExactAmountOut.", err)
	}
	return *row
}

func ParseMsgTransfer(address string, event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgTransfer.", err)
	}
	return *row
}

func ParseMsgSubmitProposal(address string, event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgSubmitProposal.", err)
	}
	return *row
}

func ParseMsgDeposit(address string, event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgDeposit.", err)
	}
	return *row
}

func ParseOsmosisReward(event db.TaxableEvent) (Row, error) {
	row := &Row{}
	err := row.EventParseBasic(event)
	if err != nil {
		config.Log.Fatal("Error with ParseOsmosisReward.", err)
	}
	return *row, err
}
