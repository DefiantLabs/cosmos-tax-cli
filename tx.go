package main

import (
	"fmt"
	"time"
)

func ProcessTxs(block Block, responseTxs []TxStruct, responseTxResponses []TxResponseStruct) []Tx {
	var currTxs []Tx
	for i, v := range responseTxs {

		//tx data and tx_response data are split into 2 arrays in the json, combine into 1 using the corresponding index
		var currTx MergedTx

		currTxResponse := responseTxResponses[i]

		currTx.TxResponse = currTxResponse
		currTx.Tx = v

		tx := ProcessTx(currTx, block)
		currTxs = append(currTxs, tx)
	}

	return currTxs
}

func ProcessTx(tx MergedTx, block Block) Tx {
	timeStamp, _ := time.Parse(time.RFC3339, tx.TxResponse.TimeStamp)

	fees := ProcessFees(tx.Tx.AuthInfo.TxFee.TxFeeAmount)

	return Tx{TimeStamp: timeStamp, Hash: tx.TxResponse.TxHash, Fees: fees, Block: block}
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
