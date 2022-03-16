package csv

import (
	"cosmos-exporter/cosmos/modules/bank"
	"cosmos-exporter/cosmos/modules/staking"
	"cosmos-exporter/db"
	"errors"
	"strings"

	stdTypes "github.com/cosmos/cosmos-sdk/types"
	"gorm.io/gorm"
)

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
)

func (ac AccointingClassification) String() string {
	//Note that "None" returns empty string since we're using this for CSV parsing.
	//Accointing considers 'Classification' an optional field, so empty is a valid value.
	return [...]string{"", "staked", "airdrop", "payment", "fee"}[ac]
}

type AccointingRow struct {
	Date            string
	InBuyAmount     float64
	InBuyAsset      string
	OutSellAmount   float64
	OutSellAsset    string
	FeeAmount       float64
	FeeAsset        string
	Classification  AccointingClassification
	TransactionType AccointingTransaction
	OperationId     string
}

//ParseBasic: Handles the fields that are shared between most types.
func (row *AccointingRow) ParseBasic(address string, event db.TaxableEvent) {
	row.Date = FormatDatetime(event.Message.Tx.TimeStamp)
	row.OperationId = event.Message.Tx.Hash

	//deposit
	if event.ReceiverAddress.Address == address {

		conversionAmount, conversionSymbol, err := db.ConvertUnits(int64(event.Amount), event.Denomination)
		if err == nil {
			row.InBuyAmount = conversionAmount
			row.InBuyAsset = conversionSymbol
		} else {
			row.InBuyAmount = event.Amount
			row.InBuyAsset = event.Denomination
		}
		row.TransactionType = Deposit

	} else if event.SenderAddress.Address == address { //withdrawal

		conversionAmount, conversionSymbol, err := db.ConvertUnits(int64(event.Amount), event.Denomination)
		if err == nil {
			row.OutSellAmount = conversionAmount
			row.OutSellAsset = conversionSymbol
		} else {
			row.OutSellAmount = event.Amount
			row.OutSellAsset = event.Denomination
		}
		row.TransactionType = Withdraw
	}
}

func ParseForAddress(address string, pgSql *gorm.DB) ([]AccointingRow, error) {
	events, err := db.GetTaxableEvents(address, pgSql)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return nil, errors.New("no events for the given address")
	}

	rows := []AccointingRow{}
	txMap := map[uint][]db.TaxableEvent{} //Map transaction ID to List of events

	//Build a map so we know which TX go with which messages
	for _, event := range events {
		if list, ok := txMap[event.Message.Tx.ID]; ok {
			list = append(list, event)
			txMap[event.Message.Tx.ID] = list
		} else {
			txMap[event.Message.Tx.ID] = []db.TaxableEvent{event}
		}
	}

	//Parse all the potentially taxable events (one transaction group at a time)
	for _, evt := range txMap {
		//For the current transaction group, generate the rows for the CSV.
		//Usually (but not always) a transaction will only have a single row in the CSV.
		rows = append(rows, ParseTx(address, evt, pgSql)...)
	}

	return rows, nil
}

//HandleFees:
//If the transaction lists the same amount of fees as there are rows in the CSV,
//then we spread the fees out one per row. Otherwise we add a line for the fees,
//where each fee has a separate line.
func HandleFees(address string, events []db.TaxableEvent, rows []AccointingRow) []AccointingRow {
	//This address didn't pay any fees
	if len(events) == 0 || events[0].Message.Tx.SignerAddress.Address != address {
		return rows
	}

	fees := strings.Split(events[0].Message.Tx.Fees, ",")
	feeCoins := []stdTypes.Coin{}

	for _, fee := range fees {
		coin, err := stdTypes.ParseCoinNormalized(fee)
		if err == nil {
			feeCoins = append(feeCoins, coin)
		}
	}

	//Stick the fees in the existing rows.
	if len(rows) >= len(feeCoins) {
		for i, fee := range feeCoins {
			currentRow := rows[i]
			currentRow.FeeAmount = fee.Amount.ToDec().MustFloat64()
			currentRow.FeeAsset = fee.GetDenom()
		}

		return rows
	}

	tx := events[0].Message.Tx
	//There's more fees than rows so generate a new row for each fee.
	for _, fee := range feeCoins {
		newRow := AccointingRow{Date: FormatDatetime(tx.TimeStamp), FeeAmount: fee.Amount.ToDec().MustFloat64(),
			FeeAsset: fee.GetDenom(), Classification: Fee, TransactionType: Withdraw}
		rows = append(rows, newRow)
	}

	return rows
}

//ParseTx: Parse the potentially taxable event
func ParseTx(address string, events []db.TaxableEvent, pgSql *gorm.DB) []AccointingRow {
	rows := []AccointingRow{}

	for _, event := range events {
		//Is this a MsgSend
		if bank.IsMsgSend[event.Message.MessageType] {
			rows = append(rows, ParseMsgSend(address, event))
		} else if staking.IsMsgWithdrawValidatorCommission[event.Message.MessageType] {
			rows = append(rows, ParseMsgWithdrawValidatorCommission(address, event))
		} else if staking.IsMsgWithdrawDelegatorReward[event.Message.MessageType] {
			rows = append(rows, ParseMsgWithdrawDelegatorReward(address, event))
		}
	}

	rows = HandleFees(address, events, rows)
	return rows
}

//ParseMsgValidatorWithdraw:
//This transaction is always a withdrawal.
func ParseMsgWithdrawValidatorCommission(address string, event db.TaxableEvent) AccointingRow {
	row := &AccointingRow{}
	row.ParseBasic(address, event)
	row.Classification = Staked
	return *row
}

//ParseMsgValidatorWithdraw:
//This transaction is always a withdrawal.
func ParseMsgWithdrawDelegatorReward(address string, event db.TaxableEvent) AccointingRow {
	row := &AccointingRow{}
	row.ParseBasic(address, event)
	row.Classification = Staked
	return *row
}

//ParseMsgSend:
//If the address we searched is the receiver, then this transaction is a deposit.
//If the address we searched is the sender, then this transaction is a withdrawal.
func ParseMsgSend(address string, event db.TaxableEvent) AccointingRow {
	row := &AccointingRow{}
	row.ParseBasic(address, event)
	return *row
}
