package core

import (
	"errors"

	parsingTypes "github.com/DefiantLabs/cosmos-exporter/cosmos/modules"
	bank "github.com/DefiantLabs/cosmos-exporter/cosmos/modules/bank"
	staking "github.com/DefiantLabs/cosmos-exporter/cosmos/modules/staking"
	tx "github.com/DefiantLabs/cosmos-exporter/cosmos/modules/tx"
	txTypes "github.com/DefiantLabs/cosmos-exporter/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-exporter/util"

	"fmt"
	"time"

	dbTypes "github.com/DefiantLabs/cosmos-exporter/db"

	"github.com/cosmos/cosmos-sdk/types"
	cosmosTx "github.com/cosmos/cosmos-sdk/types/tx"
)

//Unmarshal JSON to a particular type.
var messageTypeHandler = map[string]func() txTypes.CosmosMessage{
	"/cosmos.bank.v1beta1.MsgSend":                                func() txTypes.CosmosMessage { return &bank.WrapperMsgSend{} },
	"/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward":     func() txTypes.CosmosMessage { return &staking.WrapperMsgWithdrawDelegatorReward{} },
	"/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission": func() txTypes.CosmosMessage { return &staking.WrapperMsgWithdrawValidatorCommission{} },
}

//ParseCosmosMessageJSON - Parse a SINGLE Cosmos Message into the appropriate type.
func ParseCosmosMessage(message types.Msg, log *txTypes.TxLogMessage) (txTypes.CosmosMessage, error) {
	//Figure out what type of Message this is based on the '@type' field that is included
	//in every Cosmos Message (can be seen in raw JSON for any cosmos transaction).
	var msg txTypes.CosmosMessage
	cosmosMessage := txTypes.Message{}
	cosmosMessage.Type = types.MsgTypeURL(message)

	//So far we only parsed the '@type' field. Now we get a struct for that specific type.
	if msgHandlerFunc, ok := messageTypeHandler[cosmosMessage.Type]; ok {
		msg = msgHandlerFunc()
	} else {
		return nil, &txTypes.UnknownMessageError{MessageType: cosmosMessage.Type}
	}

	//Unmarshal the rest of the JSON now that we know the specific type.
	//Note that depending on the type, it may or may not care about logs.
	msg.HandleMsg(cosmosMessage.Type, message, log)
	return msg, nil
}

func toAttributes(attrs []types.Attribute) []txTypes.Attribute {
	list := []txTypes.Attribute{}
	for _, attr := range attrs {
		lma := txTypes.Attribute{Key: attr.Key, Value: attr.Value}
		list = append(list, lma)
	}

	return list
}

func toEvents(msgEvents types.StringEvents) []txTypes.LogMessageEvent {
	list := []txTypes.LogMessageEvent{}
	for _, evt := range msgEvents {
		lme := tx.LogMessageEvent{Type: evt.Type, Attributes: toAttributes(evt.Attributes)}
		list = append(list, lme)
	}

	return list
}

//TODO: get rid of some of the unnecessary types like cosmos-exporter/TxResponse.
//All those structs were legacy and for REST API support but we no longer really need it.
//For now I'm keeping it until we have RPC compatibility fully working and tested.
func ProcessRpcTxs(txEventResp *cosmosTx.GetTxsEventResponse) ([]dbTypes.TxDBWrapper, error) {
	var currTxDbWrappers = make([]dbTypes.TxDBWrapper, len(txEventResp.Txs))

	for txIdx := 0; txIdx < len(txEventResp.Txs); txIdx++ {
		//Indexer types only used by the indexer app (similar to the cosmos types)
		indexerMergedTx := txTypes.MergedTx{}
		indexerTx := txTypes.IndexerTx{}
		txBody := txTypes.TxBody{}
		authInfo := txTypes.TxAuthInfo{}

		currTx := txEventResp.Txs[txIdx]
		currTxResp := txEventResp.TxResponses[txIdx]
		currMessages := []types.Msg{}
		currLogMsgs := []tx.TxLogMessage{}

		// TODO: Get the TX fees, parse, put in DB, put in CSV ...
		// fees := currTx.AuthInfo.Fee
		// feeAmount := fees.Amount
		// feePayer := fees.Payer

		//Get the Messages and Message Logs
		for msgIdx := 0; msgIdx < len(currTx.Body.Messages); msgIdx++ {
			currMsg := currTx.Body.Messages[msgIdx].GetCachedValue()
			if currMsg != nil {
				msg := currMsg.(types.Msg)
				currMessages = append(currMessages, msg)

				if len(currTxResp.Logs) >= msgIdx+1 {
					msgEvents := currTxResp.Logs[msgIdx].Events

					currTxLog := tx.TxLogMessage{
						MessageIndex: msgIdx,
						Events:       toEvents(msgEvents),
					}

					currLogMsgs = append(currLogMsgs, currTxLog)
				}
			} else {
				return nil, errors.New("TX message could not be processed. CachedValue is not present.")
			}

		}

		txBody.Messages = currMessages
		indexerTx.Body = txBody
		indexerTx.AuthInfo = authInfo //Will eventually contain fees (TODO: impl fees)

		indexerTxResp := tx.TxResponse{
			TxHash:    currTxResp.TxHash,
			Height:    fmt.Sprintf("%d", currTxResp.Height),
			TimeStamp: currTxResp.Timestamp,
			RawLog:    currTxResp.RawLog,
			Log:       currLogMsgs,
			Code:      int64(currTxResp.Code),
		}

		indexerMergedTx.TxResponse = indexerTxResp
		indexerMergedTx.Tx = indexerTx

		processedTx := ProcessTx(indexerMergedTx)
		processedTx.SignerAddress = dbTypes.Address{Address: currTx.FeePayer().String()}

		//TODO: Pass in key type (may be able to split from Type PublicKey)
		//TODO: Signers is an array, need a many to many for the signers in the model
		//signerAddress, err := ParseSignerAddress(currTx.AuthInfo.SignerInfos[0].PublicKey, "")

		currTxDbWrappers[txIdx] = processedTx
	}

	return currTxDbWrappers, nil
}

func ProcessRestTxs(responseTxs []txTypes.IndexerTx, responseTxResponses []txTypes.TxResponse) []dbTypes.TxDBWrapper {
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
			cosmosMessage, err := ParseCosmosMessage(message, messageLog)

			var currMessageDBWrapper dbTypes.MessageDBWrapper
			if err == nil {
				fmt.Printf("Cosmos message of known type: %s", cosmosMessage)
				currMessage.MessageType = cosmosMessage.GetType()
				currMessageDBWrapper.Message = currMessage

				//TODO: ParseRelevantData may need the logs to get the relevant information, unless we forever do that on the ParseCosmosMessageJSON side
				var relevantData []parsingTypes.MessageRelevantInformation = cosmosMessage.ParseRelevantData()

				if len(relevantData) > 0 {
					var taxableEvents []dbTypes.TaxableEventDBWrapper = make([]dbTypes.TaxableEventDBWrapper, len(relevantData))
					for i, v := range relevantData {
						taxableEvents[i].TaxableTx.Amount = util.ToNumeric(v.Amount)
						taxableEvents[i].TaxableTx.Denomination = v.Denomination
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
