package core

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	parsingTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/authz"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/bank"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/distribution"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/gov"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/ibc"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/slashing"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/staking"
	tx "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	txTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	dbTypes "github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/incentives"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/lockup"
	"github.com/DefiantLabs/cosmos-tax-cli/util"

	"github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	cryptoTypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types"
	cosmosTx "github.com/cosmos/cosmos-sdk/types/tx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Unmarshal JSON to a particular type.
var messageTypeHandler = map[string]func() txTypes.CosmosMessage{
	bank.MsgSend:                                func() txTypes.CosmosMessage { return &bank.WrapperMsgSend{} },
	bank.MsgMultiSend:                           func() txTypes.CosmosMessage { return &bank.WrapperMsgMultiSend{} },
	distribution.MsgWithdrawDelegatorReward:     func() txTypes.CosmosMessage { return &distribution.WrapperMsgWithdrawDelegatorReward{} },
	distribution.MsgWithdrawValidatorCommission: func() txTypes.CosmosMessage { return &distribution.WrapperMsgWithdrawValidatorCommission{} },
	distribution.MsgFundCommunityPool:           func() txTypes.CosmosMessage { return &distribution.WrapperMsgFundCommunityPool{} },
	gov.MsgDeposit:                              func() txTypes.CosmosMessage { return &gov.WrapperMsgDeposit{} },
	gov.MsgSubmitProposal:                       func() txTypes.CosmosMessage { return &gov.WrapperMsgSubmitProposal{} },
	staking.MsgDelegate:                         func() txTypes.CosmosMessage { return &staking.WrapperMsgDelegate{} },
	staking.MsgUndelegate:                       func() txTypes.CosmosMessage { return &staking.WrapperMsgUndelegate{} },
	staking.MsgBeginRedelegate:                  func() txTypes.CosmosMessage { return &staking.WrapperMsgBeginRedelegate{} },
	ibc.MsgTransfer:                             func() txTypes.CosmosMessage { return &ibc.WrapperMsgTransfer{} },
}

var messageTypeIgnorer = map[string]interface{}{
	authz.MsgExec:                      nil,
	authz.MsgGrant:                     nil,
	authz.MsgRevoke:                    nil,
	distribution.MsgSetWithdrawAddress: nil,
	gov.MsgVote:                        nil,
	ibc.MsgUpdateClient:                nil,
	ibc.MsgAcknowledgement:             nil,
	ibc.MsgRecvPacket:                  nil,
	ibc.MsgTimeout:                     nil,
	ibc.MsgCreateClient:                nil,
	ibc.MsgConnectionOpenTry:           nil,
	ibc.MsgConnectionOpenConfirm:       nil,
	ibc.MsgChannelOpenTry:              nil,
	ibc.MsgChannelOpenConfirm:          nil,
	ibc.MsgConnectionOpenInit:          nil,
	ibc.MsgConnectionOpenAck:           nil,
	ibc.MsgChannelOpenInit:             nil,
	ibc.MsgChannelOpenAck:              nil,
	incentives.MsgCreateGauge:          nil,
	incentives.MsgAddToGauge:           nil,
	lockup.MsgBeginUnlocking:           nil,
	lockup.MsgLockTokens:               nil,
	slashing.MsgUnjail:                 nil,
	slashing.MsgUpdateParams:           nil,
	staking.MsgCreateValidator:         nil,
	staking.MsgEditValidator:           nil,
}

// Merge the chain specific message type handlers into the core message type handler map
// If a core message type is defined in the chain specific, it will overide the value
// in the core message type handler (useful if a chain has changed the core behavior of a base type and needs to be parsed differently).
func ChainSpecificMessageTypeHandlerBootstrap(chainID string) {
	var chainSpecificMessageTpeHandler map[string]func() txTypes.CosmosMessage
	if chainID == osmosis.ChainID {
		chainSpecificMessageTpeHandler = osmosis.MessageTypeHandler
	}
	for key, value := range chainSpecificMessageTpeHandler {
		messageTypeHandler[key] = value
	}
}

