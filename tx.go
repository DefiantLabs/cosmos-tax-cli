package main

import (
	"fmt"
	"sync"
	"time"
)

func ProcessTxs(responseTxs []TxStruct, responseTxResponses []TxResponseStruct) ([]Tx, [][]Address) {
	var currTxs = make([]Tx, len(responseTxs))
	var currTxsAddresses = make([][]Address, len(responseTxs))
	wg := sync.WaitGroup{}

	for i, v := range responseTxs {
		currTxResponse := responseTxResponses[i]
		wg.Add(1)

		go func(index int, tx TxStruct, txResponse TxResponseStruct) {
			defer wg.Done()
			//tx data and tx_response data are split into 2 arrays in the json, combine into 1 using the corresponding index
			var currTx MergedTx

			currTx.TxResponse = currTxResponse
			currTx.Tx = tx

			currTxs[index] = ProcessTx(currTx)
			txAddresses := ExtractTransactionAddresses(currTx)
			var currAddresses = make([]Address, len(txAddresses))
			for ii, vv := range txAddresses {
				currAddresses[ii] = Address{Address: vv}
			}
			currTxsAddresses[index] = currAddresses

		}(i, v, currTxResponse)

	}

	wg.Wait()
	return currTxs, currTxsAddresses
}

func ProcessTx(tx MergedTx) Tx {
	timeStamp, _ := time.Parse(time.RFC3339, tx.TxResponse.TimeStamp)

	fees := ProcessFees(tx.Tx.AuthInfo.TxFee.TxFeeAmount)
	code := tx.TxResponse.Code
	return Tx{TimeStamp: timeStamp, Hash: tx.TxResponse.TxHash, Fees: fees, Code: code}
}

//ProcessFees returns a comma delimited list of fee amount/denoms
func ProcessFees(txFees []TxFeeAmount) string {

	//can be multiple fees, make comma delimited list of fees
	//should consider separate table?
	fees := ""

	numFees := len(txFees)
	for i, fee := range txFees {
		newFee := fmt.Sprintf("%s%s", fee.Amount, fee.Denom)
		if i+1 != numFees {
			newFee = newFee + ","
		}
		fees = fees + newFee
	}

	return fees

}
