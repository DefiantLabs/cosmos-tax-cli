package csv

import (
	"bytes"
	"encoding/csv"
	"log"
)

var headers = []string{"transactionType", "date", "inBuyAmount", "inBuyAsset", "outSellAmount", "outSellAsset",
	"feeAmount (optional)", "feeAsset (optional)", "classification (optional)", "operationId (optional)"}

//RowToCsv: Build a single row of data in the format expected by 'headers'
func RowToCsv(row AccointingRow) []string {

	inAmt := row.InBuyAmount

	//TODO: Somewhere in the last few months, row.InBuyAmount is getting turned into a string
	//Is this really what we wanted to do?
	// if row.InBuyAsset != "" {
	// 	//inAmt = strconv.FormatFloat(row.InBuyAmount, 'E', -1, 64)
	// 	inAmt = fmt.Sprintf("%s", row.InBuyAmount)
	// }

	outAmt := row.OutSellAmount
	//TODO: Somewhere in the last few months, row.OutSellAmount is getting turned into a string
	//Is this really what we wanted to do?
	// if row.OutSellAsset != "" {
	// 	//outAmt = strconv.FormatFloat(row.OutSellAmount, 'E', -1, 64)
	// 	outAmt = fmt.Sprintf("%f", row.OutSellAmount)
	// }

	feeAmt := row.FeeAmount
	//TODO: Somewhere in the last few months, row.FeeAmount is getting turned into a string
	//Is this really what we wanted to do?
	// if row.FeeAsset != "" {
	// 	//feeAmt = strconv.FormatFloat(row.FeeAmount, 'E', -1, 64)
	// 	feeAmt = fmt.Sprintf("%f", row.FeeAmount)
	// }

	return []string{
		row.TransactionType.String(),
		row.Date,
		inAmt,
		row.InBuyAsset,
		outAmt,
		row.OutSellAsset,
		feeAmt,
		row.FeeAsset,
		row.Classification.String(),
		row.OperationId,
	}
}

//Create the CSV and write it to byte buffer
func ToCsv(rows []AccointingRow) bytes.Buffer {
	var b bytes.Buffer
	w := csv.NewWriter(&b)

	if err := w.Write(headers); err != nil {
		log.Fatalln("error writing header to csv:", err)
	}

	//write the accointing rows to the csv
	for _, row := range rows {
		csvForRow := RowToCsv(row)
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
