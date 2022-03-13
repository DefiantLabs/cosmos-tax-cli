package csv

import (
	"bytes"
	"encoding/csv"
	"log"
)

func ToCsv(rows []AccointingRow) bytes.Buffer {
	// records := [][]string{
	// 	{"first_name", "last_name", "username"},
	// 	{"Rob", "Pike", "rob"},
	// 	{"Ken", "Thompson", "ken"},
	// 	{"Robert", "Griesemer", "gri"},
	// }

	headers := []string{"transactionType", "date", "inBuyAmount", "inBuyAsset", "outSellAmount", "outSellAsset",
		"feeAmount (optional)", "feeAsset (optional)", "classification (optional)", "operationId (optional)"}

	var b bytes.Buffer
	w := csv.NewWriter(&b)

	if err := w.Write(headers); err != nil {
		log.Fatalln("error writing record to csv:", err)
	}

	//TODO... write the accointing rows...

	// Write any buffered data to the underlying writer (standard output).
	w.Flush()

	if err := w.Error(); err != nil {
		log.Fatal(err)
	}

	return b
}
