package cryptotaxcalculator

import (
	"github.com/DefiantLabs/cosmos-tax-cli/csv/parsers"
	"github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/gamm"
	"github.com/DefiantLabs/cosmos-tax-cli/util"
)

type OsmosisLpTxGroup struct {
	GroupedTxes map[uint][]db.TaxableTransaction // TX db ID to its messages
	Rows        []parsers.CsvRow
}

func (sf *OsmosisLpTxGroup) GetRowsForParsingGroup() []parsers.CsvRow {
	return sf.Rows
}

func (sf *OsmosisLpTxGroup) BelongsToGroup(message db.TaxableTransaction) bool {
	_, isInGroup := parsers.IsOsmosisLpTxGroup[message.Message.MessageType.MessageType]
	return isInGroup
}

func (sf *OsmosisLpTxGroup) GetGroupedTxes() map[uint][]db.TaxableTransaction {
	return sf.GroupedTxes
}

func (sf *OsmosisLpTxGroup) String() string {
	return "OsmosisLpTxGroup"
}

func (sf *OsmosisLpTxGroup) AddTxToGroup(tx db.TaxableTransaction) {
	if sf.GroupedTxes == nil {
		sf.GroupedTxes = make(map[uint][]db.TaxableTransaction)
	}
	// Add tx to group using the TX ID as key and appending to array
	if _, ok := sf.GroupedTxes[tx.Message.Tx.ID]; ok {
		sf.GroupedTxes[tx.Message.Tx.ID] = append(sf.GroupedTxes[tx.Message.Tx.ID], tx)
	} else {
		var txGrouping []db.TaxableTransaction
		txGrouping = append(txGrouping, tx)
		sf.GroupedTxes[tx.Message.Tx.ID] = txGrouping
	}
}

func (sf *OsmosisLpTxGroup) ParseGroup() error {
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
