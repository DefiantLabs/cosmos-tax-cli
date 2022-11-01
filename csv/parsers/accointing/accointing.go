package accointing

import (
	"fmt"
	"go.uber.org/zap"
	"sort"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/bank"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/distribution"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/staking"
	"github.com/DefiantLabs/cosmos-tax-cli/csv/parsers"
	"github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/gamm"
)

func (p *Parser) ProcessTaxableTx(address string, taxableTxs []db.TaxableTransaction) error {
	// Build a map, so we know which TX go with which messages
	txMap := parsers.MakeTXMap(taxableTxs)

	// Pull messages out of txMap that must be grouped together
	parsers.SeparateParsingGroups(txMap, p.ParsingGroups)

	//Parse all the potentially taxable events (one transaction group at a time)
	for _, txGroup := range txMap {
		//All messages have been removed into a parsing group
		if len(txGroup) != 0 {
			//For the current transaction group, generate the rows for the CSV.
			//Usually (but not always) a transaction will only have a single row in the CSV.
			txRows, err := ParseTx(address, txGroup)
			if err != nil {
				return err
			}
			for _, v := range txRows {
				p.Rows = append(p.Rows, v.(Row))
			}
		}
	}

	//Parse all the TXs found in the Parsing Groups
	for _, txParsingGroup := range p.ParsingGroups {
		err := txParsingGroup.ParseGroup()
		if err != nil {
			return err
		}
	}

	//Handle fees on all taxableTxs at once, we don't do this in the regular parser or in the parsing groups
	//This requires HandleFees to process the fees into unique mappings of tx -> fees (since we gather Taxable Messages in the taxableTxs)
	//If we move it into the ParseTx function or into the ParseGroup function, we may be able to reduce the logic in the HandleFees func
	feeRows, err := HandleFees(address, taxableTxs)
	if err != nil {
		return err
	}

	p.Rows = append(p.Rows, feeRows...)

	return nil
}

func (p *Parser) ProcessTaxableEvent(address string, taxableEvents []db.TaxableEvent) error {
	//Parse all the potentially taxable events
	for _, event := range taxableEvents {
		//generate the rows for the CSV.
		p.Rows = append(p.Rows, ParseEvent(address, event)...)
	}

	return nil
}

func (p *Parser) InitializeParsingGroups(config config.Config) {
	switch config.Lens.ChainID {
	case osmosis.ChainID:
		p.ParsingGroups = append(p.ParsingGroups, GetOsmosisTxParsingGroups()...)
	}
}

func (p *Parser) GetRows() []parsers.CsvRow {
	//Combine all normal rows and parser group rows into 1
	var accointingRows []Row
	var parserGroupRows []Row

	for _, v := range p.ParsingGroups {
		for _, row := range v.GetRowsForParsingGroup() {
			parserGroupRows = append(parserGroupRows, row.(Row))
		}
	}

	accointingRows = append(accointingRows, p.Rows...)
	accointingRows = append(accointingRows, parserGroupRows...)

	//Sort by date
	sort.Slice(accointingRows, func(i int, j int) bool {
		leftDate, err := time.Parse(timeLayout, accointingRows[i].Date)
		if err != nil {
			config.Log.Error("Error sorting left date.", zap.Error(err))
			return false
		}
		rightDate, err := time.Parse(timeLayout, accointingRows[j].Date)
		if err != nil {
			config.Log.Error("Error sorting right date.", zap.Error(err))
			return false
		}
		return leftDate.Before(rightDate)
	})

	//Copy AccointingRows into CsvRows for return val
	rows := make([]parsers.CsvRow, len(accointingRows))

	for i, v := range accointingRows {
		rows[i] = v
	}

	return rows
}

func (p Parser) GetHeaders() []string {
	return []string{"transactionType", "date", "inBuyAmount", "inBuyAsset", "outSellAmount", "outSellAsset",
		"feeAmount (optional)", "feeAsset (optional)", "classification (optional)", "operationId (optional)", "comments (optional)"}
}

