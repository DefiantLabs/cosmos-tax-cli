package db

import "time"

type Block struct {
	ID     uint
	Height uint64 `gorm:"uniqueIndex"`
}

type Tx struct {
	ID              uint
	TimeStamp       time.Time
	Hash            string
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
	MessageType  string `gorm:"index"`
	MessageIndex int
}

type TaxableEvent struct {
	ID                uint
	MessageId         uint
	Message           Message
	Amount            float64
	Denomination      string
	SenderAddressId   *uint `gorm:"index:idx_sender"`
	SenderAddress     Address
	ReceiverAddressId *uint `gorm:"index:idx_receiver"`
	ReceiverAddress   Address
}

type Denom struct {
	ID     uint
	Base   string
	Name   string `gorm:"uniqueIndex"`
	Symbol string
}

type DenomUnit struct {
	ID       uint
	DenomID  uint
	Denom    Denom
	Exponent uint
	Name     string `gorm:"unique"`
}

type DenomUnitAlias struct {
	ID          uint
	DenomUnitId uint
	DenomUnit   DenomUnit
	Alias       string `gorm:"unique"`
}

//Store transactions with their messages for easy database creation
type TxDBWrapper struct {
	Tx            Tx
	SignerAddress Address
	Messages      []MessageDBWrapper
}

//Store messages with their taxable events for easy database creation
type MessageDBWrapper struct {
	Message       Message
	TaxableEvents []TaxableEventDBWrapper
}

//Store taxable events with their sender/receiver address for easy database creation
type TaxableEventDBWrapper struct {
	TaxableEvent    TaxableEvent
	SenderAddress   Address
	ReceiverAddress Address
}

type DenomDBWrapper struct {
	Denom      Denom
	DenomUnits []DenomUnitDBWrapper
}

type DenomUnitDBWrapper struct {
	DenomUnit DenomUnit
	Aliases   []DenomUnitAlias
}
