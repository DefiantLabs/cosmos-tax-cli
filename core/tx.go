package core

import (
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/DefiantLabs/cosmos-exporter/config"
	parsingTypes "github.com/DefiantLabs/cosmos-exporter/cosmos/modules"
	bank "github.com/DefiantLabs/cosmos-exporter/cosmos/modules/bank"
	staking "github.com/DefiantLabs/cosmos-exporter/cosmos/modules/staking"
	tx "github.com/DefiantLabs/cosmos-exporter/cosmos/modules/tx"
	txTypes "github.com/DefiantLabs/cosmos-exporter/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-exporter/db"
	"github.com/DefiantLabs/cosmos-exporter/osmosis/modules/gamm"
	"github.com/DefiantLabs/cosmos-exporter/util"
	"go.uber.org/zap"

	"fmt"
	"time"

	dbTypes "github.com/DefiantLabs/cosmos-exporter/db"

	cryptoTypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types"
	cosmosTx "github.com/cosmos/cosmos-sdk/types/tx"
)

//Unmarshal JSON to a particular type.
var messageTypeHandler = map[string]func() txTypes.CosmosMessage{
	"/cosmos.bank.v1beta1.MsgSend":                                func() txTypes.CosmosMessage { return &bank.WrapperMsgSend{} },
	"/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward":     func() txTypes.CosmosMessage { return &staking.WrapperMsgWithdrawDelegatorReward{} },
	"/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission": func() txTypes.CosmosMessage { return &staking.WrapperMsgWithdrawValidatorCommission{} },
	"/osmosis.gamm.v1beta1.MsgSwapExactAmountIn":                  func() txTypes.CosmosMessage { return &gamm.WrapperMsgSwapExactAmountIn{} },
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

		currTx := txEventResp.Txs[txIdx]
		currTxResp := txEventResp.TxResponses[txIdx]
		currMessages := []types.Msg{}
		currLogMsgs := []tx.TxLogMessage{}

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
				return nil, errors.New("tx message could not be processed. CachedValue is not present")
			}

		}

		txBody.Messages = currMessages
		indexerTx.Body = txBody

		indexerTxResp := tx.TxResponse{
			TxHash:    currTxResp.TxHash,
			Height:    fmt.Sprintf("%d", currTxResp.Height),
			TimeStamp: currTxResp.Timestamp,
			RawLog:    currTxResp.RawLog,
			Log:       currLogMsgs,
			Code:      int64(currTxResp.Code),
		}

		indexerTx.AuthInfo = *currTx.AuthInfo
		indexerMergedTx.TxResponse = indexerTxResp
		indexerMergedTx.Tx = indexerTx
		indexerMergedTx.Tx.AuthInfo = *currTx.AuthInfo

		processedTx, err := ProcessTx(indexerMergedTx)
		if err != nil {
			return currTxDbWrappers, err
		}

		processedTx.SignerAddress = dbTypes.Address{Address: currTx.FeePayer().String()}

		//TODO: Pass in key type (may be able to split from Type PublicKey)
		//TODO: Signers is an array, need a many to many for the signers in the model
		//signerAddress, err := ParseSignerAddress(currTx.AuthInfo.SignerInfos[0].PublicKey, "")

		currTxDbWrappers[txIdx] = processedTx
	}

	return currTxDbWrappers, nil
}

var allSwaps = []gamm.ArbitrageTx{}

func AnalyzeSwaps() {
	earliestTime := time.Now()
	latestTime := time.Now()
	profit := 0.0
	fmt.Printf("%d total uosmo arbitrage swaps\n", len(allSwaps))

	for _, swap := range allSwaps {
		if swap.TokenIn.Denom == swap.TokenOut.Denom && swap.TokenIn.Denom == "uosmo" {
			amount := swap.TokenOut.Amount.Sub(swap.TokenIn.Amount)
			if amount.GT(types.ZeroInt()) {
				txProfit := amount.ToDec().Quo(types.NewDec(1000000)).MustFloat64()
				profit = profit + txProfit
			}

			if swap.BlockTime.Before(earliestTime) {
				earliestTime = swap.BlockTime
			}
			if swap.BlockTime.After(latestTime) {
				latestTime = swap.BlockTime
			}
		}
	}

	fmt.Printf("Profit (OSMO): %.10f, days: %f\n", profit, latestTime.Sub(earliestTime).Hours()/24)
}

