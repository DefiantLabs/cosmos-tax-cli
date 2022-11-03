package accointing

import (
	"fmt"
	
	"github.com/DefiantLabs/cosmos-tax-cli-private/csv/parsers"
	"github.com/DefiantLabs/cosmos-tax-cli-private/db"
	"github.com/DefiantLabs/cosmos-tax-cli-private/osmosis/modules/gamm"
	"github.com/DefiantLabs/cosmos-tax-cli-private/util"

	"github.com/preichenberger/go-coinbasepro/v2"
	"github.com/shopspring/decimal"

	"time"
)

var IsOsmosisJoin = map[string]bool{
	gamm.MsgJoinSwapExternAmountIn: true,
	gamm.MsgJoinSwapShareAmountOut: true,
	gamm.MsgJoinPool:               true,
}

var IsOsmosisExit = map[string]bool{
	gamm.MsgExitSwapShareAmountIn:   true,
	gamm.MsgExitSwapExternAmountOut: true,
	gamm.MsgExitPool:                true,
}

// Guard for adding messages to the group
var IsOsmosisLpTxGroup = make(map[string]bool)

func init() {
	for messageType := range IsOsmosisJoin {
		IsOsmosisLpTxGroup[messageType] = true
	}

	for messageType := range IsOsmosisExit {
		IsOsmosisLpTxGroup[messageType] = true
	}
}

type WrapperLpTxGroup struct {
	GroupedTxes map[uint][]db.TaxableTransaction //TX db ID to its messages
	Rows        []parsers.CsvRow
}

func (sf *WrapperLpTxGroup) GetRowsForParsingGroup() []parsers.CsvRow {
	return sf.Rows
}

func (sf *WrapperLpTxGroup) BelongsToGroup(message db.TaxableTransaction) bool {
	_, isInGroup := IsOsmosisLpTxGroup[message.Message.MessageType.MessageType]
	return isInGroup
}

func (sf *WrapperLpTxGroup) String() string {
	return "OsmosisLpTxGroup"
}

func (sf *WrapperLpTxGroup) GetGroupedTxes() map[uint][]db.TaxableTransaction {
	return sf.GroupedTxes
}

func (sf *WrapperLpTxGroup) AddTxToGroup(tx db.TaxableTransaction) {
	//Add tx to group using the TX ID as key and appending to array
	if _, ok := sf.GroupedTxes[tx.Message.Tx.ID]; ok {
		sf.GroupedTxes[tx.Message.Tx.ID] = append(sf.GroupedTxes[tx.Message.Tx.ID], tx)
	} else {
		var txGrouping []db.TaxableTransaction
		txGrouping = append(txGrouping, tx)
		sf.GroupedTxes[tx.Message.Tx.ID] = txGrouping
	}
}

func getRate(cbClient *coinbasepro.Client, coin string, transactionTime time.Time) (float64, error) {
	histRate, err := cbClient.GetHistoricRates(fmt.Sprintf("%v-USD", coin), coinbasepro.GetHistoricRatesParams{
		Start:       transactionTime.Add(-1 * time.Minute),
		End:         transactionTime,
		Granularity: 60,
	})
	if err != nil {
		return 0.0, fmt.Errorf("unable to get price for coin '%v' at time '%v'. Err: %v", coin, transactionTime, err)
	}
	if len(histRate) == 0 {
		return 0.0, fmt.Errorf("unable to get price for coin '%v' at time '%v'", coin, transactionTime)
	}

	return histRate[0].Close, nil
}

func (sf *WrapperLpTxGroup) ParseGroup() error {
	//TODO: Do specialized processing on LP messages
	cbClient := coinbasepro.NewClient()
	for _, txMessages := range sf.GroupedTxes {
		for _, message := range txMessages {
			row := Row{}
			row.TransactionType = Order
			row.OperationID = message.Message.Tx.Hash
			row.Date = message.Message.Tx.TimeStamp.Format(timeLayout)

			denomRecieved := message.DenominationReceived
			valueRecieved := message.AmountReceived
			conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(valueRecieved), denomRecieved)
			if err != nil {
				row.InBuyAmount = util.NumericToString(valueRecieved)
				row.InBuyAsset = denomRecieved.Base
			} else {
				row.InBuyAmount = conversionAmount.Text('f', -1)
				row.InBuyAsset = conversionSymbol
			}

			denomSent := message.DenominationSent
			valueSent := message.AmountSent
			conversionAmount, conversionSymbol, err = db.ConvertUnits(util.FromNumeric(valueSent), denomSent)
			if err != nil {
				row.OutSellAmount = util.NumericToString(valueSent)
				row.OutSellAsset = denomSent.Base
			} else {
				row.OutSellAmount = conversionAmount.Text('f', -1)
				row.OutSellAsset = conversionSymbol
			}

			//We deliberately exclude the GAMM tokens from OutSell/InBuy for Exits/Joins respectively
			//Accointing has no way of using the GAMM token to determine LP cost basis etc...
			if _, ok := IsOsmosisExit[message.Message.MessageType.MessageType]; ok {
				// add the value of gam tokens
				price, err := getRate(cbClient, message.DenominationReceived.Symbol, message.Message.Tx.TimeStamp)
				if err != nil {
					row.Comments = fmt.Sprintf("could not lookup value of %v %v. It will be equivolent to %v %v at %v.", message.AmountReceived, message.DenominationReceived.Base, message.AmountReceived, message.DenominationReceived.Symbol, row.Date)
				} else {
					gamValue := message.AmountReceived.Mul(decimal.NewFromFloat(price))
					row.Comments = fmt.Sprintf("%v %v on %v was $%v USD", message.AmountSent, message.DenominationSent.Base, row.Date, gamValue)
				}
			} else if _, ok := IsOsmosisJoin[message.Message.MessageType.MessageType]; ok {
				// add the value of gam tokens
				price, err := getRate(cbClient, message.DenominationSent.Symbol, message.Message.Tx.TimeStamp)
				if err != nil {
					row.Comments = fmt.Sprintf("could not lookup value of %v %v. It will be equivolent to %v %v at %v.", message.AmountReceived, message.DenominationReceived.Base, message.AmountSent, message.DenominationSent.Symbol, row.Date)
				} else {
					gamValue := message.AmountSent.Mul(decimal.NewFromFloat(price))
					row.Comments = fmt.Sprintf("%v %v on %v was $%v USD", message.AmountReceived, message.DenominationReceived.Base, row.Date, gamValue)
				}
			}
			sf.Rows = append(sf.Rows, row)
		}
	}
	return nil
}

func GetOsmosisTxParsingGroups() []parsers.ParsingGroup {
	var messageGroups []parsers.ParsingGroup

	//This appending of parsing groups establishes a precedence
	//There is a break statement in the loop doing grouping
	//Which means parsers further up the array will be preferred
	LpTxGroup := WrapperLpTxGroup{}
	LpTxGroup.GroupedTxes = make(map[uint][]db.TaxableTransaction)
	messageGroups = append(messageGroups, &LpTxGroup)

	return messageGroups
}