// ParseCosmosMessageJSON - Parse a SINGLE Cosmos Message into the appropriate type.
func ParseCosmosMessage(message types.Msg, log *txTypes.LogMessage) (txTypes.CosmosMessage, string, error) {
	// Figure out what type of Message this is based on the '@type' field that is included
	// in every Cosmos Message (can be seen in raw JSON for any cosmos transaction).
	var msg txTypes.CosmosMessage
	cosmosMessage := txTypes.Message{}
	cosmosMessage.Type = types.MsgTypeURL(message)

	// So far we only parsed the '@type' field. Now we get a struct for that specific type.
	if msgHandlerFunc, ok := messageTypeHandler[cosmosMessage.Type]; ok {
		msg = msgHandlerFunc()
	} else {
		return nil, cosmosMessage.Type, txTypes.ErrUnknownMessage
	}

	// Unmarshal the rest of the JSON now that we know the specific type.
	// Note that depending on the type, it may or may not care about logs.
	err := msg.HandleMsg(cosmosMessage.Type, message, log)
	return msg, cosmosMessage.Type, err
}

func toAttributes(attrs []types.Attribute) []txTypes.Attribute {
	list := []txTypes.Attribute{}
	for _, attr := range attrs {
		lma := txTypes.Attribute{Key: attr.Key, Value: attr.Value}
		list = append(list, lma)
	}

	return list
}

func toEvents(msgEvents types.StringEvents) (list []txTypes.LogMessageEvent) {
	for _, evt := range msgEvents {
		lme := tx.LogMessageEvent{Type: evt.Type, Attributes: toAttributes(evt.Attributes)}
		list = append(list, lme)
	}

	return list
}

// TODO: get rid of some of the unnecessary types like cosmos-tax-cli/TxResponse.
// All those structs were legacy and for REST API support but we no longer really need it.
// For now I'm keeping it until we have RPC compatibility fully working and tested.
func ProcessRPCTXs(db *gorm.DB, txEventResp *cosmosTx.GetTxsEventResponse) ([]dbTypes.TxDBWrapper, time.Time, error) {
	var currTxDbWrappers = make([]dbTypes.TxDBWrapper, len(txEventResp.Txs))
	var blockTime time.Time
	var blockTimeFound bool
	for txIdx := range txEventResp.Txs {
		// Indexer types only used by the indexer app (similar to the cosmos types)
		var indexerMergedTx txTypes.MergedTx
		var indexerTx txTypes.IndexerTx
		var txBody txTypes.Body
		var currMessages []types.Msg
		var currLogMsgs []tx.LogMessage
		currTx := txEventResp.Txs[txIdx]
		currTxResp := txEventResp.TxResponses[txIdx]

		// Get the Messages and Message Logs
		for msgIdx := range currTx.Body.Messages {
			currMsg := currTx.Body.Messages[msgIdx].GetCachedValue() // FIXME: understand why we use this....
			if currMsg != nil {
				msg := currMsg.(types.Msg)
				currMessages = append(currMessages, msg)
				if len(currTxResp.Logs) >= msgIdx+1 {
					msgEvents := currTxResp.Logs[msgIdx].Events
					currTxLog := tx.LogMessage{
						MessageIndex: msgIdx,
						Events:       toEvents(msgEvents),
					}
					currLogMsgs = append(currLogMsgs, currTxLog)
				}
			} else {
				return nil, blockTime, fmt.Errorf("tx message could not be processed. CachedValue is not present. TX Hash: %s, Msg type: %s, Msg index: %d, Code: %d",
					currTxResp.TxHash,
					currTx.Body.Messages[msgIdx].TypeUrl,
					msgIdx,
					currTxResp.Code,
				)
			}
		}

		txBody.Messages = currMessages
		indexerTx.Body = txBody

		indexerTxResp := tx.Response{
			TxHash:    currTxResp.TxHash,
			Height:    fmt.Sprintf("%d", currTxResp.Height),
			TimeStamp: currTxResp.Timestamp,
			RawLog:    currTxResp.RawLog,
			Log:       currLogMsgs,
			Code:      currTxResp.Code,
		}

		indexerTx.AuthInfo = *currTx.AuthInfo
		indexerTx.Signers = currTx.GetSigners()
		indexerMergedTx.TxResponse = indexerTxResp
		indexerMergedTx.Tx = indexerTx
		indexerMergedTx.Tx.AuthInfo = *currTx.AuthInfo

		processedTx, txTime, err := ProcessTx(db, indexerMergedTx)
		if err != nil {
			return currTxDbWrappers, blockTime, err
		}

		if !blockTimeFound {
			blockTime = txTime
		}

		processedTx.SignerAddress = dbTypes.Address{Address: currTx.FeePayer().String()}

		// TODO: Pass in key type (may be able to split from Type PublicKey)
		// TODO: Signers is an array, need a many to many for the signers in the model
		// signerAddress, err := ParseSignerAddress(currTx.AuthInfo.SignerInfos[0].PublicKey, "")

		currTxDbWrappers[txIdx] = processedTx
	}

	return currTxDbWrappers, blockTime, nil
}

