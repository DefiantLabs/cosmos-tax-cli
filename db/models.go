package db

import "time"

type Block struct {
	ID     uint
	Height uint64 `gorm:"uniqueIndex"`
}

type Tx struct {
	ID              uint
	TimeStamp       time.Time
	Hash            string `gorm:"uniqueIndex"`
	Fees            string
	Code            int64
	BlockId         uint
	Block           Block
	SignerAddressId *int //int pointer allows foreign key to be null
	SignerAddress   Address
}

type Address struct {
	ID      uint
	Address string `gorm:"uniqueIndex"`
}

type Message struct {
	ID           uint
	TxId         uint
	Tx           Tx
	MessageType  string
	MessageIndex int
}

type TaxableEvent struct {
	ID                uint
	MessageId         uint
	Message           Message
	Amount            float64
	Denomination      string
	SenderAddressId   uint
	SenderAddress     Address
	ReceiverAddressId uint
	ReceiverAddress   Address
}

type TxDBWrapper struct {
	Tx            Tx
	SignerAddress Address
	Messages      []Message
	TaxableEvents []TaxableEvent
}
