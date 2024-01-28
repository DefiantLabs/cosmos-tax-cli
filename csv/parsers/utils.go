package parsers

import (
	"fmt"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/preichenberger/go-coinbasepro/v2"
)

func GetRate(cbClient *coinbasepro.Client, coin string, transactionTime time.Time) (float64, error) {
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

func AddTxToGroupMap(groupedTxs map[uint][]db.TaxableTransaction, tx db.TaxableTransaction) map[uint][]db.TaxableTransaction {
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

func GetTxToFeesMap(groupedTxes map[uint][]db.TaxableTransaction) map[uint][]db.Fee {
	txToFees := make(map[uint][]db.Fee)

	for _, txMessages := range groupedTxes {
		for _, message := range txMessages {
			messageTx := message.Message.Tx
			if _, ok := txToFees[messageTx.ID]; !ok {
				txToFees[messageTx.ID] = append(txToFees[messageTx.ID], messageTx.Fees...)
				break
			}
		}
	}

	return txToFees
}
