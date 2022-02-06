package main

//TODO: Clean up types
type GetBlockByHeightResponse struct {
	BlockId BlockId `json:"block_id"`
	Block   Block   `json:"block"`
}

type Block struct {
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

type GetTxByHashResponse struct {
	Tx         Tx         `json:"tx"`
	TxResponse TxResponse `json:"tx_response"`
}

type Tx struct {
	Body       TxBody     `json:"body"`
	AuthInfo   TxAuthInfo `json:"auth_info"`
	Signatures []string   `json:"signatures"`
}

type TxResponse struct {
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

type GetTxByBlockHeightResponse struct {
	Txs         []Tx         `json:"txs"`
	TxResponses []TxResponse `json:"tx_responses"`
	Pagination  Pagination   `json:"pagination"`
}

type Pagination struct {
	NextKey string `json:"next_key"`
	Total   string `json:"total"`
}

type SingleTx struct {
	Tx         Tx
	TxResponse TxResponse
}
