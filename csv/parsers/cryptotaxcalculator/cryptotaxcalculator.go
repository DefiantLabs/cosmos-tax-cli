package cryptotaxcalculator

import (
	"fmt"
	"sort"
	"time"

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

func (p *Parser) GetRows(address string, startDate, endDate *time.Time) []parsers.CsvRow {
	// Combine all normal rows and parser group rows into 1
	cryptoRows := p.Rows // contains TX rows and fees as well as taxable events
	for _, v := range p.ParsingGroups {
		for _, row := range v.GetRowsForParsingGroup() {
			cryptoRows = append(cryptoRows, row.(Row))
		}
	}

	// Sort by date
	sort.Slice(cryptoRows, func(i int, j int) bool {
		return cryptoRows[i].Date.Before(cryptoRows[j].Date)
	})

	// Now that we are sorted, if we have a start date, drop everything from before it, if end date is set, drop everything after it
	var firstToKeep *int
	var lastToKeep *int
	for i := range cryptoRows {
		if startDate != nil && firstToKeep == nil {
			if cryptoRows[i].Date.Before(*startDate) {
				continue
			} else {
				startIdx := i
				firstToKeep = &startIdx
			}
		} else if endDate != nil && lastToKeep == nil {
			if cryptoRows[i].Date.Before(*endDate) {
				continue
			} else if i > 0 {
				endIdx := i - 1
				lastToKeep = &endIdx
				break
			}
		}
	}
	if firstToKeep != nil && lastToKeep != nil { // nolint:gocritic
		cryptoRows = cryptoRows[*firstToKeep:*lastToKeep]
	} else if firstToKeep != nil {
		cryptoRows = cryptoRows[*firstToKeep:]
	} else if lastToKeep != nil {
		cryptoRows = cryptoRows[:*lastToKeep]
	}

	// Copy AccointingRows into csvRows for return val
	csvRows := make([]parsers.CsvRow, len(cryptoRows))
	for i, v := range cryptoRows {
		csvRows[i] = v
	}

	return csvRows
}

func (p Parser) GetHeaders() []string {
	return []string{
		"Timestamp (UTC)",
		"Type",
		"Base Currency",
		"Base Amount",
		"Quote Currency (Optional)",
		"Quote Amount (Optional)",
		"Fee Currency (Optional)",
		"Fee Amount (Optional)",
		"From (Optional)",
		"To (Optional)",
		"Blockchain (Optional)",
		"ID (Optional",
		"Description (Optional)",
		"Reference Price Per Unit (Optional)",
		"Reference Price Currency (Optional)",
	}
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
			rows = append(rows, ParseMsgSwapExactAmountIn(address, event))
		case gamm.MsgSwapExactAmountOut:
			rows = append(rows, ParseMsgSwapExactAmountOut(address, event))
		case gov.MsgSubmitProposal:
			rows = append(rows, ParseMsgSubmitProposal(address, event))
		case gov.MsgDeposit:
			rows = append(rows, ParseMsgDeposit(address, event))
		case ibc.MsgTransfer:
			rows = append(rows, ParseMsgTransfer(address, event))
		default:
			return nil, fmt.Errorf("no parser for message type '%v'", event.Message.MessageType.MessageType)
		}
	}
	return rows, nil
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

// ParseMsgValidatorWithdraw:
// This transaction is always a withdrawal.
func ParseMsgWithdrawValidatorCommission(address string, event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgWithdrawValidatorCommission.", err)
	}
	// row.Label = Unstake
	return *row
}

func ParseMsgWithdrawDelegatorReward(address string, event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgWithdrawDelegatorReward.", err)
	}
	// row.Label = Unstake
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

func ParseMsgTransfer(address string, event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgTransfer.", err)
	}
	return *row
}

func (p *Parser) InitializeParsingGroups() {
	p.ParsingGroups = append(p.ParsingGroups, parsers.GetOsmosisTxParsingGroups()...)
}

func ParseOsmosisReward(event db.TaxableEvent) (Row, error) {
	row := &Row{}
	err := row.EventParseBasic(event)
	if err != nil {
		config.Log.Fatal("Error with ParseOsmosisReward.", err)
	}
	return *row, err
}

func ParseMsgSwapExactAmountIn(address string, event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseSwap(event, address, Buy)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgSwapExactAmountIn.", err)
	}
	return *row
}

func ParseMsgSwapExactAmountOut(address string, event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseSwap(event, address, Sell)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgSwapExactAmountOut.", err)
	}
	return *row
}