// HandleFees:
// If the transaction lists the same amount of fees as there are rows in the CSV,
// then we spread the fees out one per row. Otherwise we add a line for the fees,
// where each fee has a separate line.
func HandleFees(address string, events []db.TaxableTransaction) ([]Row, error) {
	var rows []Row
	//No events -- This address didn't pay any fees
	if len(events) == 0 {
		return rows, nil
	}

	//We need to gather all unique fees, but we are receiving Messages not Txes
	//Make a map from TX hash to fees array to keep unique
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
				err := newRow.ParseFee(txIdsToTx[id], fee)
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
func ParseEvent(address string, event db.TaxableEvent) []Row {
	rows := []Row{}

	if event.Source == db.OsmosisRewardDistribution {
		row, err := ParseOsmosisReward(address, event)
		if err != nil {
			//TODO: handle error parsing row. Should be impossible to reach this condition, ideally (once all bugs worked out)
			config.Log.Fatal("error parsing row. Should be impossible to reach this condition, ideally (once all bugs worked out)", zap.Error(err))
		}
		rows = append(rows, row)
	}

	//rows = HandleFees(address, events, rows) TODO we have no fee handler for taxable EVENTS right now
	return rows
}

// ParseTx: Parse the potentially taxable TX and Messages
// This function is used for parsing a single TX that will not need to relate to any others
// Use TX Parsing Groups to parse txes as a group
func ParseTx(address string, events []db.TaxableTransaction) ([]parsers.CsvRow, error) {
	rows := []parsers.CsvRow{}

	for _, event := range events {
		//Is this a MsgSend
		if bank.IsMsgSend[event.Message.MessageType.MessageType] {
			rows = append(rows, ParseMsgSend(address, event))
		} else if bank.IsMsgMultiSend[event.Message.MessageType.MessageType] {
			rows = append(rows, ParseMsgMultiSend(address, event))
		} else if distribution.IsMsgFundCommunityPool[event.Message.MessageType.MessageType] {
			rows = append(rows, ParseMsgFundCommunityPool(address, event))
		} else if distribution.IsMsgWithdrawValidatorCommission[event.Message.MessageType.MessageType] {
			rows = append(rows, ParseMsgWithdrawValidatorCommission(address, event))
		} else if distribution.IsMsgWithdrawDelegatorReward[event.Message.MessageType.MessageType] {
			rows = append(rows, ParseMsgWithdrawDelegatorReward(address, event))
		} else if staking.IsMsgDelegate[event.Message.MessageType.MessageType] {
			rows = append(rows, ParseMsgWithdrawDelegatorReward(address, event))
		} else if staking.IsMsgUndelegate[event.Message.MessageType.MessageType] {
			rows = append(rows, ParseMsgWithdrawDelegatorReward(address, event))
		} else if staking.IsMsgBeginRedelegate[event.Message.MessageType.MessageType] {
			rows = append(rows, ParseMsgWithdrawDelegatorReward(address, event))
		} else if gamm.IsMsgSwapExactAmountIn[event.Message.MessageType.MessageType] {
			rows = append(rows, ParseMsgSwapExactAmountIn(event))
		} else if gamm.IsMsgSwapExactAmountOut[event.Message.MessageType.MessageType] {
			rows = append(rows, ParseMsgSwapExactAmountOut(event))
		} else {
			config.Log.Error(fmt.Sprintf("No parser for message type '%v'", event.Message.MessageType.MessageType))
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
	row.Classification = Staked
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

func ParseOsmosisReward(address string, event db.TaxableEvent) (Row, error) {
	row := &Row{}
	err := row.EventParseBasic(address, event)
	if err != nil {
		config.Log.Fatal("Error with ParseOsmosisReward.", zap.Error(err))
	}
	return *row, err
}