func ProcessTx(db *gorm.DB, tx txTypes.MergedTx) (txDBWapper dbTypes.TxDBWrapper, txTime time.Time, err error) {
	txTime, err = time.Parse(time.RFC3339, tx.TxResponse.TimeStamp)
	if err != nil {
		config.Log.Error("Error parsing tx timestamp.", zap.Error(err))
		return
	}

	code := tx.TxResponse.Code

	var messages []dbTypes.MessageDBWrapper

	// non-zero code means the Tx was unsuccessful. We will still need to account for fees in both cases though.
	if code == 0 {
		// TODO: Pull this out into its own function for easier reading
		for messageIndex, message := range tx.Tx.Body.Messages {
			var currMessage dbTypes.Message
			var currMessageType dbTypes.MessageType
			currMessage.MessageIndex = messageIndex

			// Get the message log that corresponds to the current message
			var currMessageDBWrapper dbTypes.MessageDBWrapper
			messageLog := txTypes.GetMessageLogForIndex(tx.TxResponse.Log, messageIndex)
			cosmosMessage, msgType, err := ParseCosmosMessage(message, messageLog)
			if err != nil {
				currMessageType.MessageType = msgType
				currMessage.MessageType = currMessageType
				currMessageDBWrapper.Message = currMessage
				if err != txTypes.ErrUnknownMessage {
					// What should we do here? This is an actual error during parsing
					config.Log.Error(fmt.Sprintf("[Block: %v] ParseCosmosMessage failed for msg of type '%v'.", tx.TxResponse.Height, msgType), zap.Error(err))
					config.Log.Error(fmt.Sprint(messageLog))
					config.Log.Error(tx.TxResponse.TxHash)
					config.Log.Fatal("Issue parsing a cosmos msg that we DO have a parser for! PLEASE INVESTIGATE")
				}
				// if this msg isn't include in our list of those we are explicitly ignoring, do something about it.
				if _, ok := messageTypeIgnorer[msgType]; !ok {
					config.Log.Warn(fmt.Sprintf("[Block: %v] ParseCosmosMessage failed for msg of type '%v'. We do not currently have a message handler for this message type", tx.TxResponse.Height, msgType))
				}
				// println("------------------Cosmos message parsing failed. MESSAGE FORMAT FOLLOWS:---------------- \n\n")
				// spew.Dump(message)
				// println("\n------------------END MESSAGE----------------------\n")
			} else {
				config.Log.Debug(fmt.Sprintf("[Block: %v] Cosmos message of known type: %s", tx.TxResponse.Height, cosmosMessage))
				currMessageType.MessageType = cosmosMessage.GetType()
				currMessage.MessageType = currMessageType
				currMessageDBWrapper.Message = currMessage

				// TODO: ParseRelevantData may need the logs to get the relevant information, unless we forever do that on the ParseCosmosMessageJSON side
				var relevantData []parsingTypes.MessageRelevantInformation = cosmosMessage.ParseRelevantData()

				if len(relevantData) > 0 {
					var taxableTxs = make([]dbTypes.TaxableTxDBWrapper, len(relevantData))
					for i, v := range relevantData {
						if v.AmountSent != nil {
							taxableTxs[i].TaxableTx.AmountSent = util.ToNumeric(v.AmountSent)
						}
						if v.AmountReceived != nil {
							taxableTxs[i].TaxableTx.AmountReceived = util.ToNumeric(v.AmountReceived)
						}

						var denomSent dbTypes.Denom
						if v.DenominationSent != "" {
							denomSent, err = dbTypes.GetDenomForBase(v.DenominationSent)
							if err != nil {
								// attempt to add missing denoms to the database
								config.Log.Warn("Denom lookup failed. Will be inserted as UNKNOWN", zap.Error(err), zap.String("denom sent", v.DenominationSent))

								denomSent, err = dbTypes.AddUnknownDenom(db, v.DenominationSent)
								if err != nil {
									config.Log.Error("There was an error adding a missing denom", zap.Error(err), zap.String("denom received", v.DenominationSent))
									return txDBWapper, txTime, err
								}
							}

							taxableTxs[i].TaxableTx.DenominationSent = denomSent
						}

						var denomReceived dbTypes.Denom
						if v.DenominationReceived != "" {
							denomReceived, err = dbTypes.GetDenomForBase(v.DenominationReceived)
							if err != nil {
								// attempt to add missing denoms to the database
								config.Log.Error("Denom lookup failed. Will be inserted as UNKNOWN", zap.Error(err), zap.String("denom received", v.DenominationReceived))
								denomReceived, err = dbTypes.AddUnknownDenom(db, v.DenominationReceived)
								if err != nil {
									config.Log.Error("There was an error adding a missing denom", zap.Error(err), zap.String("denom received", v.DenominationReceived))
									return txDBWapper, txTime, err
								}
							}
							taxableTxs[i].TaxableTx.DenominationReceived = denomReceived
						}

						taxableTxs[i].SenderAddress = dbTypes.Address{Address: strings.ToLower(v.SenderAddress)}
						taxableTxs[i].ReceiverAddress = dbTypes.Address{Address: strings.ToLower(v.ReceiverAddress)}
					}
					currMessageDBWrapper.TaxableTxs = taxableTxs
				} else {
					currMessageDBWrapper.TaxableTxs = []dbTypes.TaxableTxDBWrapper{}
				}
			}

			messages = append(messages, currMessageDBWrapper)
		}
	}

	fees, err := ProcessFees(db, tx.Tx.AuthInfo, tx.Tx.Signers)
	if err != nil {
		return txDBWapper, txTime, err
	}

	txDBWapper.Tx = dbTypes.Tx{Hash: tx.TxResponse.TxHash, Fees: fees, Code: code}
	txDBWapper.Messages = messages

	return txDBWapper, txTime, nil
}

