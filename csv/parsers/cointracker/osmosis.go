package cointracker

import (
	"strings"

	"github.com/DefiantLabs/cosmos-tax-cli/csv/parsers"
	"github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/concentratedliquidity"
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
	sf.GroupedTxes = parsers.AddTxToGroupMap(sf.GroupedTxes, tx)
}

func (sf *OsmosisLpTxGroup) ParseGroup() error {
	txsToFees := parsers.GetTxToFeesMap(sf.GroupedTxes)
	for _, txMessages := range sf.GroupedTxes {
		for _, message := range txMessages {
			row := Row{}
			row.Date = message.Message.Tx.Block.TimeStamp.Format(TimeLayout)

			denomRecieved := message.DenominationReceived
			valueRecieved := message.AmountReceived

			if !strings.Contains(denomRecieved.Base, "gamm") {
				conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(valueRecieved), denomRecieved)
				if err != nil {
					row.ReceivedAmount = util.NumericToString(valueRecieved)
					row.ReceivedCurrency = denomRecieved.Base
				} else {
					row.ReceivedAmount = conversionAmount.Text('f', -1)
					row.ReceivedCurrency = conversionSymbol
				}
			}

			denomSent := message.DenominationSent
			valueSent := message.AmountSent

			if !strings.Contains(denomSent.Base, "gamm") {
				conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(valueSent), denomSent)
				if err != nil {
					row.SentAmount = util.NumericToString(valueSent)
					row.SentCurrency = denomSent.Base
				} else {
					row.SentAmount = conversionAmount.Text('f', -1)
					row.SentCurrency = conversionSymbol
				}
			}

			messageFee := txsToFees[message.Message.Tx.ID]
			if len(messageFee) > 0 {
				fee := messageFee[0]
				parseAndAddFeeWithDefault(&row, fee)

				// This fee has been processed, pop it off the stack
				txsToFees[message.Message.Tx.ID] = txsToFees[message.Message.Tx.ID][1:]

			}

			sf.Rows = append(sf.Rows, row)
		}
	}

	// If there are any fees left over, add them to the CSV
	for _, fees := range txsToFees {
		for _, fee := range fees {
			row := Row{}
			err := row.ParseFee(fee.Tx, fee)
			if err != nil {
				return err
			}
			sf.Rows = append(sf.Rows, row)
		}
	}

	return nil
}

type OsmosisConcentratedLiquidityTxGroup struct {
	GroupedTxes map[uint][]db.TaxableTransaction // TX db ID to its messages
	Rows        []parsers.CsvRow
}

func (sf *OsmosisConcentratedLiquidityTxGroup) GetRowsForParsingGroup() []parsers.CsvRow {
	return sf.Rows
}

func (sf *OsmosisConcentratedLiquidityTxGroup) BelongsToGroup(message db.TaxableTransaction) bool {
	_, isInGroup := parsers.IsOsmosisConcentratedLiquidity[message.Message.MessageType.MessageType]
	return isInGroup
}

func (sf *OsmosisConcentratedLiquidityTxGroup) GetGroupedTxes() map[uint][]db.TaxableTransaction {
	return sf.GroupedTxes
}

func (sf *OsmosisConcentratedLiquidityTxGroup) String() string {
	return "OsmosisLpTxGroup"
}

func (sf *OsmosisConcentratedLiquidityTxGroup) AddTxToGroup(tx db.TaxableTransaction) {
	// Add tx to group using the TX ID as key and appending to array
	if sf.GroupedTxes == nil {
		sf.GroupedTxes = make(map[uint][]db.TaxableTransaction)
	}
	sf.GroupedTxes = parsers.AddTxToGroupMap(sf.GroupedTxes, tx)
}

// Concentrated liquidit txs are grouped to be parsed together. Complex analysis may be require later, so group them now for later extension.
func (sf *OsmosisConcentratedLiquidityTxGroup) ParseGroup() error {
	txsToFees := parsers.GetTxToFeesMap(sf.GroupedTxes)
	for _, txMessages := range sf.GroupedTxes {
		for _, message := range txMessages {

			row := Row{}
			row.Date = message.Message.Tx.Block.TimeStamp.Format(TimeLayout)

			switch message.Message.MessageType.MessageType {
			case concentratedliquidity.MsgCreatePosition:
				parseAndAddSentAmountWithDefault(&row, message)
			case concentratedliquidity.MsgWithdrawPosition, concentratedliquidity.MsgTransferPositions:
				parseAndAddReceivedAmountWithDefault(&row, message)
			case concentratedliquidity.MsgAddToPosition:
				if message.DenominationReceivedID != nil {
					parseAndAddReceivedAmountWithDefault(&row, message)
				} else {
					parseAndAddSentAmountWithDefault(&row, message)
				}
			}

			messageFee := txsToFees[message.Message.Tx.ID]
			if len(messageFee) > 0 {
				fee := messageFee[0]
				parseAndAddFeeWithDefault(&row, fee)

				// This fee has been processed, pop it off the stack
				txsToFees[message.Message.Tx.ID] = txsToFees[message.Message.Tx.ID][1:]

			}
			sf.Rows = append(sf.Rows, row)
		}
	}

	// If there are any fees left over, add them to the CSV
	for _, fees := range txsToFees {
		for _, fee := range fees {
			row := Row{}
			err := row.ParseFee(fee.Tx, fee)
			if err != nil {
				return err
			}
			sf.Rows = append(sf.Rows, row)
		}
	}
	return nil
}
