package db

import (
	"gorm.io/gorm"
)

func GetTaxableTransactions(address string, db *gorm.DB) ([]TaxableTransaction, error) {
	//Look up all Transactions, and Messages for the addresses
	var taxableEvents []TaxableTransaction

	result := db.Joins("JOIN addresses ON addresses.id = taxable_tx.sender_address_id OR addresses.id = taxable_tx.receiver_address_id").
		Where("addresses.address = ?", address).
		Preload("Message").Preload("Message.Tx").Preload("Message.Tx.SignerAddress").Preload("Message.Tx.Fees").Preload("Message.Tx.Fees.Denomination").Preload("Message.Tx.Fees.PayerAddress").
		Preload("SenderAddress").Preload("ReceiverAddress").Preload("DenominationSent").Preload("DenominationReceived").Find(&taxableEvents)

	return taxableEvents, result.Error
}

func GetTaxableEvents(address string, db *gorm.DB) ([]TaxableEvent, error) {
	//Look up all TaxableEvents for the addresses
	var taxableEvents []TaxableEvent

	result := db.Joins("JOIN addresses ON addresses.id = taxable_event.address_id").
		Where("addresses.address = ?", address).Preload("EventAddress").Preload("Denomination").
		Preload("Block").Preload("Block.Chain").Find(&taxableEvents)

	return taxableEvents, result.Error
}
