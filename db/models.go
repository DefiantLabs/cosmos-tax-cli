package db

import "time"

type Block struct {
	ID           uint
	Height       int64 `gorm:"uniqueIndex:chainheight"`
	BlockchainID uint  `gorm:"uniqueIndex:chainheight"`
	Chain        Chain `gorm:"foreignKey:BlockchainID"`
}

type Chain struct {
	ID      uint   `gorm:"primaryKey"`
	ChainID string `gorm:"uniqueIndex"` //e.g. osmosis-1
	Name    string //e.g. Osmosis
}

type Tx struct {
	ID              uint
	TimeStamp       time.Time
	Hash            string
	Fees            string
	Code            int64
	BlockId         uint
	Block           Block
	SignerAddressId *int //*int allows foreign key to be null
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

const (
	OsmosisRewardDistribution uint = iota
)

//An event does not necessarily need to be part of a Transaction. For example, Osmosis rewards.
//Events can happen on chain and generate tendermint ABCI events that do not show up in transactions.
type TaxableEvent struct {
	ID             uint
	Source         uint //This will indicate what type of event occurred on chain. Currently, only used for Osmosis rewards.
	Amount         float64
	DenominationID uint
	Denomination   SimpleDenom `gorm:"foreignKey:DenominationID"`
	AddressID      uint        `gorm:"index:idx_addr"`
	EventAddress   Address     `gorm:"foreignKey:AddressID"`
	BlockID        uint
	Block          Block `gorm:"foreignKey:BlockID"`
}

type SimpleDenom struct {
	ID     uint
	Denom  string `gorm:"uniqueIndex:denom_idx"`
	Symbol string `gorm:"uniqueIndex:denom_idx"`
}

func (TaxableEvent) TableName() string {
	return "taxable_event" //Legacy
}

type TaxableTransaction struct {
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

func (TaxableTransaction) TableName() string {
	return "taxable_tx" //Legacy
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
	TaxableTx       TaxableTransaction
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
