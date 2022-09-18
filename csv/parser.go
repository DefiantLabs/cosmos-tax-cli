package csv

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/bank"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/staking"
	"github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/gamm"
	"github.com/DefiantLabs/cosmos-tax-cli/util"

	"gorm.io/gorm"
)

//Accointing CSV Explainer found here, contains info on transaction types and classifications:
//https://support.accointing.com/hc/en-us/articles/4423524486669-Uploading-data-via-the-ACCOINTING-com-CSV-XLSX-Template
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

//Interface for all TX parsing groups
type TxParsingGroup interface {
	BelongsToGroup(db.TaxableTransaction) bool
	String() string
	AddTxToGroup(db.TaxableTransaction)
	GetGroupedTxes() map[uint][]db.TaxableTransaction
	ParseGroup() ([]AccointingRow, error)
}

//Initial parsing groups is empty for now as we
//dont currently have a use-case beyond the osmosis LP groups
var txParsingGroups []TxParsingGroup

func BootstrapChainSpecificTxParsingGroups(chainId string) {

	switch chainId {
	case "osmosis-1":
		for _, v := range GetOsmosisTxParsingGroups() {
			txParsingGroups = append(txParsingGroups, v)
		}
	}
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
}

//ParseBasic: Handles the fields that are shared between most types.
func (row *AccointingRow) EventParseBasic(address string, event db.TaxableEvent) error {
	//row.Date = FormatDatetime(event.Message.Tx.TimeStamp) TODO, FML, I forgot to add a DB field for this. Ideally it should come from the block time.
	//row.OperationId = ??? TODO - maybe use the block hash or something. This isn't a TX so there is no TX hash. Have to test Accointing response to using block hash.

	//deposit
	if event.EventAddress.Address == address {
		conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(event.Amount), event.Denomination)
		if err == nil {
			row.InBuyAmount = conversionAmount.String()
			row.InBuyAsset = conversionSymbol
		} else {
			row.InBuyAmount = util.NumericToString(event.Amount)
			row.InBuyAsset = event.Denomination.Base
		}
		row.TransactionType = Deposit
		return nil
	}

	return errors.New("unknown TaxableEvent with ID " + strconv.FormatUint(uint64(event.ID), 10))
}

//ParseBasic: Handles the fields that are shared between most types.
func (row *AccointingRow) ParseBasic(address string, event db.TaxableTransaction) error {
	row.Date = FormatDatetime(event.Message.Tx.TimeStamp)
	row.OperationId = event.Message.Tx.Hash

	//deposit
	if event.ReceiverAddress.Address == address {
		conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(event.AmountReceived), event.DenominationReceived)
		if err == nil {
			row.InBuyAmount = conversionAmount.String()
			row.InBuyAsset = conversionSymbol
		} else {
			return fmt.Errorf("Cannot parse denom units for TX %s (classification: deposit)\n", row.OperationId)
		}
		row.TransactionType = Deposit

	} else if event.SenderAddress.Address == address { //withdrawal

		conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(event.AmountSent), event.DenominationSent)
		if err == nil {
			row.OutSellAmount = conversionAmount.String()
			row.OutSellAsset = conversionSymbol
		} else {
			return fmt.Errorf("Cannot parse denom units for TX %s (classification: withdrawal)\n", row.OperationId)
		}
		row.TransactionType = Withdraw
	}

	return nil
}

func (row *AccointingRow) ParseSwap(address string, event db.TaxableTransaction) error {
	row.Date = FormatDatetime(event.Message.Tx.TimeStamp)
	row.OperationId = event.Message.Tx.Hash
	row.TransactionType = Order

	recievedConversionAmount, recievedConversionSymbol, err := db.ConvertUnits(util.FromNumeric(event.AmountReceived), event.DenominationReceived)
	if err == nil {
		row.InBuyAmount = recievedConversionAmount.String()
		row.InBuyAsset = recievedConversionSymbol
	} else {
		return fmt.Errorf("Cannot parse denom units for TX %s (classification: swap received)\n", row.OperationId)
	}

	sentConversionAmount, sentConversionSymbol, err := db.ConvertUnits(util.FromNumeric(event.AmountSent), event.DenominationSent)
	if err == nil {
		row.OutSellAmount = sentConversionAmount.String()
		row.OutSellAsset = sentConversionSymbol
	} else {
		return fmt.Errorf("Cannot parse denom units for TX %s (classification: swap sent)\n", row.OperationId)
	}

	return nil
}

func ParseTaxableEvents(address string, pgSql *gorm.DB) ([]AccointingRow, error) {
	rows := []AccointingRow{}

	taxableEvents, err := db.GetTaxableEvents(address, pgSql)
	if err != nil {
		return nil, err
	}

	if len(taxableEvents) == 0 {
		return rows, nil
	}

	//Parse all the potentially taxable events
	for _, event := range taxableEvents {
		//generate the rows for the CSV.
		rows = append(rows, ParseEvent(address, event)...)
	}

	return rows, nil
}

func ParseTaxableTransactions(address string, pgSql *gorm.DB) ([]AccointingRow, error) {
	taxableTxs, err := db.GetTaxableTransactions(address, pgSql)
	if err != nil {
		return nil, err
	}

	if len(taxableTxs) == 0 {
		return nil, errors.New("no events for the given address")
	}

	rows := []AccointingRow{}
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
			for groupIndex, txGroup := range txParsingGroups {
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
			var currentGroup TxParsingGroup = txParsingGroups[groupIndex]

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
				rows = append(rows, txRows...)
			} else {
				return nil, err
			}
		}
	}

	//Parse all the txes found in the Parsing Groups
	for _, txParsingGroup := range txParsingGroups {

		txRows, err := txParsingGroup.ParseGroup()
		if err == nil {
			rows = append(rows, txRows...)
		} else {
			return nil, err
		}
	}

	return rows, nil
}

func ParseForAddress(address string, pgSql *gorm.DB) ([]AccointingRow, error) {
	allRows := []AccointingRow{}
	rows, err := ParseTaxableTransactions(address, pgSql)
	if err != nil {
		//TODO
		//We need to HANDLE the error in a way that notifies end users (of the website) that the CSV download failed.
		//For now we just kill the program (that way I can't forget TODO this)
		fmt.Println(err)
		os.Exit(1)
	}

	allRows = append(allRows, rows...)

	//For now this gets all taxable events, which is only Osmosis rewards. Later we'll need to update the query to only grab Osmosis events.
	//Reason being, presumably users will have an option to select or ignore certain chains.
	rows, err = ParseTaxableEvents(address, pgSql)
	if err != nil {
		//TODO
		//We need to HANDLE the error in a way that notifies end users (of the website) that the CSV download failed.
		//For now we just kill the program (that way I can't forget TODO this)
		fmt.Println(err)
		os.Exit(1)
	}

	allRows = append(allRows, rows...)
	return allRows, err
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
func ParseTx(address string, events []db.TaxableTransaction) ([]AccointingRow, error) {
	rows := []AccointingRow{}

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

	rows, err := HandleFees(address, events, rows)
	return rows, err
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
