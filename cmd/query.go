package cmd

import (
	"fmt"
	"log"
	"os"
	"sort"
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
			config.Log.Error("Error getting cmd help.", zap.Error(err))
			config.Log.Fatal(fmt.Sprintf("Invalid format %s, valid formats are %s", format, parsers))
		}

		// validate that 1 or more address have been provided
		if len(address) == 0 && len(addresses) == 0 {
			cmd.Help() //nolint:errcheck
			config.Log.Fatal("One or more addresses must be provided")
		}
		// validate that addresses were not provided in more than one way
		if len(address) > 0 && len(addresses) > 0 {
			cmd.Help() //nolint:errcheck
			config.Log.Fatal("If you would like to specify multiple addresses, please use the 'addresses' flag")
		}
		if len(addresses) == 0 {
			addresses = []string{address}
		}

		// TODO: split out setup methods and only call necessary ones
		_, db, _, err := setup(conf)
		if err != nil {
			config.Log.Fatal("Error setting up query", zap.Error(err))
		}

		var headers []string
		var csvRows []parsers_pkg.CsvRow
		for _, address := range addresses {
			var addressRows []parsers_pkg.CsvRow

			addressRows, headers, err = csv.ParseForAddress(address, db, format, conf)
			if err != nil {
				log.Println(address)
				config.Log.Fatal("Error calling parser for address", zap.Error(err))
			}
			csvRows = append(csvRows, addressRows...)
		}

		// re-sort rows if needed
		if len(addresses) > 1 {
			// set the appropriate time format for the parser
			timeLayout := "2006-01-02 15:04:05"
			timeCol := 0
			if format == "accointing" {
				timeLayout = "01/02/2006 15:04:05"
				timeCol = 1
			}
			// Sort by date
			sort.Slice(csvRows, func(i int, j int) bool {
				leftDate, err := time.Parse(timeLayout, csvRows[i].GetRowForCsv()[timeCol])
				if err != nil {
					config.Log.Error("Error sorting left date.", zap.Error(err))
					return false
				}
				rightDate, err := time.Parse(timeLayout, csvRows[j].GetRowForCsv()[timeCol])
				if err != nil {
					config.Log.Error("Error sorting right date.", zap.Error(err))
					return false
				}
				return leftDate.Before(rightDate)
			})
		}

		buffer := csv.ToCsv(csvRows, headers)
		err = os.WriteFile(output, buffer.Bytes(), 0600)
		if err != nil {
			config.Log.Fatal("Error writing out CSV", zap.Error(err))
		}
	},
}

var (
	address   string   // flag storage for the address to query on
	addresses []string // flag storage for the addresses to query on
	output    string   // flag storage for the output file location
	format    string   // flag storage for the output format
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

	queryCmd.Flags().StringVar(&address, "address", "", "The address to query for")
	queryCmd.Flags().StringSliceVar(&addresses, "addresses", []string{}, "The addresses to query for")

	queryCmd.Flags().StringVar(&output, "output", "./output.csv", "The output location")
	queryCmd.Flags().StringVar(&format, "format", validFormats[0], "The format to output")

	rootCmd.AddCommand(queryCmd)
}
