package test

import (
	"os"
	"testing"

	"github.com/DefiantLabs/cosmos-exporter/csv"
	"github.com/DefiantLabs/cosmos-exporter/db"
)

func TestCsvForAddress(t *testing.T) {
	addressRegex := "juno(valoper)?1[a-z0-9]{38}"
	addressPrefix := "juno"
	gorm, _ := db_setup(addressRegex, addressPrefix)
	//address := "juno1mt72y3jny20456k247tc5gf2dnat76l4ynvqwl"
	//address := "juno130mdu9a0etmeuw52qfxk73pn0ga6gawk4k539x" //strangelove's delegator
	address := "juno1m2hg5t7n8f6kzh8kmh98phenk8a4xp5wyuz34y" //local test key address
	csvRows, err := csv.ParseForAddress(address, gorm)
	if err != nil || len(csvRows) == 0 {
		t.Fatal("Failed to lookup taxable events")
	}

	buffer := csv.ToCsv(csvRows)
	if len(buffer.Bytes()) == 0 {
		t.Fatal("CSV length should never be 0, there are always headers!")
	}

	err = os.WriteFile("accointing.csv", buffer.Bytes(), 0644)
	if err != nil {
		t.Fatal("Failed to write CSV to disk")
	}
}

func TestLookupTxForAddresses(t *testing.T) {
	addressRegex := "juno(valoper)?1[a-z0-9]{38}"
	addressPrefix := "juno"
	gorm, _ := db_setup(addressRegex, addressPrefix)
	//"juno1txpxafd7q96nkj5jxnt7qnqy4l0rrjyuv6dgte"
	//juno1mt72y3jny20456k247tc5gf2dnat76l4ynvqwl
	taxableEvts, err := db.GetTaxableTransactions("juno1txpxafd7q96nkj5jxnt7qnqy4l0rrjyuv6dgte", gorm)
	if err != nil || len(taxableEvts) == 0 {
		t.Fatal("Failed to lookup taxable events")
	}
}
