package main

import "encoding/json"

//TODO: Clean up types
type GetBlockByHeightResponse struct {
	BlockId BlockId       `json:"block_id"`
	Block   BlockResponse `json:"block"`
}

type BlockResponse struct {
	BlockData   BlockData   `json:"data"`
	BlockHeader BlockHeader `json:"header"`
}

type BlockId struct {
	Hash string `json:"hash"`
}

type BlockData struct {
	Txs []string `json:"txs"`
}

type BlockHeader struct {
	Height string `json:"height"`
}

type GetTxByBlockHeightResponse struct {
	Txs         []TxStruct         `json:"txs"`
	TxResponses []TxResponseStruct `json:"tx_responses"`
	Pagination  Pagination         `json:"pagination"`
}

type TxStruct struct {
	Body       TxBody     `json:"body"`
	AuthInfo   TxAuthInfo `json:"auth_info"`
	Signatures []string   `json:"signatures"`
}

type TxResponseStruct struct {
	TxHash    string         `json:"txhash"`
	Height    string         `json:"height"`
	TimeStamp string         `json:"timestamp"`
	Code      int64          `json:"code"`
	RawLog    string         `json:"raw_log"`
	Log       []TxLogMessage `json:"logs"`
}

// TxLogMessage:
// Cosmos blockchains return Transactions with an array of "logs" e.g.
//
// "logs": [
//	{
//		"msg_index": 0,
//		"events": [
//		  {
//			"type": "coin_received",
//			"attributes": [
//			  {
//				"key": "receiver",
//				"value": "juno128taw6wkhfq29u83lmh5qyfv8nff6h0w577vsy"
//			  }, ...
//			]
//		  } ...
//
// The individual log always has a msg_index corresponding to the Message from the Transaction.
// But the events are specific to each Message type, for example MsgSend might be different from
// any other message type.
//
// This struct just parses the KNOWN fields and leaves the other fields as raw JSON.
// More specific type parsers for each message type can parse those fields if they choose to.
type TxLogMessage struct {
	MessageIndex int             `json:"msg_index"`
	Events       json.RawMessage `json:"events"`
}

type TxBody struct {
	Messages []interface{} `json:"messages"`
}

type TxAuthInfo struct {
	TxFee         TxFee          `json:"fee"`
	TxSignerInfos []TxSignerInfo `json:"signer_infos"`
}

type TxFee struct {
	TxFeeAmount []TxFeeAmount `json:"amount"`
	GasLimit    string        `json:"gas_limit"`
}

type TxFeeAmount struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

type TxSignerInfo struct {
	PublicKey PublicKey `json:"public_key"`
}

type PublicKey struct {
	Type string `json:"@type"`
	Key  string `json:"key"`
}

type Pagination struct {
	NextKey string `json:"next_key"`
	Total   string `json:"total"`
}

//In the json, TX data is split into 2 arrays, used to merge the full dataset
type MergedTx struct {
	Tx         TxStruct
	TxResponse TxResponseStruct
}

type GetLatestBlockResponse struct {
	BlockId BlockId       `json:"block_id"`
	Block   BlockResponse `json:"block"`
}
