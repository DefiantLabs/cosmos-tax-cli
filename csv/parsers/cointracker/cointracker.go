package cointracker

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
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/concentratedliquidity"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/gamm"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/poolmanager"
	"github.com/DefiantLabs/cosmos-tax-cli/util"
)

func (p *Parser) TimeLayout() string {
	return TimeLayout
}

func (p *Parser) ProcessTaxableTx(address string, taxableTxs []db.TaxableTransaction, taxableFees []db.Fee) error {
	// Build a map, so we know which TX go with which messages
	txMap := parsers.MakeTXMap(taxableTxs)

	feesWithoutTx := []db.Fee{}
	for _, fee := range taxableFees {
		if _, ok := txMap[fee.Tx.ID]; !ok {
			feesWithoutTx = append(feesWithoutTx, fee)
		}
	}

	// Pull messages out of txMap that must be grouped together
	parsers.SeparateParsingGroups(txMap, p.ParsingGroups)

	// Parse all the potentially taxable events (one transaction group at a time)
	for _, txGroup := range txMap {
		// All messages have been removed into a parsing group
		if len(txGroup) != 0 {
			var fees []db.Fee

			if len(txGroup) > 0 {
				fees = txGroup[0].Message.Tx.Fees
			}
			// For the current transaction group, generate the rows for the CSV.
			// Usually (but not always) a transaction will only have a single row in the CSV.
			txRows, err := ParseTx(address, txGroup, fees)
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

	for _, fee := range feesWithoutTx {
		row := Row{}
		err := row.ParseFee(fee.Tx, fee)
		if err != nil {
			return err
		}

		p.Rows = append(p.Rows, row)
	}

	return nil
}

func (p *Parser) ProcessTaxableEvent(taxableEvents []db.TaxableEvent) error {
	// Parse all the potentially taxable events
	for _, event := range taxableEvents {
		// generate the rows for the CSV.
		rows, err := ParseEvent(event)
		if err != nil {
			return err
		}
		p.Rows = append(p.Rows, rows...)
	}

	return nil
}

func (p *Parser) InitializeParsingGroups() {
	p.ParsingGroups = append(p.ParsingGroups, parsers.GetOsmosisTxParsingGroups()...)
}

func (p *Parser) GetRows(address string, startDate, endDate *time.Time) ([]parsers.CsvRow, error) {
	// Combine all normal rows and parser group rows into 1
	cointrackerRows := p.Rows // contains TX rows and fees as well as taxable events
	for _, v := range p.ParsingGroups {
		for _, row := range v.GetRowsForParsingGroup() {
			cointrackerRows = append(cointrackerRows, row.(Row))
		}
	}

	// Sort by date
	sort.Slice(cointrackerRows, func(i int, j int) bool {
		leftDate, err := time.Parse(TimeLayout, cointrackerRows[i].Date)
		if err != nil {
			config.Log.Error("Error sorting left date.", err)
			return false
		}
		rightDate, err := time.Parse(TimeLayout, cointrackerRows[j].Date)
		if err != nil {
			config.Log.Error("Error sorting right date.", err)
			return false
		}
		return leftDate.Before(rightDate)
	})

	// Now that we are sorted, if we have a start date, drop everything from before it, if end date is set, drop everything after it
	// Now that we are sorted, if we have a start date, drop everything from before it, if end date is set, drop everything after it
	var rowsToKeep []*Row
	for i := range cointrackerRows {
		rowDate, err := time.Parse(TimeLayout, cointrackerRows[i].Date)
		if err != nil {
			config.Log.Error("Error parsing row date.", err)
			return nil, err
		}
		if startDate != nil && rowDate.Before(*startDate) {
			continue
		}
		if endDate != nil && rowDate.After(*endDate) {
			break
		}
		rowsToKeep = append(rowsToKeep, &cointrackerRows[i])
	}

	// Copy cointrackerRows into csvRows for return val
	csvRows := make([]parsers.CsvRow, len(rowsToKeep))
	for i, v := range rowsToKeep {
		csvRows[i] = v
	}

	return csvRows, nil
}

func (p Parser) GetHeaders() []string {
	return []string{"Date", "Received Quantity", "Received Currency", "Sent Quantity", "Sent Currency", "Fee Amount", "Fee Currency", "Tag"}
}

// ParseEvent: Parse the potentially taxable event
func ParseEvent(event db.TaxableEvent) (rows []Row, err error) {
	if event.Source == db.OsmosisRewardDistribution {
		row, err := ParseOsmosisReward(event)
		if err != nil {
			config.Log.Error("error parsing row. Should be impossible to reach this condition, ideally (once all bugs worked out)", err)
			return nil, err
		}
		rows = append(rows, row)
	}

	return rows, err
}

// ParseTx: Parse the potentially taxable TX and Messages
// This function is used for parsing a single TX that will not need to relate to any others
// Use TX Parsing Groups to parse txes as a group
func ParseTx(address string, events []db.TaxableTransaction, fees []db.Fee) (rows []parsers.CsvRow, err error) {
	currFeeIndex := 0
	for _, event := range events {
		var newRow Row
		var err error
		switch event.Message.MessageType.MessageType {
		case bank.MsgSendV0:
			newRow, err = ParseMsgSend(address, event)
		case bank.MsgSend:
			newRow, err = ParseMsgSend(address, event)
		case bank.MsgMultiSendV0:
			newRow, err = ParseMsgMultiSend(address, event)
		case bank.MsgMultiSend:
			newRow, err = ParseMsgMultiSend(address, event)
		case distribution.MsgFundCommunityPool:
			newRow, err = ParseMsgFundCommunityPool(address, event)
		case distribution.MsgWithdrawValidatorCommission:
			newRow, err = ParseMsgWithdrawValidatorCommission(address, event)
		case distribution.MsgWithdrawRewards:
			newRow, err = ParseMsgWithdrawDelegatorReward(address, event)
		case distribution.MsgWithdrawDelegatorReward:
			newRow, err = ParseMsgWithdrawDelegatorReward(address, event)
		case staking.MsgDelegate:
			newRow, err = ParseMsgWithdrawDelegatorReward(address, event)
		case staking.MsgUndelegate:
			newRow, err = ParseMsgWithdrawDelegatorReward(address, event)
		case staking.MsgBeginRedelegate:
			newRow, err = ParseMsgWithdrawDelegatorReward(address, event)
		case gov.MsgSubmitProposal, gov.MsgSubmitProposalV1:
			newRow, err = ParseMsgSubmitProposal(address, event)
		case gov.MsgDeposit, gov.MsgDepositV1:
			newRow, err = ParseMsgDeposit(address, event)
		case gamm.MsgSwapExactAmountIn:
			newRow, err = ParseMsgSwapExactAmountIn(event)
		case gamm.MsgSwapExactAmountOut:
			newRow, err = ParseMsgSwapExactAmountOut(event)
		case ibc.MsgTransfer:
			newRow, err = ParseMsgTransfer(address, event)
		case ibc.MsgAcknowledgement:
			newRow, err = ParseMsgAcknowledgement(address, event)
		case ibc.MsgRecvPacket:
			newRow, err = ParseMsgRecvPacket(address, event)
		case poolmanager.MsgSplitRouteSwapExactAmountIn, poolmanager.MsgSwapExactAmountIn, poolmanager.MsgSwapExactAmountOut:
			newRow, err = ParsePoolManagerSwap(event)
		case concentratedliquidity.MsgCollectIncentives, concentratedliquidity.MsgCollectSpreadRewards:
			newRow, err = ParseConcentratedLiquidityCollection(event)
		default:
			config.Log.Errorf("no parser for message type '%v'", event.Message.MessageType.MessageType)
			continue
		}

		if err != nil {
			config.Log.Errorf("error parsing message type '%v': %v", event.Message.MessageType.MessageType, err)
			continue
		}

		// Attach fees to the transaction events
		if currFeeIndex < len(fees) {
			if fees[currFeeIndex].PayerAddress.Address == address {
				conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(fees[currFeeIndex].Amount), fees[currFeeIndex].Denomination)
				if err != nil {
					config.Log.Errorf("error parsing fee: %v", err)
				} else {
					newRow.FeeAmount = conversionAmount.String()
					newRow.FeeCurrency = conversionSymbol
				}
			}
			currFeeIndex++
		}

		rows = append(rows, newRow)
	}

	// Check if fees have all been processed based on last processed index
	if currFeeIndex < len(fees) {
		// Create empty row for the fees that weren't processed
		for i := currFeeIndex; i < len(fees); i++ {
			if fees[i].PayerAddress.Address == address {
				newRow := Row{}
				err = newRow.ParseFee(fees[i].Tx, fees[i])
				if err != nil {
					config.Log.Errorf("error parsing fee: %v", err)
					continue
				}
				rows = append(rows, newRow)
			}
		}
	}

	return rows, nil
}

// ParseMsgValidatorWithdraw:
// This transaction is always a withdrawal.
func ParseMsgWithdrawValidatorCommission(address string, event db.TaxableTransaction) (Row, error) {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Error("Error with ParseMsgWithdrawValidatorCommission.", err)
	}
	// row.Label = Unstake
	return *row, err
}

