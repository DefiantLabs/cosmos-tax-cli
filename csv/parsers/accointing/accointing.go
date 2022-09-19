package accointing

import (
	"fmt"
	"os"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/bank"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/staking"
	"github.com/DefiantLabs/cosmos-tax-cli/csv/parsers"
	"github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/gamm"
)

func (p *AccointingParser) ProcessTaxableTx(address string, taxableTxs []db.TaxableTransaction) error {
	//process taxableTx into Rows above
	txMap := map[uint][]db.TaxableTransaction{} //Map transaction ID to List of events

	//Build a map so we know which TX go with which messages
	for _, taxableTx := range taxableTxs {
		if list, ok := txMap[taxableTx.Message.Tx.ID]; ok {
			list = append(list, taxableTx)
			txMap[taxableTx.Message.Tx.ID] = list
		} else {
			txMap[taxableTx.Message.Tx.ID] = []db.TaxableTransaction{taxableTx}
		}
	}

	//TODO: Can probably reduce complexity
	//The basic idea is we want to do the following:
	//1. Loop through each message for each transaction
	//2. Check if it belongs in a group by message type
	//3. Gather indicies of all messages that belong in each group
	//4. Remove them from the normal txMap
	//5. Add them to the group-secific txMap
	//The last two steps ensure that the message will not be parsed twice
	for v, tx := range txMap {
		//map: [group index] to []indexes of the current tx messages that belong in that group
		var groupsToMessageIds map[int][]int = make(map[int][]int)

		//TODO: Remove me, useless outside print for demo
		messagesToRemove := 0

		for messageIndex, message := range tx {
			for groupIndex, txGroup := range p.ParsingGroups {
				//Store index of current message if it belongs in the group
				if txGroup.BelongsToGroup(message) {
					if _, ok := groupsToMessageIds[groupIndex]; ok {
						groupsToMessageIds[groupIndex] = append(groupsToMessageIds[groupIndex], messageIndex)
					} else {
						var messageArray []int
						messageArray = append(messageArray, messageIndex)
						groupsToMessageIds[groupIndex] = messageArray
					}
					messagesToRemove += 1

					//Add it to the first group it belongs to and no others
					//This establishes a precedence and prevents messages from being duplicated in many groups
					break
				}
			}
		}

		//split off the messages into their respective group
		for groupIndex, messageIndices := range groupsToMessageIds {
			var currentGroup parsers.ParsingGroup = p.ParsingGroups[groupIndex]

			//used to keep the index relevant after splicing
			numElementsRemoved := 0
			for _, messageIndex := range messageIndices {
				//Get message to remove at index - numElementsRemoved
				var indexToRemove int = messageIndex - numElementsRemoved
				var messageToRemove db.TaxableTransaction = tx[indexToRemove]

				//Add to group and remove from original TX
				currentGroup.AddTxToGroup(messageToRemove)
				tx = append(tx[:indexToRemove], tx[indexToRemove+1:]...)
				//overwrite the txMaps value at this tx to remove
				txMap[v] = tx
				numElementsRemoved = numElementsRemoved + 1
			}

		}
	}

	//Parse all the potentially taxable events (one transaction group at a time)
	for _, txGroup := range txMap {
		//All messages have been removed into a parsing group
		if len(txGroup) != 0 {
			//For the current transaction group, generate the rows for the CSV.
			//Usually (but not always) a transaction will only have a single row in the CSV.
			txRows, err := ParseTx(address, txGroup)
			if err == nil {
				for _, v := range txRows {
					p.Rows = append(p.Rows, v.(AccointingRow))
				}
			} else {
				return err
			}
		}
	}

	//Parse all the txes found in the Parsing Groups
	for _, txParsingGroup := range p.ParsingGroups {
		txParsingGroup.ParseGroup()
	}

	return nil
}

func (p *AccointingParser) ProcessTaxableEvent(address string, taxableEvents []db.TaxableEvent) error {
	//process taxableTx into Rows above

	if len(taxableEvents) == 0 {
		return nil
	}

	//Parse all the potentially taxable events
	for _, event := range taxableEvents {
		//generate the rows for the CSV.
		p.Rows = append(p.Rows, ParseEvent(address, event)...)
	}

	return nil
}

func (p *AccointingParser) InitializeParsingGroups(config config.Config) {
	switch config.Lens.ChainID {
	case "osmosis-1":
		for _, v := range GetOsmosisTxParsingGroups() {
			p.ParsingGroups = append(p.ParsingGroups, v)
		}
	}
}

func (p *AccointingParser) GetRows() []parsers.CsvRow {
	rows := make([]parsers.CsvRow, len(p.Rows))

	for i, v := range p.Rows {
		rows[i] = v
	}

	for _, v := range p.ParsingGroups {
		for _, vv := range v.GetRowsForParsingGroup() {
			rows = append(rows, vv)
		}
	}

	return rows
}

func (parser AccointingParser) GetHeaders() []string {
	return []string{"transactionType", "date", "inBuyAmount", "inBuyAsset", "outSellAmount", "outSellAsset",
		"feeAmount (optional)", "feeAsset (optional)", "classification (optional)", "operationId (optional)", "comments (optional)"}
}

