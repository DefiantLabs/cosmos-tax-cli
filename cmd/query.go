package cmd

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
	"github.com/DefiantLabs/cosmos-tax-cli-private/csv"
	parsers_pkg "github.com/DefiantLabs/cosmos-tax-cli-private/csv/parsers"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Queries the currently indexed data.",
	Long: `Queries the indexed data according to a configuration. Mainly address based. Apply
	your address to the command and a CSV export with your data for your address will be generated.`,
	// If we want to pass errors up to the
	Run: func(cmd *cobra.Command, args []string) {
		found := false
		parsers := parsers_pkg.GetParserKeys()
		for _, v := range parsers {
			if v == format {
				found = true
				break
			}
		}

		if !found {
			err := cmd.Help()
			if err != nil {
				config.Log.Error("Error getting cmd help.", zap.Error(err))
			}
			config.Log.Fatal(fmt.Sprintf("Invalid format %s, valid formats are %s", format, parsers))
		}

		// TODO: split out setup methods and only call necessary ones
		_, db, _, err := setup(conf)
		if err != nil {
			config.Log.Fatal("Error setting up query", zap.Error(err))
		}

		// Validate addresses
		for _, address := range addresses {
			if strings.Contains(address, ",") {
				throwValidationErr(cmd, fmt.Sprintf("Invalid address '%v'. Addresses cannot contain commas", address))
			} else if strings.Contains(address, " ") {
				throwValidationErr(cmd, fmt.Sprintf("Invalid address '%v'. Addresses cannot contain spaces", address))
			}
		}

		// Validate and set dates
		var startDate *time.Time
		var endDate *time.Time
		expectedLayout := "2006-01-02:15:04:05"
		if startDateStr != "" {
			parsedDate, err := time.Parse(expectedLayout, startDateStr)
			if err != nil {
				throwValidationErr(cmd, fmt.Sprintf("Invalid start date '%v'.", startDateStr))
			}
			startDate = &parsedDate
		}
		if endDateStr != "" {
			parsedDate, err := time.Parse(expectedLayout, endDateStr)
			if err != nil {
				throwValidationErr(cmd, fmt.Sprintf("Invalid end date '%v'.", endDateStr))

			}
			endDate = &parsedDate
		}

		// Get data for each address
		var headers []string
		var csvRows []parsers_pkg.CsvRow
		for _, address := range addresses {
			var addressRows []parsers_pkg.CsvRow

			addressRows, headers, err = csv.ParseForAddress(address, startDate, endDate, db, format, conf)
			if err != nil {
				log.Println(address)
				config.Log.Fatal("Error calling parser for address", zap.Error(err))
			}

			csvRows = append(csvRows, addressRows...)
		}

		// re-sort rows if needed
		if len(addresses) > 1 {
			sortRows(csvRows, format)
		}

		buffer := csv.ToCsv(csvRows, headers)
		err = os.WriteFile(output, buffer.Bytes(), 0600)
		if err != nil {
			config.Log.Fatal("Error writing out CSV", zap.Error(err))
		}
	},
}

func throwValidationErr(cmd *cobra.Command, cause string) {
	err := cmd.Help()
	if err != nil {
		config.Log.Error("Error getting cmd help.", zap.Error(err))
	}
	config.Log.Fatal(cause)
}

// TODO: Figure out a way to make this cleaner by moving it into a function on the parse itself
func sortRows(csvRows []parsers_pkg.CsvRow, format string) {
	// set the appropriate time format for the parser
	timeLayout := "2006-01-02 15:04:05"
	if format == "accointing" {
		timeLayout = "01/02/2006 15:04:05"
	}
	// Sort by date
	sort.Slice(csvRows, func(i int, j int) bool {
		leftDate, err := time.Parse(timeLayout, csvRows[i].GetDate())
		if err != nil {
			config.Log.Error("Error sorting left date.", zap.Error(err))
			return false
		}
		rightDate, err := time.Parse(timeLayout, csvRows[j].GetDate())
		if err != nil {
			config.Log.Error("Error sorting right date.", zap.Error(err))
			return false
		}
		return leftDate.Before(rightDate)
	})
}

var (
	addresses    []string // flag storage for the addresses to query on
	output       string   // flag storage for the output file location
	format       string   // flag storage for the output format
	startDateStr string
	endDateStr   string
)

func init() {
	// Setup Logger
	logLevel := conf.Log.Level
	logPath := conf.Log.Path
	config.DoConfigureLogger(logPath, logLevel)

	validFormats := parsers_pkg.GetParserKeys()
	if len(validFormats) == 0 {
		config.Log.Fatal("Error during initialization, no CSV parsers found.")
	}

	queryCmd.Flags().StringSliceVar(&addresses, "address", nil, "A comma separated list of the address(s) to query. (Both '--address addr1,addr2' and '--address addr1 --address addr2' are valid)")
	err := queryCmd.MarkFlagRequired("address")
	if err != nil {
		config.Log.Fatal("Error marking address field as required during query init. Err: ", zap.Error(err))
	}

	queryCmd.Flags().StringVar(&output, "output", "./output.csv", "The output location")
	queryCmd.Flags().StringVar(&format, "format", validFormats[0], "The format to output")

	// date range
	queryCmd.Flags().StringVar(&startDateStr, "start-date", "", "If set, tx before this date will be ignored. (Dates must be specified in the format 'YYYY-MM-DD:HH:MM:SS' in UTC)")
	queryCmd.Flags().StringVar(&endDateStr, "end-date", "", "If set, tx on or after this date will be ignored. (Dates must be specified in the format 'YYYY-MM-DD:HH:MM:SS' in UTC)")

	rootCmd.AddCommand(queryCmd)
}
