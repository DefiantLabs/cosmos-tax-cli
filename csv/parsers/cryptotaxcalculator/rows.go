package cryptotaxcalculator

import (
	"errors"
	"fmt"

	"github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/util"
)

func (row Row) GetRowForCsv() []string {
	return []string{
		row.Date,
		row.Type,
		row.BaseCurrency,
		row.BaseAmount,
		row.QuoteCurrency,
		row.QuoteAmount,
		row.FeeCurrency,
		row.FeeAmount,
		row.From,
		row.To,
		row.Blockchain,
		row.ID,
		row.Description,
		row.ReferencePricePerUnit,
		row.ReferencePriceCurrency,
	}
}

func (row Row) GetDate() string {
	return row.Date
}

func (row *Row) EventParseBasic(event db.TaxableEvent) error {
	row.Date = event.Block.TimeStamp.Format(TimeLayout)

	conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(event.Amount), event.Denomination)
	if err == nil {
		row.BaseAmount = conversionAmount.Text('f', -1)
		row.BaseCurrency = conversionSymbol
	} else {
		row.BaseAmount = util.NumericToString(event.Amount)
		row.BaseCurrency = event.Denomination.Base
	}

	return nil
}

// ParseBasic: Handles the fields that are shared between most types.
func (row *Row) ParseBasic(address string, event db.TaxableTransaction) error {
	row.Date = event.Message.Tx.Block.TimeStamp.Format(TimeLayout)
	row.ID = event.Message.Tx.Hash

	// deposit
	if event.ReceiverAddress.Address == address {
		conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(event.AmountReceived), event.DenominationReceived)
		if err != nil {
			return fmt.Errorf("cannot parse denom units for TX %s (classification: deposit)", row.ID)
		}
		row.BaseAmount = conversionAmount.Text('f', -1)
		row.BaseCurrency = conversionSymbol
		row.Type = FlatDeposit
	} else if event.SenderAddress.Address == address { // withdrawal
		conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(event.AmountSent), event.DenominationSent)
		if err != nil {
			return fmt.Errorf("cannot parse denom units for TX %s (classification: withdrawal)", row.ID)
		}
		row.BaseAmount = conversionAmount.Text('f', -1)
		row.BaseCurrency = conversionSymbol
		row.Type = FlatWithdrawal
	}

	row.From = event.SenderAddress.Address
	row.To = event.ReceiverAddress.Address
	for _, fee := range event.Message.Tx.Fees {
		if fee.PayerAddress.Address == address {
			sentConversionAmount, sentConversionSymbol, err := db.ConvertUnits(util.FromNumeric(fee.Amount), fee.Denomination)
			if err != nil {
				return fmt.Errorf("cannot parse denom units for TX %s (classification: swap sent)", row.ID)
			}

			row.FeeAmount = sentConversionAmount.Text('f', -1)
			row.FeeCurrency = sentConversionSymbol
		}
	}

	return nil
}

func (row *Row) ParseFee(fee db.Fee) error {
	row.Date = fee.Tx.Block.TimeStamp.Format(TimeLayout)
	row.ID = fee.Tx.Hash
	row.Type = Fee

	sentConversionAmount, sentConversionSymbol, err := db.ConvertUnits(util.FromNumeric(fee.Amount), fee.Denomination)
	if err != nil {
		return fmt.Errorf("cannot parse denom units for TX %s (classification: swap sent)", row.ID)
	}

	row.BaseAmount = sentConversionAmount.Text('f', -1)
	row.BaseCurrency = sentConversionSymbol

	return nil
}

func (row *Row) ParseSwap(event db.TaxableTransaction, address, eventType string) error {
	row.Date = event.Message.Tx.Block.TimeStamp.Format(TimeLayout)
	row.ID = event.Message.Tx.Hash
	row.Type = eventType

	recievedConversionAmount, recievedConversionSymbol, err := db.ConvertUnits(util.FromNumeric(event.AmountReceived), event.DenominationReceived)
	if err != nil {
		return fmt.Errorf("cannot parse denom units for TX %s (classification: swap received)", row.ID)
	}

	row.BaseAmount = recievedConversionAmount.Text('f', -1)
	row.BaseCurrency = recievedConversionSymbol

	sentConversionAmount, sentConversionSymbol, err := db.ConvertUnits(util.FromNumeric(event.AmountSent), event.DenominationSent)
	if err != nil {
		return fmt.Errorf("cannot parse denom units for TX %s (classification: swap sent)", row.ID)
	}

	row.QuoteAmount = sentConversionAmount.Text('f', -1)
	row.QuoteCurrency = sentConversionSymbol

	for _, fee := range event.Message.Tx.Fees {
		if fee.PayerAddress.Address == address {
			sentConversionAmount, sentConversionSymbol, err := db.ConvertUnits(util.FromNumeric(fee.Amount), fee.Denomination)
			if err != nil {
				return fmt.Errorf("cannot parse denom units for TX %s (classification: swap sent)", row.ID)
			}

			row.FeeAmount = sentConversionAmount.Text('f', -1)
			row.FeeCurrency = sentConversionSymbol
		}
	}

	return nil
}

func parseAndAddSentAmount(row *Row, event db.TaxableTransaction) error {
	conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(event.AmountSent), event.DenominationSent)
	if err != nil {
		return errors.New("cannot parse denom units")
	}
	row.QuoteAmount = conversionAmount.Text('f', -1)
	row.QuoteCurrency = conversionSymbol

	return nil
}

func parseAndAddSentAmountWithDefault(row *Row, event db.TaxableTransaction) {
	err := parseAndAddSentAmount(row, event)
	if err != nil {
		row.QuoteAmount = util.NumericToString(event.AmountSent)
		row.QuoteCurrency = event.DenominationSent.Base
	}
}

func parseAndAddReceivedAmount(row *Row, event db.TaxableTransaction) error {
	conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(event.AmountReceived), event.DenominationReceived)
	if err != nil {
		return errors.New("cannot parse denom units")
	}
	row.BaseAmount = conversionAmount.Text('f', -1)
	row.BaseCurrency = conversionSymbol

	return nil
}

func parseAndAddReceivedAmountWithDefault(row *Row, event db.TaxableTransaction) {
	err := parseAndAddReceivedAmount(row, event)
	if err != nil {
		row.BaseAmount = util.NumericToString(event.AmountReceived)
		row.BaseCurrency = event.DenominationReceived.Base
	}
}

func parseAndAddFee(row *Row, fee db.Fee) error {
	conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(fee.Amount), fee.Denomination)
	if err != nil {
		return errors.New("cannot parse denom units")
	}
	row.FeeAmount = conversionAmount.Text('f', -1)
	row.FeeCurrency = conversionSymbol

	return nil
}

func parseAndAddFeeWithDefault(row *Row, fee db.Fee) {
	err := parseAndAddFee(row, fee)
	if err != nil {
		row.FeeAmount = util.NumericToString(fee.Amount)
		row.FeeCurrency = fee.Denomination.Base
	}
}