func ProcessTx(tx txTypes.MergedTx) (dbTypes.TxDBWrapper, error) {

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

			if msgSwapExactIn, ok := cosmosMessage.(*gamm.WrapperMsgSwapExactAmountIn); ok {
				t, err := time.Parse(time.RFC3339, tx.TxResponse.TimeStamp)
				if err == nil {
					newSwap := gamm.ArbitrageTx{TokenIn: msgSwapExactIn.TokenIn, TokenOut: msgSwapExactIn.TokenOut, BlockTime: t}
					allSwaps = append(allSwaps, newSwap)
				}
			}

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
						if v.AmountSent != nil {
							taxableEvents[i].TaxableTx.AmountSent = util.ToNumeric(v.AmountSent)
						}
						if v.AmountReceived != nil {
							taxableEvents[i].TaxableTx.AmountReceived = util.ToNumeric(v.AmountReceived)
						}

						var denomSent dbTypes.Denom
						if v.DenominationSent != "" {
							denomSent, err = db.GetDenomForBase(v.DenominationSent)
							if err != nil {
								config.Logger.Error("Denom lookup", zap.Error(err), zap.String("denom sent", v.DenominationSent))
								return txDBWapper, err
							}
							taxableEvents[i].TaxableTx.DenominationSent = denomSent
						}

						var denomReceived dbTypes.Denom
						if v.DenominationReceived != "" {
							denomReceived, err = db.GetDenomForBase(v.DenominationReceived)
							if err != nil {
								config.Logger.Error("Denom lookup", zap.Error(err), zap.String("denom received", v.DenominationReceived))
								return txDBWapper, err
							}
							taxableEvents[i].TaxableTx.DenominationReceived = denomReceived
						}

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

	fees, err := ProcessFees(tx.Tx.AuthInfo)
	if err != nil {
		return txDBWapper, err
	}

	txDBWapper.Tx = dbTypes.Tx{TimeStamp: timeStamp, Hash: tx.TxResponse.TxHash, Fees: fees, Code: code}
	txDBWapper.Messages = messages

	return txDBWapper, nil
}

//ProcessFees returns a comma delimited list of fee amount/denoms
func ProcessFees(authInfo cosmosTx.AuthInfo) ([]dbTypes.Fee, error) {
	//TODO handle granter? Almost nobody uses it.
	feeCoins := authInfo.Fee.Amount
	payer := authInfo.Fee.GetPayer()
	fees := []dbTypes.Fee{}

	for _, coin := range feeCoins {
		zeroFee := big.NewInt(0)

		//There are chains like Osmosis that do not require TX fees for certain TXs
		if zeroFee.Cmp(coin.Amount.BigInt()) != 0 {
			amount := util.ToNumeric(coin.Amount.BigInt())
			denom, denomErr := db.GetDenomForBase(coin.Denom)
			if denomErr != nil {
				return nil, denomErr
			}
			payerAddr := dbTypes.Address{}

			if payer == "" {
				cpk := authInfo.SignerInfos[0].PublicKey.GetCachedValue()
				pubKey := cpk.(cryptoTypes.PubKey)
				hexPub := hex.EncodeToString(pubKey.Bytes())
				bechAddr, err := ParseSignerAddress(hexPub, "")
				if err != nil {
					fmt.Printf("Err %s\n", err.Error())
				} else {
					payerAddr.Address = bechAddr
				}
			} else {
				payerAddr.Address = payer
			}

			fees = append(fees, dbTypes.Fee{Amount: amount, Denomination: denom, PayerAddress: payerAddr})
		}
	}

	return fees, nil
}