// ProcessFees returns a comma delimited list of fee amount/denoms
func ProcessFees(db *gorm.DB, authInfo cosmosTx.AuthInfo, signers []types.AccAddress) ([]dbTypes.Fee, error) {
	// TODO handle granter? Almost nobody uses it.
	feeCoins := authInfo.Fee.Amount
	payer := authInfo.Fee.GetPayer()
	fees := []dbTypes.Fee{}

	for _, coin := range feeCoins {
		zeroFee := big.NewInt(0)

		// There are chains like Osmosis that do not require TX fees for certain TXs
		if zeroFee.Cmp(coin.Amount.BigInt()) != 0 {
			amount := util.ToNumeric(coin.Amount.BigInt())
			denom, err := dbTypes.GetDenomForBase(coin.Denom)
			if err != nil {
				// attempt to add missing denoms to the database
				config.Log.Error("Denom lookup failed. Will be inserted as UNKNOWN", zap.Error(err), zap.String("denom received", coin.Denom))
				denom, err = dbTypes.AddUnknownDenom(db, coin.Denom)
				if err != nil {
					config.Log.Error("There was an error adding a missing denom", zap.Error(err), zap.String("denom received", coin.Denom))
					return nil, err
				}
			}
			payerAddr := dbTypes.Address{}
			if payer != "" {
				payerAddr.Address = payer
			} else {
				if authInfo.SignerInfos[0].PublicKey == nil && len(signers) > 0 {
					payerAddr.Address = signers[0].String()
				} else {
					var pubKey cryptoTypes.PubKey
					cpk := authInfo.SignerInfos[0].PublicKey.GetCachedValue()

					// if this is a multisig msg, handle it specially
					if strings.Contains(authInfo.SignerInfos[0].ModeInfo.GetMulti().String(), "mode:SIGN_MODE_LEGACY_AMINO_JSON") {
						pubKey = cpk.(*multisig.LegacyAminoPubKey).GetPubKeys()[0]
					} else {
						pubKey = cpk.(cryptoTypes.PubKey)
					}
					hexPub := hex.EncodeToString(pubKey.Bytes())
					bechAddr, err := ParseSignerAddress(hexPub, "")
					if err != nil {
						config.Log.Error(fmt.Sprintf("Error parsing signer address '%v' for tx.", hexPub), zap.Error(err))
					} else {
						payerAddr.Address = bechAddr
					}
				}
			}

			fees = append(fees, dbTypes.Fee{Amount: amount, Denomination: denom, PayerAddress: payerAddr})
		}
	}

	return fees, nil
}
