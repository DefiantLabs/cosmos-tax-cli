package csv

import (
	"bytes"
	"encoding/csv"
	"log"

	"github.com/DefiantLabs/cosmos-tax-cli-private/csv/parsers"
)

// Create the CSV and write it to byte buffer
func ToCsv(rows []parsers.CsvRow, headers []string) bytes.Buffer {
	var b bytes.Buffer
	w := csv.NewWriter(&b)

	if err := w.Write(headers); err != nil {
		log.Fatalln("error writing header to csv:", err)
	}

	// write the accointing rows to the csv
	for _, row := range rows {
		csvForRow := row.GetRowForCsv()
		if err := w.Write(csvForRow); err != nil {
			log.Fatalln("error writing record to csv:", err)
		}
	}

	// Write any buffered data to the underlying writer (standard output).
	w.Flush()

	if err := w.Error(); err != nil {
		log.Fatal(err)
	}

	return b
}
