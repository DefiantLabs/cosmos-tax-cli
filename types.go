package main

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
	TxHash    string `json:"txhash"`
	Height    string `json:"height"`
	TimeStamp string `json:"timestamp"`
	Code      int    `json:"code"`
}

type TxBody struct {
	Messages []interface{} `json:"messages"`
}

type TxAuthInfo struct {
	TxFee TxFee `json:"fee"`
}

type TxFee struct {
	TxFeeAmount []TxFeeAmount `json:"amount"`
	GasLimit    string        `json:"gas_limit"`
}

type TxFeeAmount struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
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
