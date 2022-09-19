package accointing

import (
	"github.com/DefiantLabs/cosmos-tax-cli/csv/parsers"
	"github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/util"
)

var IsOsmosisJoin = map[string]bool{
	"/osmosis.gamm.v1beta1.MsgJoinSwapExternAmountIn": true,
	"/osmosis.gamm.v1beta1.MsgJoinSwapShareAmountOut": true,
	"/osmosis.gamm.v1beta1.MsgJoinPool":               true,
}

var IsOsmosisExit = map[string]bool{
	"/osmosis.gamm.v1beta1.MsgExitSwapShareAmountIn":   true,
	"/osmosis.gamm.v1beta1.MsgExitSwapExternAmountOut": true,
	"/osmosis.gamm.v1beta1.MsgExitPool":                true,
}

//Guard for adding messages to the group
var IsOsmosisLpTxGroup = make(map[string]bool)

func init() {
	for messageType, _ := range IsOsmosisJoin {
		IsOsmosisLpTxGroup[messageType] = true
	}

	for messageType, _ := range IsOsmosisExit {
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
	_, isInGroup := IsOsmosisLpTxGroup[message.Message.MessageType]
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

func (sf *WrapperLpTxGroup) ParseGroup() error {
	//TODO: Do specialized processing on LP messages
	for _, txMessages := range sf.GroupedTxes {
		for _, message := range txMessages {

			row := AccointingRow{}
			row.OperationId = message.Message.Tx.Hash
			row.Date = FormatDatetime(message.Message.Tx.TimeStamp)
			//We deliberately exclude the GAMM tokens from OutSell/InBuy for Exits/Joins respectively
			//Accointing has no way of using the GAMM token to determine LP cost basis etc...
			if _, ok := IsOsmosisExit[message.Message.MessageType]; ok {
				denomRecieved := message.DenominationReceived
				valueRecieved := message.AmountReceived
				conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(valueRecieved), denomRecieved)
				if err != nil {
					row.InBuyAmount = util.NumericToString(valueRecieved)
					row.InBuyAsset = denomRecieved.Base
				} else {
					row.InBuyAmount = conversionAmount.String()
					row.InBuyAsset = conversionSymbol
				}

				row.TransactionType = Deposit
				row.Classification = LiquidityPool
				sf.Rows = append(sf.Rows, row)
			} else if _, ok := IsOsmosisJoin[message.Message.MessageType]; ok {
				denomSent := message.DenominationSent
				valueSent := message.AmountSent
				conversionAmount, conversionSymbol, err := db.ConvertUnits(util.FromNumeric(valueSent), denomSent)
				if err != nil {
					row.OutSellAmount = util.NumericToString(valueSent)
					row.OutSellAsset = denomSent.Base
				} else {
					row.OutSellAmount = conversionAmount.String()
					row.OutSellAsset = conversionSymbol
				}
				row.TransactionType = Withdraw
				sf.Rows = append(sf.Rows, row)
			}
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
