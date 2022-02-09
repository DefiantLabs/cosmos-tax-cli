package main

import "time"

type Block struct {
	ID     uint
	Height uint64 `gorm:"uniqueIndex"`
}

type Tx struct {
	ID        uint
	TimeStamp time.Time
	Hash      string `gorm:"uniqueIndex"`
	Fees      string
	Code      int64
	//foreign key relation between blocks and txs
	BlockId int
	Block   Block
	//Many to Many relation for multiple addresses found in tx
	Addresses []Address `gorm:"many2many:tx_addresses;"`
}

type Address struct {
	ID      uint
	Address string `gorm:"uniqueIndex"`
}
