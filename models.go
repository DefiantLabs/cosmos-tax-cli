package main

import "time"

type Blocks struct {
	ID     uint
	Height uint64
}

type Txs struct {
	ID        uint
	TimeStamp time.Time
	Hash      string
	Fees      string
	//foreign key relation between blocks and txs
	BlocksId int
	Blocks   Blocks
}
