package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
	"github.com/DefiantLabs/cosmos-tax-cli-private/csv"
	parsers_pkg "github.com/DefiantLabs/cosmos-tax-cli-private/csv/parsers"

	"github.com/spf13/cobra"
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
				config.Log.Error("Error getting cmd help.", err)
			}
			config.Log.Fatal(fmt.Sprintf("Invalid format %s, valid formats are %s", format, parsers))
		}

		_, _, db, _, err := setup(conf)
		if err != nil {
			config.Log.Fatal("Error setting up query", err)
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

		csvRows, headers, err := csv.ParseForAddress(addresses, startDate, endDate, db, format, conf)
		if err != nil {
			log.Println(addresses)
			config.Log.Fatal("Error calling parser for address", err)
		}

		buffer := csv.ToCsv(csvRows, headers)
		fmt.Println(buffer.String())
	},
}

func throwValidationErr(cmd *cobra.Command, cause string) {
	err := cmd.Help()
	if err != nil {
		config.Log.Error("Error getting cmd help.", err)
	}
	config.Log.Fatal(cause)
}

var (
	addresses    []string // flag storage for the addresses to query on
	format       string   // flag storage for the output format
	startDateStr string
	endDateStr   string
)

func init() {
	validFormats := parsers_pkg.GetParserKeys()
	if len(validFormats) == 0 {
		config.Log.Fatal("Error during initialization, no CSV parsers found.")
	}

	queryCmd.Flags().StringSliceVar(&addresses, "address", nil, "A comma separated list of the address(s) to query. (Both '--address addr1,addr2' and '--address addr1 --address addr2' are valid)")
	err := queryCmd.MarkFlagRequired("address")
	if err != nil {
		config.Log.Fatal("Error marking address field as required during query init. Err: ", err)
	}

	queryCmd.Flags().StringVar(&format, "format", validFormats[0], "The format to output")

	// date range
	queryCmd.Flags().StringVar(&startDateStr, "start-date", "", "If set, tx before this date will be ignored. (Dates must be specified in the format 'YYYY-MM-DD:HH:MM:SS' in UTC)")
	queryCmd.Flags().StringVar(&endDateStr, "end-date", "", "If set, tx on or after this date will be ignored. (Dates must be specified in the format 'YYYY-MM-DD:HH:MM:SS' in UTC)")

	rootCmd.AddCommand(queryCmd)
}
