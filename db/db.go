package db

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

func GetAddresses(addressList []string, db *gorm.DB) ([]Address, error) {
	//Look up all DB Addresses that match the search
	var addresses []Address
	result := db.Where("address IN ?", addressList).Find(&addresses)
	fmt.Printf("Found %d addresses in the db\n", result.RowsAffected)
	if result.Error != nil {
		fmt.Printf("Error %s searching DB for addresses.\n", result.Error)
	}

	return addresses, result.Error
}

//PostgresDbConnect connects to the database according to the passed in parameters
func PostgresDbConnect(host string, port string, database string, user string, password string, level string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable", host, port, database, user, password)
	gormLogLevel := logger.Silent

	if level == "info" {
		gormLogLevel = logger.Info
	}
	return gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(gormLogLevel)})
}

//PostgresDbConnect connects to the database according to the passed in parameters
func PostgresDbConnectLogInfo(host string, port string, database string, user string, password string) (*gorm.DB, error) {

	dsn := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable", host, port, database, user, password)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Info)})
}

//MigrateModels runs the gorm automigrations with all the db models. This will migrate as needed and do nothing if nothing has changed.
func MigrateModels(db *gorm.DB) error {
	return db.AutoMigrate(
		&Block{},
		&Tx{},
		&Address{},
		&Message{},
		&TaxableEvent{},
		&Denom{},
		&DenomUnit{},
		&DenomUnitAlias{},
	)
}

func GetHighestIndexedBlock(db *gorm.DB) Block {
	var block Block
	//this can potentially be optimized by getting max first and selecting it (this gets translated into a select * limit 1)
	db.Table("blocks").Order("height desc").First(&block)
	return block
}

func IndexNewBlock(db *gorm.DB, block Block, txs []TxDBWrapper) error {

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

func UpsertDenoms(db *gorm.DB, denoms []DenomDBWrapper) error {

	return db.Transaction(func(dbTransaction *gorm.DB) error {

		for _, denom := range denoms {

			if err := dbTransaction.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}},
				DoUpdates: clause.AssignmentColumns([]string{"symbol", "base"}),
			}).Create(&denom.Denom).Error; err != nil {
				return err
			}

			for _, denomUnit := range denom.DenomUnits {
				denomUnit.DenomUnit.Denom = denom.Denom

				if err := dbTransaction.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "name"}},
					DoUpdates: clause.AssignmentColumns([]string{"exponent"}),
				}).Create(&denomUnit.DenomUnit).Error; err != nil {
					return err
				}

				for _, denomAlias := range denomUnit.Aliases {

					denomAlias.DenomUnit = denomUnit.DenomUnit
					if err := dbTransaction.Clauses(clause.OnConflict{
						DoNothing: true,
					}).Create(&denomAlias).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})

}
