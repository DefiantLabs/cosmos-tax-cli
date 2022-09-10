package csv

import (
	"fmt"

	"github.com/DefiantLabs/cosmos-tax-cli/db"
)

//Guard for adding messages to the group
var IsOsmosisLpTxGroup = map[string]bool{
	"/osmosis.gamm.v1beta1.MsgJoinSwapExternAmountIn":  true,
	"/osmosis.gamm.v1beta1.MsgJoinSwapShareAmountOut":  true,
	"/osmosis.gamm.v1beta1.MsgJoinPool":                true,
	"/osmosis.gamm.v1beta1.MsgExitSwapShareAmountIn":   true,
	"/osmosis.gamm.v1beta1.MsgExitSwapExternAmountOut": true,
	"/osmosis.gamm.v1beta1.MsgExitPool":                true,
}

type WrapperLpTxGroup struct {
	GroupedTxes map[uint][]db.TaxableTransaction //TX db ID to its messages
}

func (sf WrapperLpTxGroup) BelongsToGroup(msgType string) bool {
	_, isInGroup := IsOsmosisLpTxGroup[msgType]
	return isInGroup
}

func (sf WrapperLpTxGroup) String() string {
	return "OsmosisLpTxGroup"
}

func (sf WrapperLpTxGroup) GetGroupedTxes() map[uint][]db.TaxableTransaction {
	return sf.GroupedTxes
}

func (sf WrapperLpTxGroup) AddTxToGroup(tx db.TaxableTransaction) {
	//Add tx to group using the TX ID as key and appending to array
	if _, ok := sf.GroupedTxes[tx.Message.Tx.ID]; ok {
		sf.GroupedTxes[tx.Message.Tx.ID] = append(sf.GroupedTxes[tx.Message.Tx.ID], tx)
	} else {
		var txGrouping []db.TaxableTransaction
		txGrouping = append(txGrouping, tx)
		sf.GroupedTxes[tx.Message.Tx.ID] = txGrouping
	}
}

func (sf WrapperLpTxGroup) ParseGroup() ([]AccointingRow, error) {
	var rows []AccointingRow

	//TODO: Do specialized processing on LP messages
	for i, txMessages := range sf.GroupedTxes {
		fmt.Printf("TX with ID %d has %d message(s) in the parsing group\n", i, len(txMessages))
		for _, message := range txMessages {
			fmt.Println("Processing message", message.Message.MessageType)
		}
	}
	return rows, nil
}

func GetOsmosisTxParsingGroups() []TxParsingGroup {
	var messageGroups []TxParsingGroup

	//This appending of parsing groups establishes a precedence
	//There is a break statement in the loop doing grouping
	//Which means parsers further up the array will be preferred
	LpTxGroup := WrapperLpTxGroup{}
	LpTxGroup.GroupedTxes = make(map[uint][]db.TaxableTransaction)
	messageGroups = append(messageGroups, LpTxGroup)

	return messageGroups
}
