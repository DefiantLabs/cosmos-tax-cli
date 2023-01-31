package cryptotaxcalculator

import (
	"github.com/DefiantLabs/cosmos-tax-cli/csv/parsers"
	"github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/util"
)

func ParseGroup(sf *parsers.WrapperLpTxGroup) error {
	for _, txMessages := range sf.GroupedTxes {
		for _, message := range txMessages {
			depositRow := Row{}
			depositRow.Type = FlatDeposit
			depositRow.ID = message.Message.Tx.Hash
			depositRow.Date = message.Message.Tx.Block.TimeStamp

			denomRecieved := message.DenominationReceived
			valueRecieved := message.AmountReceived
			conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(valueRecieved), denomRecieved)
			if err != nil {
				depositRow.BaseAmount = util.NumericToString(valueRecieved)
				depositRow.BaseCurrency = denomRecieved.Base
			} else {
				depositRow.BaseAmount = conversionAmount.Text('f', -1)
				depositRow.BaseCurrency = conversionSymbol
			}

			sf.Rows = append(sf.Rows, depositRow)

			withdrawalRow := Row{}
			withdrawalRow.Type = FlatWithdrawal
			withdrawalRow.ID = message.Message.Tx.Hash
			withdrawalRow.Date = message.Message.Tx.Block.TimeStamp

			denomSent := message.DenominationSent
			valueSent := message.AmountSent
			conversionAmount, conversionSymbol, err = db.ConvertUnits(util.FromNumeric(valueSent), denomSent)
			if err != nil {
				withdrawalRow.BaseAmount = util.NumericToString(valueSent)
				withdrawalRow.BaseCurrency = denomSent.Base
			} else {
				withdrawalRow.BaseAmount = conversionAmount.Text('f', -1)
				withdrawalRow.BaseCurrency = conversionSymbol
			}

			sf.Rows = append(sf.Rows, withdrawalRow)
		}
	}
	return nil
}
