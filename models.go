package main

import "time"

type Block struct {
	ID     uint
	Height uint64
}

type Tx struct {
	ID        uint
	TimeStamp time.Time
	Hash      string
	Fees      string
	//foreign key relation between blocks and txs
	BlocksId int
	Blocks   Block
}
