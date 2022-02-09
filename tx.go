package main

import (
	"fmt"
	"sync"
	"time"
)

type TxWithAddresses struct {
	Tx        Tx
	Addresses []Address
}

func ProcessTxs(responseTxs []TxStruct, responseTxResponses []TxResponseStruct) []TxWithAddresses {
	var currTxsWithAddresses = make([]TxWithAddresses, len(responseTxs))
	wg := sync.WaitGroup{}

	for i, currTx := range responseTxs {
		currTxResponse := responseTxResponses[i]
		wg.Add(1)

		go func(index int, tx TxStruct, txResponse TxResponseStruct) {
			defer wg.Done()
			//tx data and tx_response data are split into 2 arrays in the json, combine into 1 using the corresponding index
			var mergedTx MergedTx
			mergedTx.TxResponse = currTxResponse
			mergedTx.Tx = tx

			processedTx := ProcessTx(mergedTx)

			txAddresses := ExtractTransactionAddresses(mergedTx)
			var currAddresses = make([]Address, len(txAddresses))
			for ii, address := range txAddresses {
				currAddresses[ii] = Address{Address: address}
			}

			currTxsWithAddresses[index] = TxWithAddresses{Tx: processedTx, Addresses: currAddresses}

		}(i, currTx, currTxResponse)

	}

	wg.Wait()
	return currTxsWithAddresses
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
