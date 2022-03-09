package main

import (
	"encoding/json"
	"fmt"
	"time"

	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/davecgh/go-spew/spew"
)

type TxWithAddresses struct {
	Tx        Tx
	Addresses []Address
}

//MessageEnvelope Allows delayed parsing with RawMessage type.
//see http://eagain.net/articles/go-json-kind/ and https://golang.org/pkg/encoding/json/#RawMessage
type MessageEnvelope struct {
	Type string `json:"@type"`
}

type WrapperMsgSend struct {
	Type          string `json:"@type"`
	CosmosMsgSend bankTypes.MsgSend
}

//CosmUnmarshal(): Unmarshal JSON for MsgSend
func (sf *WrapperMsgSend) CosmUnmarshal(msgType string, raw []byte) error {
	sf.Type = msgType
	if err := json.Unmarshal(raw, &sf.CosmosMsgSend); err != nil {
		fmt.Println("Error parsing message: " + err.Error())
		return err
	}
	return nil
}

//CosmosMessage represents a Cosmos blockchain Message (part of a transaction).
//CosmUnmarshal() unmarshals the specific cosmos message type (e.g. MsgSend).
//First arg must always be the message type itself, as this won't be parsed in CosmUnmarshal.
type CosmosMessage interface {
	CosmUnmarshal(string, []byte) error
}

//Unmarshal JSON to a particular type.
var messageTypeHandler = map[string]func() CosmosMessage{
	"/cosmos.bank.v1beta1.MsgSend": func() CosmosMessage { return &WrapperMsgSend{} },
}

//ParseCosmosMessageJSON - Parse a single Cosmos Message into the appropriate type.
func ParseCosmosMessageJSON(input []byte) (CosmosMessage, error) {

	//We start with the SINGLE cosmos message. If a transaction has multiple messages,
	//you must call ParseCosmosMessageJSON multiple times, once per message.
	cosmosMessage := MessageEnvelope{}
	if err := json.Unmarshal([]byte(input), &cosmosMessage); err != nil {
		fmt.Printf("Error parsing Cosmos message: %v\n", err)
		return nil, err
	}

	msg := messageTypeHandler[cosmosMessage.Type]()
	msg.CosmUnmarshal(cosmosMessage.Type, []byte(input))

	return msg, nil
}

func ProcessTxs(responseTxs []TxStruct, responseTxResponses []TxResponseStruct) []TxWithAddresses {
	var currTxsWithAddresses = make([]TxWithAddresses, len(responseTxs))
	//wg := sync.WaitGroup{}

	for i, currTx := range responseTxs {
		currTxResponse := responseTxResponses[i]
		//wg.Add(1)

		//go func(i int, currTx TxStruct, txResponse TxResponseStruct) {
		//	defer wg.Done()
		//tx data and tx_response data are split into 2 arrays in the json, combine into 1 using the corresponding index
		var mergedTx MergedTx
		mergedTx.TxResponse = currTxResponse
		mergedTx.Tx = currTx

		processedTx := ProcessTx(mergedTx)

		txAddresses := ExtractTransactionAddresses(mergedTx)
		var currAddresses = make([]Address, len(txAddresses))
		for ii, address := range txAddresses {
			currAddresses[ii] = Address{Address: address}
		}

		currTxsWithAddresses[i] = TxWithAddresses{Tx: processedTx, Addresses: currAddresses}

		//}(i, currTx, currTxResponse)

	}

	//wg.Wait()
	return currTxsWithAddresses
}

func ProcessTx(tx MergedTx) Tx {
	timeStamp, _ := time.Parse(time.RFC3339, tx.TxResponse.TimeStamp)
	for _, message := range tx.Tx.Body.Messages {
		jsonString, _ := json.Marshal(message)
		cosmosMessage, err := ParseCosmosMessageJSON(jsonString)
		if err != nil {
			fmt.Printf("Cosmos message: %+v", cosmosMessage)
		} else {
			println("------------------Cosmos message parsing failed. MESSAGE FORMAT FOLLOWS:---------------- \n\n")
			spew.Dump(message)
			println("\n------------------END MESSAGE----------------------\n")
		}

	}

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