//HandleFees:
//If the transaction lists the same amount of fees as there are rows in the CSV,
//then we spread the fees out one per row. Otherwise we add a line for the fees,
//where each fee has a separate line.
func HandleFees(address string, events []db.TaxableTransaction, rows []AccointingRow) ([]AccointingRow, error) {
	//No events -- This address didn't pay any fees
	if len(events) == 0 {
		return rows, nil
	}

	fees := events[0].Message.Tx.Fees

	for _, fee := range fees {
		payer := fee.PayerAddress.Address
		if payer != address {
			return rows, nil
		}
	}

	//Stick the fees in the existing rows.
	if len(rows) >= len(fees) {
		for i, fee := range fees {
			conversionAmount, conversionSymbol, err := db.ConvertUnits(fee.Amount.BigInt(), fee.Denomination)
			if err == nil {
				rows[i].FeeAmount = conversionAmount.String()
				rows[i].FeeAsset = conversionSymbol
			} else {
				return nil, fmt.Errorf("Cannot parse fee units for TX %s\n", events[0].Message.Tx.Hash)
			}
		}

		return rows, nil
	}

	tx := events[0].Message.Tx
	//There's more fees than rows so generate a new row for each fee.
	for _, fee := range fees {
		feeUnits, feeSymbol, err := db.ConvertUnits(fee.Amount.BigInt(), fee.Denomination)
		if err != nil {
			return nil, fmt.Errorf("Cannot parse fee units for TX %s\n", events[0].Message.Tx.Hash)
		}

		newRow := AccointingRow{Date: FormatDatetime(tx.TimeStamp), FeeAmount: feeUnits.String(),
			FeeAsset: feeSymbol, Classification: Fee, TransactionType: Withdraw}
		rows = append(rows, newRow)
	}

	return rows, nil
}

//ParseEvent: Parse the potentially taxable event
func ParseEvent(address string, event db.TaxableEvent) []AccointingRow {
	rows := []AccointingRow{}

	if event.Source == db.OsmosisRewardDistribution {
		row, err := ParseOsmosisReward(address, event)
		if err == nil {
			rows = append(rows, row)
		} else {
			//TODO: handle error parsing row. Should be impossible to reach this condition, ideally (once all bugs worked out)
			os.Exit(1)
		}
	}

	//rows = HandleFees(address, events, rows) TODO we have no fee handler for taxable EVENTS right now
	return rows
}

//ParseTx: Parse the potentially taxable TX and Messages
//This function is used for parsing a single TX that will not need to relate to any others
//Use TX Parsing Groups to parse txes as a group
func ParseTx(address string, events []db.TaxableTransaction) ([]parsers.CsvRow, error) {
	rows := []parsers.CsvRow{}

	for _, event := range events {
		//Is this a MsgSend
		if bank.IsMsgSend[event.Message.MessageType] {
			rows = append(rows, ParseMsgSend(address, event))
		} else if staking.IsMsgWithdrawValidatorCommission[event.Message.MessageType] {
			rows = append(rows, ParseMsgWithdrawValidatorCommission(address, event))
		} else if staking.IsMsgWithdrawDelegatorReward[event.Message.MessageType] {
			rows = append(rows, ParseMsgWithdrawDelegatorReward(address, event))
		} else if gamm.IsMsgSwapExactAmountIn[event.Message.MessageType] {
			rows = append(rows, ParseMsgSwapExactAmountIn(address, event))
		} else if gamm.IsMsgSwapExactAmountOut[event.Message.MessageType] {
			rows = append(rows, ParseMsgSwapExactAmountOut(address, event))
		} else {
			fmt.Println("No parser for message type", event.Message.MessageType)
		}
	}

	// rows, err := HandleFees(address, events, rows)
	return rows, nil
}

//ParseMsgValidatorWithdraw:
//This transaction is always a withdrawal.
func ParseMsgWithdrawValidatorCommission(address string, event db.TaxableTransaction) AccointingRow {
	row := &AccointingRow{}
	row.ParseBasic(address, event)
	row.Classification = Staked
	return *row
}

//ParseMsgValidatorWithdraw:
//This transaction is always a withdrawal.
func ParseMsgWithdrawDelegatorReward(address string, event db.TaxableTransaction) AccointingRow {
	row := &AccointingRow{}
	row.ParseBasic(address, event)
	row.Classification = Staked
	return *row
}

//ParseMsgSend:
//If the address we searched is the receiver, then this transaction is a deposit.
//If the address we searched is the sender, then this transaction is a withdrawal.
func ParseMsgSend(address string, event db.TaxableTransaction) AccointingRow {
	row := &AccointingRow{}
	row.ParseBasic(address, event)
	return *row
}

func ParseMsgSwapExactAmountIn(address string, event db.TaxableTransaction) AccointingRow {
	row := &AccointingRow{}
	row.ParseSwap(address, event)
	return *row
}

func ParseMsgSwapExactAmountOut(address string, event db.TaxableTransaction) AccointingRow {
	row := &AccointingRow{}
	row.ParseSwap(address, event)
	return *row
}

func ParseOsmosisReward(address string, event db.TaxableEvent) (AccointingRow, error) {
	row := &AccointingRow{}
	err := row.EventParseBasic(address, event)
	return *row, err
}
