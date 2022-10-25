package accointing

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/util"
)

func (row AccointingRow) GetRowForCsv() []string {

	return []string{
		row.TransactionType.String(),
		row.Date,
		row.InBuyAmount,
		row.InBuyAsset,
		row.OutSellAmount,
		row.OutSellAsset,
		row.FeeAmount,
		row.FeeAsset,
		row.Classification.String(),
		row.OperationId,
		"",
	}
}

// ParseBasic: Handles the fields that are shared between most types.
func (row *AccointingRow) EventParseBasic(address string, event db.TaxableEvent) error {
	//row.Date = FormatDatetime(event.Message.Tx.TimeStamp) TODO, FML, I forgot to add a DB field for this. Ideally it should come from the block time.
	//row.OperationId = ??? TODO - maybe use the block hash or something. This isn't a TX so there is no TX hash. Have to test Accointing response to using block hash.

	//deposit
	if event.EventAddress.Address == address {
		conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(event.Amount), event.Denomination)
		if err == nil {
			row.InBuyAmount = conversionAmount.Text('f', -1)
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

// ParseBasic: Handles the fields that are shared between most types.
func (row *AccointingRow) ParseBasic(address string, event db.TaxableTransaction) error {
	row.Date = FormatDatetime(event.Message.Tx.TimeStamp)
	row.OperationId = event.Message.Tx.Hash

	//deposit
	if event.ReceiverAddress.Address == address {
		conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(event.AmountReceived), event.DenominationReceived)
		if err == nil {
			row.InBuyAmount = conversionAmount.Text('f', -1)
			row.InBuyAsset = conversionSymbol
		} else {
			return fmt.Errorf("Cannot parse denom units for TX %s (classification: deposit)\n", row.OperationId)
		}
		row.TransactionType = Deposit

	} else if event.SenderAddress.Address == address { //withdrawal

		conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(event.AmountSent), event.DenominationSent)
		if err == nil {
			row.OutSellAmount = conversionAmount.Text('f', -1)
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
		row.InBuyAmount = recievedConversionAmount.Text('f', -1)
		row.InBuyAsset = recievedConversionSymbol
	} else {
		return fmt.Errorf("Cannot parse denom units for TX %s (classification: swap received)\n", row.OperationId)
	}

	sentConversionAmount, sentConversionSymbol, err := db.ConvertUnits(util.FromNumeric(event.AmountSent), event.DenominationSent)
	if err == nil {
		row.OutSellAmount = sentConversionAmount.Text('f', -1)
		row.OutSellAsset = sentConversionSymbol
	} else {
		return fmt.Errorf("Cannot parse denom units for TX %s (classification: swap sent)\n", row.OperationId)
	}

	return nil
}

func (row *AccointingRow) ParseFee(tx db.Tx, fee db.Fee) error {
	row.Date = FormatDatetime(tx.TimeStamp)
	row.OperationId = tx.Hash
	row.TransactionType = Withdraw
	row.Classification = Fee
	sentConversionAmount, sentConversionSymbol, err := db.ConvertUnits(util.FromNumeric(fee.Amount), fee.Denomination)
	if err == nil {
		row.OutSellAmount = sentConversionAmount.Text('f', -1)
		row.OutSellAsset = sentConversionSymbol
	} else {
		return fmt.Errorf("Cannot parse denom units for TX %s (classification: swap sent)\n", row.OperationId)
	}

	return nil
}
