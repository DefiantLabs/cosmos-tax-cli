package csv

import (
	"cosmos-exporter/cosmos/modules/bank"
	"cosmos-exporter/db"

	"gorm.io/gorm"
)

type AccointingRow struct {
	Date           string
	InBuyAmount    float64
	InBuyAsset     string
	OutSellAmount  float64
	OutSellAsset   float64
	FeeAmount      float64
	FeeAsset       string
	Classification string
	OperationId    string
}

func ParseForAddresses(addresses []string, pgSql *gorm.DB) ([]AccointingRow, error) {
	events, err := db.GetTaxableEvents(addresses, pgSql)
	if err != nil || len(events) == 0 {
		return nil, err
	}

	rows := []AccointingRow{}
	txMap := map[uint][]db.TaxableEvent{} //Map transaction ID to List of events

	//Build a map so we know which TX go with which messages
	for _, event := range events {
		if list, ok := txMap[event.Message.Tx.ID]; ok {
			list = append(list, event)
			txMap[event.Message.Tx.ID] = list
		} else {
			txMap[event.Message.Tx.ID] = []db.TaxableEvent{event}
		}
	}

	for _, evt := range txMap {
		rows = append(rows, ParseTx(evt)...)
	}

	return rows, nil
}

//ParseTx: Parse the potentially taxable event
func ParseTx(events []db.TaxableEvent) []AccointingRow {
	rows := []AccointingRow{}

	for _, event := range events {
		//Is this a MsgSend
		if bank.MsgSend[event.Message.MessageType] {
			rows = append(rows, ParseMsgSend(event))
		}
	}

	return rows
}

func ParseMsgSend(event db.TaxableEvent) AccointingRow {
	row := AccointingRow{}
	return row
}
