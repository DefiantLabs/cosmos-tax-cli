package main

import (
	parsingTypes "cosmos-exporter/cosmos/modules"
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

func ProcessTxs(responseTxs []txTypes.TxStruct, responseTxResponses []txTypes.TxResponseStruct) []dbTypes.TxDBWrapper {
	var currTxDbWrappers = make([]dbTypes.TxDBWrapper, len(responseTxs))
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

		//This is being reworked
		// txAddresses := ExtractTransactionAddresses(mergedTx)
		// var currAddresses = make([]dbTypes.Address, len(txAddresses))
		// for ii, address := range txAddresses {
		// 	currAddresses[ii] = dbTypes.Address{Address: address}
		// }

		var signer dbTypes.Address

		//TODO: Pass in key type (may be able to split from Type PublicKey)
		//TODO: Signers is an array, need a many to many for the signers in the model
		signerAddress, err := ParseSignerAddress(currTx.AuthInfo.TxSignerInfos[0].PublicKey.Key, "")

		if err != nil {
			signer.Address = ""
		} else {
			signer.Address = signerAddress
		}

		processedTx.SignerAddress = signer

		currTxDbWrappers[i] = processedTx
	}

	//wg.Wait()
	return currTxDbWrappers
}

func ProcessTx(tx txTypes.MergedTx) dbTypes.TxDBWrapper {

	var txDBWapper dbTypes.TxDBWrapper

	timeStamp, _ := time.Parse(time.RFC3339, tx.TxResponse.TimeStamp)

	code := tx.TxResponse.Code

	var messages []dbTypes.MessageDBWrapper
	if code == 0 {
		//TODO: Pull this out into its own function for easier reading
		for messageIndex, message := range tx.Tx.Body.Messages {
			var currMessage dbTypes.Message
			currMessage.MessageIndex = messageIndex

			//Get the message log that corresponds to the current message
			messageLog := txTypes.GetMessageLogForIndex(tx.TxResponse.Log, messageIndex)
			jsonString, _ := json.Marshal(message)
			cosmosMessage, err := ParseCosmosMessageJSON(jsonString, messageLog)

			var currMessageDBWrapper dbTypes.MessageDBWrapper
			if err == nil {
				fmt.Printf("Cosmos message of known type: %s", cosmosMessage)
				currMessage.MessageType = cosmosMessage.GetType()
				currMessageDBWrapper.Message = currMessage

				//TODO: ParseRelevantData may need the logs to get the relevant information, unless we forever do that on the PasrseCosmosMessageJSON side
				var relevantData []parsingTypes.MessageRelevantInformation = cosmosMessage.ParseRelevantData()

				if len(relevantData) > 0 {
					var taxableEvents []dbTypes.TaxableEventDBWrapper = make([]dbTypes.TaxableEventDBWrapper, len(relevantData))
					for i, v := range relevantData {
						taxableEvents[i].TaxableEvent.Amount = v.Amount
						taxableEvents[i].TaxableEvent.Denomination = v.Denomination
						taxableEvents[i].SenderAddress = dbTypes.Address{Address: v.SenderAddress}
						taxableEvents[i].ReceiverAddress = dbTypes.Address{Address: v.ReceiverAddress}
					}
					currMessageDBWrapper.TaxableEvents = taxableEvents
				} else {
					currMessageDBWrapper.TaxableEvents = []dbTypes.TaxableEventDBWrapper{}
				}

			} else {
				println(err.Error())

				//type cast on error allows getting message type if it was parsed correctly
				re, ok := err.(*txTypes.UnknownMessageError)
				if ok {
					currMessage.MessageType = re.Type()
					currMessageDBWrapper.Message = currMessage
				} else {
					//What should we do here? This is an actual error during parsing
				}

				//println("------------------Cosmos message parsing failed. MESSAGE FORMAT FOLLOWS:---------------- \n\n")
				//spew.Dump(message)
				//println("\n------------------END MESSAGE----------------------\n")
			}

			messages = append(messages, currMessageDBWrapper)

		}
	}

	fees := ProcessFees(tx.Tx.AuthInfo.TxFee.TxFeeAmount)

	txDBWapper.Tx = dbTypes.Tx{TimeStamp: timeStamp, Hash: tx.TxResponse.TxHash, Fees: fees, Code: code}
	txDBWapper.Messages = messages

	return txDBWapper
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
