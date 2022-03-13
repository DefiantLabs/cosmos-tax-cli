package db

import "gorm.io/gorm"

func GetTaxableEvents(addressList []string, db *gorm.DB) ([]TaxableEvent, error) {
	//Look up all TaxableEvents, Transactions, and Messages for the addresses
	var taxableEvents []TaxableEvent

	result := db.Joins("JOIN addresses ON addresses.id = taxable_events.sender_address_id OR addresses.id = taxable_events.receiver_address_id").
		Where("addresses.address IN ?", addressList).Find(&taxableEvents)

	return taxableEvents, result.Error
}
