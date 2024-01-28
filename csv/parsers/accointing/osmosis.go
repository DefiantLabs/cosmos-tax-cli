package accointing

import (
	"fmt"
	"strconv"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/csv/parsers"
	"github.com/DefiantLabs/cosmos-tax-cli/db"

	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/concentratedliquidity"
	"github.com/preichenberger/go-coinbasepro/v2"
)

func addTxToGroupMap(groupedTxs map[uint][]db.TaxableTransaction, tx db.TaxableTransaction) map[uint][]db.TaxableTransaction {
	// Add tx to group using the TX ID as key and appending to array
	if _, ok := groupedTxs[tx.Message.Tx.ID]; ok {
		groupedTxs[tx.Message.Tx.ID] = append(groupedTxs[tx.Message.Tx.ID], tx)
	} else {
		var txGrouping []db.TaxableTransaction
		txGrouping = append(txGrouping, tx)
		groupedTxs[tx.Message.Tx.ID] = txGrouping
	}
	return groupedTxs
}

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
	// Add tx to group using the TX ID as key and appending to array
	if sf.GroupedTxes == nil {
		sf.GroupedTxes = make(map[uint][]db.TaxableTransaction)
	}
	sf.GroupedTxes = addTxToGroupMap(sf.GroupedTxes, tx)
}

func (sf *OsmosisLpTxGroup) ParseGroup() error {
	cbClient := coinbasepro.NewClient()
	for _, txMessages := range sf.GroupedTxes {
		for _, message := range txMessages {
			row := Row{}
			row.TransactionType = Order
			row.OperationID = message.Message.Tx.Hash
			row.Date = message.Message.Tx.Block.TimeStamp.Format(TimeLayout)

			parseAndAddReceivedAmountWithDefault(&row, message)
			parseAndAddSentAmountWithDefault(&row, message)

			// We deliberately exclude the GAMM tokens from OutSell/InBuy for Exits/Joins respectively
			// Accointing has no way of using the GAMM token to determine LP cost basis etc...
			if _, ok := parsers.IsOsmosisExit[message.Message.MessageType.MessageType]; ok {
				// add the value of gam tokens
				price, err := parsers.GetRate(cbClient, message.DenominationReceived.Symbol, message.Message.Tx.Block.TimeStamp)
				if err != nil {
					row.Comments = fmt.Sprintf("could not lookup value of %v %v. It will be equivalent to %v %v at %v.", row.OutSellAmount, row.OutSellAsset, row.InBuyAmount, row.InBuyAsset, row.Date)
				} else {
					receivedAmount, err := strconv.ParseFloat(row.InBuyAmount, 64)
					if err != nil {
						config.Log.Fatal(fmt.Sprintf("Could not parse amount %v", row.InBuyAmount), err)
					}
					gamValue := receivedAmount * price
					row.Comments = fmt.Sprintf("%v %v on %v was $%v USD", row.OutSellAmount, row.OutSellAsset, row.Date, gamValue)
				}
			} else if _, ok := parsers.IsOsmosisJoin[message.Message.MessageType.MessageType]; ok {
				// add the value of gam tokens
				price, err := parsers.GetRate(cbClient, message.DenominationSent.Symbol, message.Message.Tx.Block.TimeStamp)
				if err != nil {
					row.Comments = fmt.Sprintf("could not lookup value of %v %v. It will be equivalent to %v %v at %v.", row.InBuyAmount, row.InBuyAsset, row.OutSellAmount, row.OutSellAsset, row.Date)
				} else {
					sentAmount, err := strconv.ParseFloat(row.OutSellAmount, 64)
					if err != nil {
						config.Log.Fatal(fmt.Sprintf("Could not parse amount %v", row.OutSellAmount), err)
					}
					gamValue := sentAmount * price
					row.Comments = fmt.Sprintf("%v %v on %v was $%v USD", row.InBuyAmount, row.InBuyAsset, row.Date, gamValue)
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
	sf.GroupedTxes = addTxToGroupMap(sf.GroupedTxes, tx)
}

// Concentrated liquidit txs are grouped to be parsed together. Complex analysis may be require later, so group them now for later extension.
func (sf *OsmosisConcentratedLiquidityTxGroup) ParseGroup() error {
	for _, txMessages := range sf.GroupedTxes {
		for _, message := range txMessages {
			row := Row{}
			row.OperationID = message.Message.Tx.Hash
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
			sf.Rows = append(sf.Rows, row)
		}
	}
	return nil
}
