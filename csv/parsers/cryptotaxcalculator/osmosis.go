package cryptotaxcalculator

import (
	"github.com/DefiantLabs/cosmos-tax-cli/csv/parsers"
	"github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/gamm"
	"github.com/DefiantLabs/cosmos-tax-cli/util"
)

func ParseGroup(sf *parsers.WrapperLpTxGroup) error {
	for _, txMessages := range sf.GroupedTxes {
		for _, message := range txMessages {
			row := Row{}
			row.ID = message.Message.Tx.Hash
			row.Date = message.Message.Tx.Block.TimeStamp.Format(TimeLayout)

			if message.Message.MessageType.MessageType == gamm.MsgJoinSwapExternAmountIn ||
				message.Message.MessageType.MessageType == gamm.MsgExitSwapShareAmountIn ||
				message.Message.MessageType.MessageType == gamm.MsgJoinPool {
				row.Type = Buy
			} else {
				row.Type = Sell
			}

			denomRecieved := message.DenominationReceived
			valueRecieved := message.AmountReceived
			conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(valueRecieved), denomRecieved)
			if err != nil {
				row.BaseAmount = util.NumericToString(valueRecieved)
				row.BaseCurrency = denomRecieved.Base
			} else {
				row.BaseAmount = conversionAmount.Text('f', -1)
				row.BaseCurrency = conversionSymbol
			}

			denomSent := message.DenominationSent
			valueSent := message.AmountSent
			conversionAmount, conversionSymbol, err = db.ConvertUnits(util.FromNumeric(valueSent), denomSent)
			if err != nil {
				row.QuoteAmount = util.NumericToString(valueSent)
				row.QuoteCurrency = denomSent.Base
			} else {
				row.QuoteAmount = conversionAmount.Text('f', -1)
				row.QuoteCurrency = conversionSymbol
			}

			sf.Rows = append(sf.Rows, row)
		}
	}
	return nil
}
