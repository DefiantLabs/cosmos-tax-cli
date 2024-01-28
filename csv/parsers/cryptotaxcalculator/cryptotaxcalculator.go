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
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/concentratedliquidity"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/gamm"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/poolmanager"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/tokenfactory"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/valsetpref"
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

	for _, fee := range feesWithoutTx {
		row := Row{}
		err := row.ParseFee(fee)
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

func (p *Parser) GetRows(address string, startDate, endDate *time.Time) ([]parsers.CsvRow, error) {
	// Combine all normal rows and parser group rows into 1
	cryptoRows := p.Rows // contains TX rows and fees as well as taxable events
	for _, v := range p.ParsingGroups {
		for _, row := range v.GetRowsForParsingGroup() {
			cryptoRows = append(cryptoRows, row.(Row))
		}
	}

	// Sort by date
	sort.Slice(cryptoRows, func(i int, j int) bool {
		leftDate, err := time.Parse(TimeLayout, cryptoRows[i].Date)
		if err != nil {
			config.Log.Error("Error sorting left date.", err)
			return false
		}
		rightDate, err := time.Parse(TimeLayout, cryptoRows[j].Date)
		if err != nil {
			config.Log.Error("Error sorting right date.", err)
			return false
		}
		return leftDate.Before(rightDate)
	})

	var rowsToKeep []*Row
	for i := range cryptoRows {
		rowDate, err := time.Parse(TimeLayout, cryptoRows[i].Date)
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
		rowsToKeep = append(rowsToKeep, &cryptoRows[i])
	}

	// Copy AccointingRows into csvRows for return val
	csvRows := make([]parsers.CsvRow, len(rowsToKeep))
	for i, v := range rowsToKeep {
		csvRows[i] = v
	}

	return csvRows, nil
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
func ParseEvent(event db.TaxableEvent) (rows []Row, err error) {
	if event.Source == db.OsmosisRewardDistribution {
		row, err := ParseOsmosisReward(event)
		if err != nil {
			config.Log.Error("error parsing row. Should be impossible to reach this condition, ideally (once all bugs worked out)", err)
			return nil, err
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func ParseTx(address string, events []db.TaxableTransaction) (rows []parsers.CsvRow, err error) {
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
		case gamm.MsgSwapExactAmountIn:
			newRow, err = ParseMsgSwapExactAmountIn(address, event)
		case gamm.MsgSwapExactAmountOut:
			newRow, err = ParseMsgSwapExactAmountOut(address, event)
		case gov.MsgSubmitProposal, gov.MsgSubmitProposalV1:
			newRow, err = ParseMsgSubmitProposal(address, event)
		case gov.MsgDeposit, gov.MsgDepositV1:
			newRow, err = ParseMsgDeposit(address, event)
		case ibc.MsgTransfer:
			newRow, err = ParseMsgTransfer(address, event)
		case ibc.MsgAcknowledgement:
			newRow, err = ParseMsgAcknowledgement(address, event)
		case ibc.MsgRecvPacket:
			newRow, err = ParseMsgRecvPacket(address, event)
		case poolmanager.MsgSplitRouteSwapExactAmountIn, poolmanager.MsgSwapExactAmountIn, poolmanager.MsgSwapExactAmountOut:
			newRow, err = ParsePoolManagerSwap(address, event)
		case concentratedliquidity.MsgCollectIncentives, concentratedliquidity.MsgCollectSpreadRewards:
			newRow, err = ParseConcentratedLiquidityCollection(event)
		case valsetpref.MsgDelegateBondedTokens, valsetpref.MsgUndelegateFromValidatorSet, valsetpref.MsgRedelegateValidatorSet, valsetpref.MsgWithdrawDelegationRewards, valsetpref.MsgDelegateToValidatorSet, valsetpref.MsgUndelegateFromRebalancedValidatorSet:
			newRow, err = ParseValsetPrefRewards(event)
		case tokenfactory.MsgMint, tokenfactory.MsgBurn:
			newRow, err = ParseTokenFactoryEvents(address, event)
		default:
			config.Log.Errorf("no parser for message type '%v'", event.Message.MessageType.MessageType)
			continue
		}

		if err != nil {
			config.Log.Errorf("error parsing message type '%v': %v", event.Message.MessageType.MessageType, err)
			continue
		}

		rows = append(rows, newRow)
	}
	return rows, nil
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

func ParseMsgWithdrawDelegatorReward(address string, event db.TaxableTransaction) (Row, error) {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Error("Error with ParseMsgWithdrawDelegatorReward.", err)
	}
	// row.Label = Unstake
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
		return *row, fmt.Errorf("cannot parse denom units for TX %s", event.Message.Tx.Hash)
	}

	row.BaseAmount = conversionAmount.Text('f', -1)
	row.BaseCurrency = conversionSymbol

	if event.ReceiverAddress.Address == address {
		row.Type = Buy
	} else if event.SenderAddress.Address == address { // withdrawal
		row.Type = Sell
	}

	row.From = event.SenderAddress.Address
	row.To = event.ReceiverAddress.Address

	row.Date = event.Message.Tx.Block.TimeStamp.Format(TimeLayout)
	row.ID = event.Message.Tx.Hash
	return *row, err
}

func ParseMsgRecvPacket(address string, event db.TaxableTransaction) (Row, error) {
	row := &Row{}

	denomToUse := event.DenominationReceived
	amountToUse := event.AmountReceived

	conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(amountToUse), denomToUse)
	if err != nil {
		config.Log.Error("Error with ParseMsgAcknowledgement.", err)
		return *row, fmt.Errorf("cannot parse denom units for TX %s", event.Message.Tx.Hash)
	}

	row.BaseAmount = conversionAmount.Text('f', -1)
	row.BaseCurrency = conversionSymbol

	if event.ReceiverAddress.Address == address {
		row.Type = Buy
	} else if event.SenderAddress.Address == address { // withdrawal
		row.Type = Sell
	}

	row.From = event.SenderAddress.Address
	row.To = event.ReceiverAddress.Address

	row.Date = event.Message.Tx.Block.TimeStamp.Format(TimeLayout)
	row.ID = event.Message.Tx.Hash
	return *row, err
}

func (p *Parser) InitializeParsingGroups() {
	p.ParsingGroups = append(p.ParsingGroups, &OsmosisLpTxGroup{}, &OsmosisConcentratedLiquidityTxGroup{})
}

func ParseOsmosisReward(event db.TaxableEvent) (Row, error) {
	row := &Row{}
	err := row.EventParseBasic(event)
	if err != nil {
		config.Log.Error("Error with ParseOsmosisReward.", err)
	}
	return *row, err
}

func ParseMsgSwapExactAmountIn(address string, event db.TaxableTransaction) (Row, error) {
	row := &Row{}
	err := row.ParseSwap(event, address, Buy)
	if err != nil {
		config.Log.Error("Error with ParseMsgSwapExactAmountIn.", err)
	}
	return *row, err
}

func ParseMsgSwapExactAmountOut(address string, event db.TaxableTransaction) (Row, error) {
	row := &Row{}
	err := row.ParseSwap(event, address, Sell)
	if err != nil {
		config.Log.Error("Error with ParseMsgSwapExactAmountOut.", err)
	}
	return *row, err
}

func ParsePoolManagerSwap(address string, event db.TaxableTransaction) (Row, error) {
	row := &Row{}
	err := row.ParseSwap(event, address, Buy)
	if err != nil {
		config.Log.Error("Error with ParsePoolManagerSwap.", err)
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

	row.BaseAmount = conversionAmount.Text('f', -1)
	row.BaseCurrency = conversionSymbol
	row.Type = Receive
	row.Date = event.Message.Tx.Block.TimeStamp.Format(TimeLayout)

	return *row, err
}

func ParseValsetPrefRewards(event db.TaxableTransaction) (Row, error) {
	row := &Row{}
	row.Type = Receive
	row.Date = event.Message.Tx.Block.TimeStamp.Format(TimeLayout)

	err := parseAndAddReceivedAmount(row, event)
	if err != nil {
		return *row, err
	}

	return *row, nil
}

func ParseTokenFactoryEvents(address string, event db.TaxableTransaction) (Row, error) {
	row := &Row{}
	err := row.ParseBasic(address, event)
	if err != nil {
		config.Log.Error("Error with ParseMsgMultiSend.", err)
	}

	return *row, nil
}
