package main

import (
	bank "cosmos-exporter/cosmos/modules/bank"
	staking "cosmos-exporter/cosmos/modules/staking"
	txTypes "cosmos-exporter/cosmos/modules/tx"
	dbTypes "cosmos-exporter/db"
	"encoding/json"
	"fmt"
	"time"
)

//Unmarshal JSON to a particular type.
var messageTypeHandler = map[string]func() txTypes.CosmosMessage{
	"/cosmos.bank.v1beta1.MsgSend":                                func() txTypes.CosmosMessage { return &bank.WrapperMsgSend{} },
	"/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward":     func() txTypes.CosmosMessage { return &staking.WrapperMsgWithdrawDelegatorReward{} },
	"/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission": func() txTypes.CosmosMessage { return &staking.WrapperMsgWithdrawValidatorCommission{} },
}

//ParseCosmosMessageJSON - Parse a SINGLE Cosmos Message into the appropriate type.
func ParseCosmosMessageJSON(input []byte, log *txTypes.TxLogMessage) (txTypes.CosmosMessage, error) {
	//Figure out what type of Message this is based on the '@type' field that is included
	//in every Cosmos Message (can be seen in raw JSON for any cosmos transaction).
	var msg txTypes.CosmosMessage
	cosmosMessage := txTypes.Message{}
	if err := json.Unmarshal([]byte(input), &cosmosMessage); err != nil {
		fmt.Printf("Error parsing Cosmos message: %v\n", err)
		return nil, err
	}

	//So far we only parsed the '@type' field. Now we get a struct for that specific type.
	if msgHandlerFunc, ok := messageTypeHandler[cosmosMessage.Type]; ok {
		msg = msgHandlerFunc()
	} else {
		return nil, &txTypes.UnknownMessageError{MessageType: cosmosMessage.Type}
	}

	//Unmarshal the rest of the JSON now that we know the specific type.
	//Note that depending on the type, it may or may not care about logs.
	msg.CosmUnmarshal(cosmosMessage.Type, []byte(input), log)
	return msg, nil
}

func ProcessTxs(responseTxs []txTypes.TxStruct, responseTxResponses []txTypes.TxResponseStruct) []dbTypes.TxWithAddress {
	var currTxsWithAddresses = make([]dbTypes.TxWithAddress, len(responseTxs))
	//wg := sync.WaitGroup{}

	for i, currTx := range responseTxs {
		currTxResponse := responseTxResponses[i]
		//wg.Add(1)

		//go func(i int, currTx TxStruct, txResponse TxResponseStruct) {
		//	defer wg.Done()
		//tx data and tx_response data are split into 2 arrays in the json, combine into 1 using the corresponding index
		var mergedTx txTypes.MergedTx
		mergedTx.TxResponse = currTxResponse
		mergedTx.Tx = currTx

		processedTx := ProcessTx(mergedTx)

		txAddresses := ExtractTransactionAddresses(mergedTx)
		var currAddresses = make([]dbTypes.Address, len(txAddresses))
		for ii, address := range txAddresses {
			currAddresses[ii] = dbTypes.Address{Address: address}
		}

		var signer dbTypes.Address

		//TODO: Pass in key type (may be able to split from Type PublicKey)
		//TODO: Signers is an array, need a many to many for the signers in the model
		signerAddress, err := ParseSignerAddress(currTx.AuthInfo.TxSignerInfos[0].PublicKey.Key, "")

		if err != nil {
			signer.Address = ""
		} else {
			signer.Address = signerAddress
		}

		currTxsWithAddresses[i] = dbTypes.TxWithAddress{Tx: processedTx, SignerAddress: signer}

	}

	//wg.Wait()
	return currTxsWithAddresses
}

func ProcessTx(tx txTypes.MergedTx) dbTypes.Tx {
	timeStamp, _ := time.Parse(time.RFC3339, tx.TxResponse.TimeStamp)
	for messageIndex, message := range tx.Tx.Body.Messages {
		//Get the message log that corresponds to the current message
		messageLog := txTypes.GetMessageLogForIndex(tx.TxResponse.Log, messageIndex)
		jsonString, _ := json.Marshal(message)
		cosmosMessage, err := ParseCosmosMessageJSON(jsonString, messageLog)

		if err == nil {
			fmt.Printf("Cosmos message of known type: %s", cosmosMessage)
			//println(tx.TxResponse.Log)
		} else {
			println(err.Error())
			//println("------------------Cosmos message parsing failed. MESSAGE FORMAT FOLLOWS:---------------- \n\n")
			//spew.Dump(message)
			//println("\n------------------END MESSAGE----------------------\n")
		}

	}

	fees := ProcessFees(tx.Tx.AuthInfo.TxFee.TxFeeAmount)
	code := tx.TxResponse.Code
	return dbTypes.Tx{TimeStamp: timeStamp, Hash: tx.TxResponse.TxHash, Fees: fees, Code: code}
}

//ProcessFees returns a comma delimited list of fee amount/denoms
func ProcessFees(txFees []txTypes.TxFeeAmount) string {

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
