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
	)
}

func GetHighestIndexedBlock(db *gorm.DB) Block {
	var block Block
	//this can potentially be optimized by getting max first and selecting it (this gets translated into a select * limit 1)
	db.Table("blocks").Order("height desc").First(&block)
	return block
}

func IndexNewBlock(db *gorm.DB, block Block, txs []Tx, addresses [][]Address) error {
	// return any error will rollback
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&block).Error; err != nil {
			return err
		}

		for i, _ := range txs {
			for ii, _ := range addresses[i] {
				if err := db.Where(&addresses[i][ii]).FirstOrCreate(&addresses[i][ii]).Error; err != nil {
					return err
				}
			}

			txs[i].Block = block
			txs[i].Addresses = addresses[i]

		}

		if len(txs) != 0 {
			if err := tx.Create(txs).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
