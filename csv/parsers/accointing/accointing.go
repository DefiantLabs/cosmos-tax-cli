package accointing

import (
	"fmt"
	"go.uber.org/zap"
	"log"
	"sort"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/bank"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/distribution"
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
		p.ParsingGroups = append(p.ParsingGroups, GetOsmosisTxParsingGroups()...)
	}
}

func (p *AccointingParser) GetRows() []parsers.CsvRow {
	//Combine all normal rows and parser group rows into 1
	var accointingRows []AccointingRow
	var parserGroupRows []AccointingRow

	for _, v := range p.ParsingGroups {
		for _, row := range v.GetRowsForParsingGroup() {
			parserGroupRows = append(parserGroupRows, row.(AccointingRow))
		}
	}

	accointingRows = append(accointingRows, p.Rows...)
	accointingRows = append(accointingRows, parserGroupRows...)

	//Sort by date
	sort.Slice(accointingRows, func(i int, j int) bool {
		leftDate, err := DateFromString(accointingRows[i].Date)
		if err != nil {
			config.Log.Error("Error sorting left date.", zap.Error(err))
			return false
		}
		rightDate, err := DateFromString(accointingRows[j].Date)
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

func (parser AccointingParser) GetHeaders() []string {
	return []string{"transactionType", "date", "inBuyAmount", "inBuyAsset", "outSellAmount", "outSellAsset",
		"feeAmount (optional)", "feeAsset (optional)", "classification (optional)", "operationId (optional)", "comments (optional)"}
}

// HandleFees:
// If the transaction lists the same amount of fees as there are rows in the CSV,
// then we spread the fees out one per row. Otherwise we add a line for the fees,
// where each fee has a separate line.
func HandleFees(address string, events []db.TaxableTransaction) ([]AccointingRow, error) {
	var rows []AccointingRow
	//No events -- This address didn't pay any fees
	if len(events) == 0 {
		return rows, nil
	}

	//We need to gather all unique fees, but we are receiving Messages not Txes
	//Make a map from TX hash to fees array to keep unique
	txToFeesMap := make(map[uint][]db.Fee)
	txIdsToTx := make(map[uint]db.Tx)
	for _, event := range events {
		txId := event.Message.Tx.ID
		feeStore := event.Message.Tx.Fees
		txToFeesMap[txId] = feeStore
		txIdsToTx[txId] = event.Message.Tx
	}

	for id, txFees := range txToFeesMap {
		for _, fee := range txFees {
			if fee.PayerAddress.Address == address {
				newRow := AccointingRow{}
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
func ParseEvent(address string, event db.TaxableEvent) []AccointingRow {
	rows := []AccointingRow{}

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
			rows = append(rows, ParseMsgSwapExactAmountIn(address, event))
		} else if gamm.IsMsgSwapExactAmountOut[event.Message.MessageType.MessageType] {
			rows = append(rows, ParseMsgSwapExactAmountOut(address, event))
		} else {
			config.Log.Error(fmt.Sprintf("No parser for message type '%v'", event.Message.MessageType.MessageType))
		}
	}

	return rows, nil
}

// ParseMsgValidatorWithdraw:
// This transaction is always a withdrawal.
func ParseMsgWithdrawValidatorCommission(address string, event db.TaxableTransaction) AccointingRow {
	row := &AccointingRow{}
	err := row.ParseBasic(address, event)
	if err != nil {
		log.Printf("Error with ParseMsgWithdrawValidatorCommission. Err: %v", err)
	}
	row.Classification = Staked
	return *row
}

// ParseMsgValidatorWithdraw:
// This transaction is always a withdrawal.
func ParseMsgWithdrawDelegatorReward(address string, event db.TaxableTransaction) AccointingRow {
	row := &AccointingRow{}
	err := row.ParseBasic(address, event)
	if err != nil {
		log.Printf("Error with ParseMsgWithdrawDelegatorReward. Err: %v", err)
	}
	row.Classification = Staked
	return *row
}

// ParseMsgSend:
// If the address we searched is the receiver, then this transaction is a deposit.
// If the address we searched is the sender, then this transaction is a withdrawal.
func ParseMsgSend(address string, event db.TaxableTransaction) AccointingRow {
	row := &AccointingRow{}
	err := row.ParseBasic(address, event)
	if err != nil {
		log.Printf("Error with ParseMsgSend. Err: %v", err)
	}
	return *row
}

func ParseMsgMultiSend(address string, event db.TaxableTransaction) AccointingRow {
	row := &AccointingRow{}
	err := row.ParseBasic(address, event)
	if err != nil {
		log.Printf("Error with ParseMsgMultiSend. Err: %v", err)
	}
	return *row
}

func ParseMsgFundCommunityPool(address string, event db.TaxableTransaction) AccointingRow {
	row := &AccointingRow{}
	err := row.ParseBasic(address, event)
	if err != nil {
		log.Printf("Error with ParseMsgFundCommunityPool. Err: %v", err)
	}
	return *row
}

func ParseMsgSwapExactAmountIn(address string, event db.TaxableTransaction) AccointingRow {
	row := &AccointingRow{}
	err := row.ParseSwap(address, event)
	if err != nil {
		log.Printf("Error with ParseMsgSwapExactAmountIn. Err: %v", err)
	}
	return *row
}

func ParseMsgSwapExactAmountOut(address string, event db.TaxableTransaction) AccointingRow {
	row := &AccointingRow{}
	err := row.ParseSwap(address, event)
	if err != nil {
		log.Printf("Error with ParseMsgSwapExactAmountOut. Err: %v", err)
	}
	return *row
}

func ParseOsmosisReward(address string, event db.TaxableEvent) (AccointingRow, error) {
	row := &AccointingRow{}
	err := row.EventParseBasic(address, event)
	if err != nil {
		log.Printf("Error with ParseOsmosisReward. Err: %v", err)
	}
	return *row, err
}
