package main

import (
	"fmt"

	dbTypes "cosmos-exporter/db"

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
		&dbTypes.Block{},
		&dbTypes.Tx{},
		&dbTypes.Address{},
		&dbTypes.Message{},
		&dbTypes.TaxableEvent{},
	)
}

func GetHighestIndexedBlock(db *gorm.DB) dbTypes.Block {
	var block dbTypes.Block
	//this can potentially be optimized by getting max first and selecting it (this gets translated into a select * limit 1)
	db.Table("blocks").Order("height desc").First(&block)
	return block
}

func IndexNewBlock(db *gorm.DB, block dbTypes.Block, txs []dbTypes.TxDBWrapper) error {

	//consider optimizing the transaction, but how? Ordering matters due to foreign key constraints
	//Order required: Block -> (For each Tx: Signer Address -> Tx -> (For each Message: Message -> Taxable Events))
	//Also, foreign key relations are struct value based so create needs to be called first to get right foreign key ID
	return db.Transaction(func(dbTransaction *gorm.DB) error {
		// return any error will rollback
		if err := dbTransaction.Create(&block).Error; err != nil {
			return err
		}

		for _, transaction := range txs {

			if transaction.SignerAddress.Address != "" {
				//viewing gorm logs shows this gets translated into a single ON CONFLICT DO NOTHING RETURNING "id"
				if err := dbTransaction.Where(&transaction.SignerAddress).FirstOrCreate(&transaction.SignerAddress).Error; err != nil {
					return err
				}
				//store created db model in signer address, creates foreign key relation
				transaction.Tx.SignerAddress = transaction.SignerAddress
			} else {
				//store null foreign key relation in signer address id
				//This should never happen and indicates an error somewhere in parsing
				//Consider removing?
				transaction.Tx.SignerAddressId = nil
			}

			transaction.Tx.Block = block

			if err := dbTransaction.Create(&transaction.Tx).Error; err != nil {
				return err
			}

			for _, message := range transaction.Messages {
				message.Message.Tx = transaction.Tx
				if err := dbTransaction.Create(&message.Message).Error; err != nil {
					return err
				}

				for _, taxableEvent := range message.TaxableEvents {

					if taxableEvent.SenderAddress.Address != "" {
						if err := dbTransaction.Where(&taxableEvent.SenderAddress).FirstOrCreate(&taxableEvent.SenderAddress).Error; err != nil {
							return err
						}
						//store created db model in sender address, creates foreign key relation
						taxableEvent.TaxableEvent.SenderAddress = taxableEvent.SenderAddress
					} else {
						//nil creates null foreign key relation
						taxableEvent.TaxableEvent.SenderAddressId = nil
					}

					if taxableEvent.ReceiverAddress.Address != "" {
						if err := dbTransaction.Where(&taxableEvent.ReceiverAddress).FirstOrCreate(&taxableEvent.ReceiverAddress).Error; err != nil {
							return err
						}
						//store created db model in receiver address, creates foreign key relation
						taxableEvent.TaxableEvent.ReceiverAddress = taxableEvent.ReceiverAddress
					} else {
						//nil creates null foreign key relation
						taxableEvent.TaxableEvent.ReceiverAddressId = nil
					}
					taxableEvent.TaxableEvent.Message = message.Message
					if err := dbTransaction.Create(&taxableEvent.TaxableEvent).Error; err != nil {
						return err
					}
				}
			}

		}

		return nil
	})
}
