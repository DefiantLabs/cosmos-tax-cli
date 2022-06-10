package db

import "gorm.io/gorm"

func GetTaxableEvents(address string, db *gorm.DB) ([]TaxableTransaction, error) {
	//Look up all TaxableEvents, Transactions, and Messages for the addresses
	var taxableEvents []TaxableTransaction

	result := db.Joins("JOIN addresses ON addresses.id = taxable_tx.sender_address_id OR addresses.id = taxable_tx.receiver_address_id").
		Where("addresses.address = ?", address).Preload("Message").Preload("Message.Tx").Preload("Message.Tx.SignerAddress").
		Preload("SenderAddress").Preload("ReceiverAddress").Find(&taxableEvents)

	return taxableEvents, result.Error
}
