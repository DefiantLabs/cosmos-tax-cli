package main

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

//PostgresDbConnect connects to the database according to the passed in parameters
func PostgresDbConnect(host string, port string, database string, user string, password string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable", host, port, database, user, password)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

//MigrateModels runs the gorm automigrations with all the db models. This will migrate as needed and do nothing if nothing has changed.
func MigrateModels(db *gorm.DB) error {
	return db.AutoMigrate(
		&Block{},
		&Tx{},
	)
}

func GetHighestBlock(db *gorm.DB) Block {
	var block Block
	//this can potentially be optimized by getting max first and selecting it (this gets translated into a select * limit 1)
	db.Table("blocks").Order("height desc").First(&block)
	return block
}

func IndexNewTx(db *gorm.DB, tx SingleTx, block Block) {
	timeStamp, _ := time.Parse(time.RFC3339, tx.TxResponse.TimeStamp)

	var fees string = ""

	//can be multiple fees, make comma delimited list of fees
	//should consider separate table?
	numFees := len(tx.Tx.AuthInfo.TxFee.TxFeeAmount)
	for i, fee := range tx.Tx.AuthInfo.TxFee.TxFeeAmount {
		newFee := fmt.Sprintf("%s%s", fee.Amount, fee.Denom)
		if i+1 != numFees {
			newFee = newFee + ","
		}
		fees = fees + newFee
	}

	newTx := Tx{TimeStamp: timeStamp, Hash: tx.TxResponse.TxHash, Fees: fees, Blocks: block}
	db.Create(&newTx)
}
