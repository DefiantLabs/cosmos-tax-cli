package db

import (
	"time"

	"github.com/shopspring/decimal"
)

type Block struct {
	ID           uint
	TimeStamp    time.Time
	Height       int64 `gorm:"uniqueIndex:chainheight"`
	BlockchainID uint  `gorm:"uniqueIndex:chainheight"`
	Chain        Chain `gorm:"foreignKey:BlockchainID"`
}

type FailedBlock struct {
	ID           uint
	Height       int64 `gorm:"uniqueIndex:failedchainheight"`
	BlockchainID uint  `gorm:"uniqueIndex:failedchainheight"`
	Chain        Chain `gorm:"foreignKey:BlockchainID"`
}

type Chain struct {
	ID      uint   `gorm:"primaryKey"`
	ChainID string `gorm:"uniqueIndex"` // e.g. osmosis-1
	Name    string // e.g. Osmosis
}

type Tx struct {
	ID              uint
	Hash            string
	Code            uint32
	BlockID         uint
	Block           Block
	SignerAddressID *int // *int allows foreign key to be null
	SignerAddress   Address
	Fees            []Fee
}

type Fee struct {
	ID             uint `gorm:"primaryKey"`
	TxID           uint
	Amount         decimal.Decimal `gorm:"type:decimal(78,0);"`
	DenominationID uint
	Denomination   Denom   `gorm:"foreignKey:DenominationID"`
	PayerAddressID uint    `gorm:"index:idx_payer_addr"`
	PayerAddress   Address `gorm:"foreignKey:PayerAddressID"`
}

// dbTypes.Address{Address: currTx.FeePayer().String()}

type Address struct {
	ID      uint
	Address string `gorm:"uniqueIndex"`
}

type MessageType struct {
	ID          uint   `gorm:"primaryKey"`
	MessageType string `gorm:"uniqueIndex;not null"`
}

type Message struct {
	ID            uint
	TxID          uint
	Tx            Tx
	MessageTypeID uint `gorm:"foreignKey:MessageTypeID"`
	MessageType   MessageType
	MessageIndex  int
}

const (
	OsmosisRewardDistribution uint = iota
)

// An event does not necessarily need to be part of a Transaction. For example, Osmosis rewards.
// Events can happen on chain and generate tendermint ABCI events that do not show up in transactions.
type TaxableEvent struct {
	ID             uint
	Source         uint            // This will indicate what type of event occurred on chain. Currently, only used for Osmosis rewards.
	Amount         decimal.Decimal `gorm:"type:decimal(78,0);"` // 2^256 or 78 digits, cosmos Int can be up to this length
	DenominationID uint
	Denomination   Denom   `gorm:"foreignKey:DenominationID"`
	AddressID      uint    `gorm:"index:idx_addr"`
	EventAddress   Address `gorm:"foreignKey:AddressID"`
	EventHash      string
	BlockID        uint
	Block          Block `gorm:"foreignKey:BlockID"`
}

// type SimpleDenom struct {
// 	ID     uint
// 	Denom  string `gorm:"uniqueIndex:denom_idx"`
// 	Symbol string `gorm:"uniqueIndex:denom_idx"`
// }

func (TaxableEvent) TableName() string {
	return "taxable_event"
}

type TaxableTransaction struct {
	ID                     uint
	MessageID              uint
	Message                Message         `gorm:"foreignKey:MessageID"`
	AmountSent             decimal.Decimal `gorm:"type:decimal(78,0);"`
	AmountReceived         decimal.Decimal `gorm:"type:decimal(78,0);"`
	DenominationSentID     *uint
	DenominationSent       Denom `gorm:"foreignKey:DenominationSentID"`
	DenominationReceivedID *uint
	DenominationReceived   Denom `gorm:"foreignKey:DenominationReceivedID"`
	SenderAddressID        *uint `gorm:"index:idx_sender"`
	SenderAddress          Address
	ReceiverAddressID      *uint `gorm:"index:idx_receiver"`
	ReceiverAddress        Address
}

func (TaxableTransaction) TableName() string {
	return "taxable_tx" // Legacy
}

type Denom struct {
	ID     uint
	Base   string `gorm:"uniqueIndex"`
	Name   string
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
	DenomUnitID uint
	DenomUnit   DenomUnit
	Alias       string `gorm:"unique"`
}

// Store transactions with their messages for easy database creation
type TxDBWrapper struct {
	Tx            Tx
	SignerAddress Address
	Messages      []MessageDBWrapper
}

// Store messages with their taxable events for easy database creation
type MessageDBWrapper struct {
	Message    Message
	TaxableTxs []TaxableTxDBWrapper
}

// Store taxable tx with their sender/receiver address for easy database creation
type TaxableTxDBWrapper struct {
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
