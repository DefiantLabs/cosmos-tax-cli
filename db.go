package main

import (
	"fmt"

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
		&Address{},
		&Message{},
		&TaxableEvent{},
	)
}

func GetHighestIndexedBlock(db *gorm.DB) Block {
	var block Block
	//this can potentially be optimized by getting max first and selecting it (this gets translated into a select * limit 1)
	db.Table("blocks").Order("height desc").First(&block)
	return block
}

func IndexNewBlock(db *gorm.DB, block Block, txs []TxWithAddress) error {
	// return any error will rollback
	return db.Transaction(func(dbTransaction *gorm.DB) error {
		if err := dbTransaction.Create(&block).Error; err != nil {
			return err
		}

		for _, transaction := range txs {
			//viewing gorm logs shows this gets translated into a single ON CONFLICT DO NOTHING RETURNING "id"
			if err := dbTransaction.Where(&transaction.SignerAddress).FirstOrCreate(&transaction.SignerAddress).Error; err != nil {
				return err
			}

			transaction.Tx.SignerAddress = transaction.SignerAddress
			transaction.Tx.Block = block

			if err := dbTransaction.Create(&transaction.Tx).Error; err != nil {
				return err
			}

		}

		return nil
	})
}