// ParseMsgValidatorWithdraw:
// This transaction is always a withdrawal.
func ParseMsgWithdrawDelegatorReward(address string, event db.TaxableTransaction) (Row, error) {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Error("Error with ParseMsgWithdrawDelegatorReward.", err)
	}
	// row.Label = Unstake
	return *row, err
}

// ParseMsgSend:
// If the address we searched is the receiver, then this transaction is a deposit.
// If the address we searched is the sender, then this transaction is a withdrawal.
func ParseMsgSend(address string, event db.TaxableTransaction) (Row, error) {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Error("Error with ParseMsgSend.", err)
	}
	return *row, err
}

func ParseMsgMultiSend(address string, event db.TaxableTransaction) (Row, error) {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Error("Error with ParseMsgMultiSend.", err)
	}
	return *row, err
}

func ParseMsgFundCommunityPool(address string, event db.TaxableTransaction) (Row, error) {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Error("Error with ParseMsgFundCommunityPool.", err)
	}
	return *row, err
}

func ParseMsgSwapExactAmountIn(event db.TaxableTransaction) (Row, error) {
	row := &Row{}
	err := row.ParseSwap(event)
	if err != nil {
		config.Log.Error("Error with ParseMsgSwapExactAmountIn.", err)
	}
	return *row, err
}

