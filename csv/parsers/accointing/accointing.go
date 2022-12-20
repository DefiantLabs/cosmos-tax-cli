package accointing

import (
	"fmt"
	"sort"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
	"github.com/DefiantLabs/cosmos-tax-cli-private/core"
	"github.com/DefiantLabs/cosmos-tax-cli-private/cosmos/modules/bank"
	"github.com/DefiantLabs/cosmos-tax-cli-private/cosmos/modules/distribution"
	"github.com/DefiantLabs/cosmos-tax-cli-private/cosmos/modules/gov"
	"github.com/DefiantLabs/cosmos-tax-cli-private/cosmos/modules/ibc"
	"github.com/DefiantLabs/cosmos-tax-cli-private/cosmos/modules/staking"
	"github.com/DefiantLabs/cosmos-tax-cli-private/csv/parsers"
	"github.com/DefiantLabs/cosmos-tax-cli-private/db"
	"github.com/DefiantLabs/cosmos-tax-cli-private/osmosis"
	"github.com/DefiantLabs/cosmos-tax-cli-private/osmosis/modules/gamm"
)

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
	if config.Lens.ChainID == osmosis.ChainID {
		p.ParsingGroups = append(p.ParsingGroups, GetOsmosisTxParsingGroups()...)
	}
}

func (p *Parser) GetRows(address string, startDate, endDate *time.Time) []parsers.CsvRow {
	// Combine all normal rows and parser group rows into 1
	accointingRows := p.Rows // contains TX rows and fees as well as taxable events
	for _, v := range p.ParsingGroups {
		for _, row := range v.GetRowsForParsingGroup() {
			accointingRows = append(accointingRows, row.(Row))
		}
	}

	// Sort by date
	sort.Slice(accointingRows, func(i int, j int) bool {
		leftDate, err := time.Parse(timeLayout, accointingRows[i].Date)
		if err != nil {
			config.Log.Error("Error sorting left date.", err)
			return false
		}
		rightDate, err := time.Parse(timeLayout, accointingRows[j].Date)
		if err != nil {
			config.Log.Error("Error sorting right date.", err)
			return false
		}
		return leftDate.Before(rightDate)
	})

	// Now that we are sorted, if we have a start date, drop everything from before it, if end date is set, drop everything after it
	var firstToKeep *int
	var lastToKeep *int
	for i := range accointingRows {
		if startDate != nil && firstToKeep == nil {
			rowDate, err := time.Parse(timeLayout, accointingRows[i].Date)
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
			rowDate, err := time.Parse(timeLayout, accointingRows[i].Date)
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
		accointingRows = accointingRows[*firstToKeep:*lastToKeep]
	} else if firstToKeep != nil {
		accointingRows = accointingRows[*firstToKeep:]
	} else if lastToKeep != nil {
		accointingRows = accointingRows[:*lastToKeep]
	}

	// Copy AccointingRows into csvRows for return val
	csvRows := make([]parsers.CsvRow, len(accointingRows))
	for i, v := range accointingRows {
		v.Comments = fmt.Sprintf("Address: %s %s", address, v.Comments)
		csvRows[i] = v
	}

	return csvRows
}

func (p Parser) GetHeaders() []string {
	return []string{"transactionType", "date", "inBuyAmount", "inBuyAsset", "outSellAmount", "outSellAsset",
		"feeAmount (optional)", "feeAsset (optional)", "classification (optional)", "operationId (optional)", "comments (optional)"}
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
			config.Log.Fatal("error parsing row. Should be impossible to reach this condition, ideally (once all bugs worked out)", err)
		}
		rows = append(rows, row)
	}

	// rows = HandleFees(address, events, rows) TODO we have no fee handler for taxable EVENTS right now
	return rows
}

// ParseTx: Parse the potentially taxable TX and Messages
// This function is used for parsing a single TX that will not need to relate to any others
// Use TX Parsing Groups to parse txes as a group.
// NOTE: just because two message types are related on chain does not mean they must be parsed as a group.
// Whether or not a message must be parsed as a group depends on whether the taxable implications are clear without further context.
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

// ParseMsgValidatorWithdraw:
// This transaction is always a withdrawal.
func ParseMsgWithdrawValidatorCommission(address string, event db.TaxableTransaction) Row {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Fatal("Error with ParseMsgWithdrawValidatorCommission.", err)
	}
	row.Classification = Staked
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
	row.Classification = Staked
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
	selfTransfer := false

	senderAddrPrefix := core.GetAddressPrefix(event.SenderAddress.Address)
	receiverAddrPrefix := core.GetAddressPrefix(event.ReceiverAddress.Address)
	if senderAddrPrefix != "" && receiverAddrPrefix != "" {
		selfTransfer = core.IsAddressEqual(event.SenderAddress.Address, senderAddrPrefix, event.ReceiverAddress.Address, receiverAddrPrefix)
	}

	// The base bech32 address was the same, so this was a self transfer and is not taxable
	if selfTransfer {
		row.Classification = Ignored
	}

	if err != nil {
		config.Log.Fatal("Error with ParseMsgTransfer.", err)
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
