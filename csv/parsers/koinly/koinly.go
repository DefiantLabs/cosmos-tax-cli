package koinly

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/bank"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/distribution"
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
var coinScalingMap = map[string]int{}
var coinMaxDigitsMap = map[string]int{}

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
		err := txParsingGroup.ParseGroup()
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

func (p *Parser) InitializeParsingGroups(config config.Config) {
}

func (p *Parser) GetRows() []parsers.CsvRow {
	// Combine all normal rows and parser group rows into 1
	koinlyRows := p.Rows // contains TX rows and fees as well as taxable events
	for _, v := range p.ParsingGroups {
		for _, row := range v.GetRowsForParsingGroup() {
			koinlyRows = append(koinlyRows, row.(Row))
		}
	}

	// Sort by date
	sort.Slice(koinlyRows, func(i int, j int) bool {
		leftDate, err := time.Parse(TimeLayout, koinlyRows[i].Date)
		if err != nil {
			config.Log.Error("Error sorting left date.", zap.Error(err))
			return false
		}
		rightDate, err := time.Parse(TimeLayout, koinlyRows[j].Date)
		if err != nil {
			config.Log.Error("Error sorting right date.", zap.Error(err))
			return false
		}
		return leftDate.Before(rightDate)
	})

	// Before generating a CSV, we need to do a pass over all the cells to calculate proper scaling
	calculateScaling(koinlyRows)

	// Copy koinlyRows into csvRows for return val
	csvRows := make([]parsers.CsvRow, len(koinlyRows))
	for i, v := range koinlyRows {
		// Scale amounts as needed for koinly's limit of 10^15
		var shiftedCoins []string
		if _, isShifted := coinReplacementMap[v.ReceivedCurrency]; isShifted {
			shiftedCoins = append(shiftedCoins, v.ReceivedCurrency)
			updatedAmount, updatedDenom := adjustUnitsAndDenoms(v.ReceivedAmount, v.ReceivedCurrency)
			v.ReceivedAmount = updatedAmount
			v.ReceivedCurrency = updatedDenom
		}
		if _, isShifted := coinReplacementMap[v.SentCurrency]; isShifted {
			shiftedCoins = append(shiftedCoins, v.SentCurrency)
			updatedAmount, updatedDenom := adjustUnitsAndDenoms(v.SentAmount, v.SentCurrency)
			v.SentAmount = updatedAmount
			v.SentCurrency = updatedDenom
		}
		switch len(shiftedCoins) {
		case 1:
			coin := shiftedCoins[0]
			v.Description = fmt.Sprintf("%v [1 %v = %.0f %v]", v.Description,
				coinReplacementMap[coin], math.Pow(10, float64(coinScalingMap[coin])), coin)
		case 2:
			coin1 := shiftedCoins[0]
			coin2 := shiftedCoins[1]
			v.Description = fmt.Sprintf("%v [1 %v = %.0f %v and 1 %v = %.0f %v]", v.Description,
				coinReplacementMap[coin1], math.Pow(10, float64(coinScalingMap[coin1])), coin1,
				coinReplacementMap[coin2], math.Pow(10, float64(coinScalingMap[coin2])), coin2)
		}

		csvRows[i] = v
	}

	return csvRows
}

// calculateScaling will take a pass through all rows and determine how/which coins to scale
// if either sent or received coin is in the map of unsupported coins, add it to replacement map
// for each replaced coin, track max number of digits
func calculateScaling(rows []Row) {
	for _, row := range rows {
		for _, unsupportedCoin := range unsupportedCoins {
			if strings.Contains(row.ReceivedCurrency, unsupportedCoin) {
				if _, ok := coinReplacementMap[row.ReceivedCurrency]; !ok {
					coinReplacementMap[row.ReceivedCurrency] = fmt.Sprintf("NULL%d", len(coinReplacementMap)+1)
					coinMaxDigitsMap[row.ReceivedCurrency] = len(row.ReceivedAmount)
				} else if coinMaxDigitsMap[row.ReceivedCurrency] < len(row.ReceivedAmount) {
					coinMaxDigitsMap[row.ReceivedCurrency] = len(row.ReceivedAmount)
				}
			}
			if strings.Contains(row.SentCurrency, unsupportedCoin) {
				if _, ok := coinReplacementMap[row.SentCurrency]; !ok {
					coinReplacementMap[row.SentCurrency] = fmt.Sprintf("NULL%d", len(coinReplacementMap)+1)
					coinMaxDigitsMap[row.ReceivedCurrency] = len(row.SentAmount)
				} else if coinMaxDigitsMap[row.ReceivedCurrency] < len(row.SentAmount) {
					coinMaxDigitsMap[row.ReceivedCurrency] = len(row.SentAmount)
				}
			}
		}
	}
	// now that all coins have been mapped, determine scaling
	for coin, maxDigits := range coinMaxDigitsMap {
		shift := 0
		for maxDigits-shift > 15 {
			shift += 3
		}
		coinScalingMap[coin] = shift
	}
}

// adjustUnitsAndDenoms will adjust amounts and denominations in the following ways
// - Amounts cannot be greater than 10^15
// - Some coins are not supported and need to be replaced by "NULL{N}" where N is the index of each unique currency
// - TODO: Make sure this is up to date
func adjustUnitsAndDenoms(amount, unit string) (updatedAmount string, updatedUnit string) {
	idx := len(amount) - coinScalingMap[unit]
	updatedAmount = amount[:idx] + "." + amount[idx:]
	updatedUnit = coinReplacementMap[unit]
	return
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
			// TODO: handle error parsing row. Should be impossible to reach this condition, ideally (once all bugs worked out)
			config.Log.Fatal("error parsing row. Should be impossible to reach this condition, ideally (once all bugs worked out)", zap.Error(err))
		}
		rows = append(rows, row)
	}

	// rows = HandleFees(address, events, rows) TODO we have no fee handler for taxable EVENTS right now
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
		config.Log.Fatal("Error with ParseMsgWithdrawValidatorCommission.", zap.Error(err))
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
		config.Log.Fatal("Error with ParseMsgWithdrawDelegatorReward.", zap.Error(err))
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
		config.Log.Fatal("Error with ParseMsgSend.", zap.Error(err))
	}
	return *row
}

func ParseMsgMultiSend(address string, event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgMultiSend.", zap.Error(err))
	}
	return *row
}

func ParseMsgFundCommunityPool(address string, event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgFundCommunityPool.", zap.Error(err))
	}
	return *row
}

func ParseMsgSwapExactAmountIn(event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseSwap(event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgSwapExactAmountIn.", zap.Error(err))
	}
	return *row
}

func ParseMsgSwapExactAmountOut(event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseSwap(event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgSwapExactAmountOut.", zap.Error(err))
	}
	return *row
}

func ParseMsgTransfer(address string, event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgTransfer.", zap.Error(err))
	}
	return *row
}

func ParseOsmosisReward(event db.TaxableEvent) (Row, error) {
	row := &Row{}
	err := row.EventParseBasic(event)
	if err != nil {
		config.Log.Fatal("Error with ParseOsmosisReward.", zap.Error(err))
	}
	return *row, err
}