func ParseMsgSwapExactAmountOut(event db.TaxableTransaction) (Row, error) {
	row := &Row{}
	err := row.ParseSwap(event)
	if err != nil {
		config.Log.Error("Error with ParseMsgSwapExactAmountOut.", err)
	}
	return *row, err
}

func ParseMsgTransfer(address string, event db.TaxableTransaction) (Row, error) {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Error("Error with ParseMsgTransfer.", err)
	}
	return *row, err
}

func ParseMsgAcknowledgement(address string, event db.TaxableTransaction) (Row, error) {
	row := &Row{}

	denomToUse := event.DenominationSent
	amountToUse := event.AmountSent

	conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(amountToUse), denomToUse)
	if err != nil {
		config.Log.Error("Error with ParseMsgAcknowledgement.", err)
		return *row, fmt.Errorf("cannot parse denom units for TX %s (classification: withdrawal)", event.Message.Tx.Hash)
	}

	if event.ReceiverAddress.Address == address {
		row.ReceivedAmount = conversionAmount.Text('f', -1)
		row.ReceivedCurrency = conversionSymbol
	} else if event.SenderAddress.Address == address { // withdrawal
		row.SentAmount = conversionAmount.Text('f', -1)
		row.SentCurrency = conversionSymbol
	}

	row.Date = event.Message.Tx.Block.TimeStamp.Format(TimeLayout)
	return *row, err
}

func ParseMsgRecvPacket(address string, event db.TaxableTransaction) (Row, error) {
	row := &Row{}

	denomToUse := event.DenominationReceived
	amountToUse := event.AmountReceived

	conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(amountToUse), denomToUse)
	if err != nil {
		config.Log.Error("Error with ParseMsgAcknowledgement.", err)
		return *row, fmt.Errorf("cannot parse denom units for TX %s (classification: withdrawal)", event.Message.Tx.Hash)
	}

	if event.ReceiverAddress.Address == address {
		row.ReceivedAmount = conversionAmount.Text('f', -1)
		row.ReceivedCurrency = conversionSymbol
	} else if event.SenderAddress.Address == address { // withdrawal
		row.SentAmount = conversionAmount.Text('f', -1)
		row.SentCurrency = conversionSymbol
	}

	row.Date = event.Message.Tx.Block.TimeStamp.Format(TimeLayout)
	return *row, err
}

func ParseMsgSubmitProposal(address string, event db.TaxableTransaction) (Row, error) {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Error("Error with ParseMsgSubmitProposal.", err)
	}
	return *row, err
}

func ParseMsgDeposit(address string, event db.TaxableTransaction) (Row, error) {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Error("Error with ParseMsgDeposit.", err)
	}
	return *row, err
}

func ParsePoolManagerSwap(event db.TaxableTransaction) (Row, error) {
	row := &Row{}
	err := row.ParseSwap(event)
	if err != nil {
		config.Log.Error("Error with ParseMsgSwapExactAmountOut.", err)
	}
	return *row, err
}

func ParseOsmosisReward(event db.TaxableEvent) (Row, error) {
	row := &Row{}
	err := row.EventParseBasic(event)
	if err != nil {
		config.Log.Error("Error with ParseOsmosisReward.", err)
	}
	return *row, err
}

func ParseConcentratedLiquidityCollection(event db.TaxableTransaction) (Row, error) {
	row := &Row{}
	denomToUse := event.DenominationReceived
	amountToUse := event.AmountReceived

	conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(amountToUse), denomToUse)
	if err != nil {
		config.Log.Error("Error with ParseConcentratedLiquidityCollection.", err)
		return *row, fmt.Errorf("cannot parse denom units for TX %s (classification: deposit)", event.Message.Tx.Hash)
	}

	row.ReceivedAmount = conversionAmount.Text('f', -1)
	row.ReceivedCurrency = conversionSymbol
	row.Date = event.Message.Tx.Block.TimeStamp.Format(TimeLayout)

	return *row, err
}
