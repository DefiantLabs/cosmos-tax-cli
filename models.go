package main

import "time"

type BlockModel struct {
	ID     uint
	Height uint64
}

type TxModel struct {
	ID        uint
	TimeStamp time.Time
	Hash      string
	FeeDenom  string
	FeeAmount string
}
