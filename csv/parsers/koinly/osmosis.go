package koinly

import (
	"fmt"
	"strconv"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/csv/parsers"
	"github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/concentratedliquidity"
	"github.com/DefiantLabs/cosmos-tax-cli/util"

	"github.com/preichenberger/go-coinbasepro/v2"
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
	cbClient := coinbasepro.NewClient()
	for _, txMessages := range sf.GroupedTxes {
		for _, message := range txMessages {
			row := Row{}
			row.TxHash = message.Message.Tx.Hash
			row.Date = message.Message.Tx.Block.TimeStamp.Format(TimeLayout)

			denomRecieved := message.DenominationReceived
			valueRecieved := message.AmountReceived
			conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(valueRecieved), denomRecieved)
			if err != nil {
				row.ReceivedAmount = util.NumericToString(valueRecieved)
				row.ReceivedCurrency = denomRecieved.Base
			} else {
				row.ReceivedAmount = conversionAmount.Text('f', -1)
				row.ReceivedCurrency = conversionSymbol
			}

			denomSent := message.DenominationSent
			valueSent := message.AmountSent
			conversionAmount, conversionSymbol, err = db.ConvertUnits(util.FromNumeric(valueSent), denomSent)
			if err != nil {
				row.SentAmount = util.NumericToString(valueSent)
				row.SentCurrency = denomSent.Base
			} else {
				row.SentAmount = conversionAmount.Text('f', -1)
				row.SentCurrency = conversionSymbol
			}

			// We deliberately exclude the GAMM tokens from OutSell/InBuy for Exits/Joins respectively
			// Accointing has no way of using the GAMM token to determine LP cost basis etc...
			if _, ok := parsers.IsOsmosisExit[message.Message.MessageType.MessageType]; ok {
				row.Label = LiquidityOut
				// add the value of gam tokens
				price, err := parsers.GetRate(cbClient, message.DenominationReceived.Symbol, message.Message.Tx.Block.TimeStamp)
				if err != nil {
					row.Description = fmt.Sprintf("could not lookup value of %v %v. It will be equivalent to %v %v at %v.", row.SentAmount, row.SentCurrency, row.ReceivedAmount, row.ReceivedCurrency, row.Date)
				} else {
					receivedAmount, err := strconv.ParseFloat(row.ReceivedAmount, 64)
					if err != nil {
						config.Log.Fatal(fmt.Sprintf("Could not parse amount %v", row.ReceivedAmount), err)
					}
					gamValue := receivedAmount * price
					row.Description = fmt.Sprintf("%v %v on %v was $%v USD", row.SentAmount, row.SentCurrency, row.Date, gamValue)
				}
			} else if _, ok := parsers.IsOsmosisJoin[message.Message.MessageType.MessageType]; ok {
				row.Label = LiquidityIn
				// add the value of gam tokens
				price, err := parsers.GetRate(cbClient, message.DenominationSent.Symbol, message.Message.Tx.Block.TimeStamp)
				if err != nil {
					row.Description = fmt.Sprintf("could not lookup value of %v %v. It will be equivalent to %v %v at %v.", row.ReceivedAmount, row.ReceivedCurrency, row.SentAmount, row.SentCurrency, row.Date)
				} else {
					sentAmount, err := strconv.ParseFloat(row.SentAmount, 64)
					if err != nil {
						config.Log.Fatal(fmt.Sprintf("Could not parse amount %v", row.SentAmount), err)
					}
					gamValue := sentAmount * price
					row.Description = fmt.Sprintf("%v %v on %v was $%v USD", row.ReceivedAmount, row.ReceivedCurrency, row.Date, gamValue)
				}
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
			row.TxHash = message.Message.Tx.Hash
			switch message.Message.MessageType.MessageType {
			case concentratedliquidity.MsgCreatePosition:
				parseAndAddSentAmountWithDefault(&row, message)
				row.Label = LiquidityIn
			case concentratedliquidity.MsgWithdrawPosition, concentratedliquidity.MsgTransferPositions:
				parseAndAddReceivedAmountWithDefault(&row, message)
				row.Label = LiquidityOut
			case concentratedliquidity.MsgAddToPosition:
				if message.DenominationReceivedID != nil {
					parseAndAddReceivedAmountWithDefault(&row, message)
					row.Label = LiquidityIn
				} else {
					parseAndAddSentAmountWithDefault(&row, message)
					row.Label = LiquidityOut
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
