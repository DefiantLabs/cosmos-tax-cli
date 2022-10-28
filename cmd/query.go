package cmd

import (
	"fmt"
	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"go.uber.org/zap"
	"log"
	"os"

	"github.com/DefiantLabs/cosmos-tax-cli/csv"
	"github.com/DefiantLabs/cosmos-tax-cli/csv/parsers"
	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Queries the currently indexed data.",
	Long: `Queries the indexed data according to a configuration. Mainly address based. Apply
	your address to the command and a CSV export with your data for your address will be generated.`,
	//If we want to pass errors up to the
	Run: func(cmd *cobra.Command, args []string) {
		found := false
		parsers := parsers.GetParserKeys()
		for _, v := range parsers {
			if v == format {
				found = true
				break
			}
		}

		if !found {
			err := cmd.Help()
			log.Println("Error getting cmd help. Err: ", err)
			cobra.CheckErr(fmt.Sprintf("Invalid format %s, valid formats are %s", format, parsers))
		}

		//TODO: split out setup methods and only call necessary ones
		_, db, _, err := setup(conf)
		cobra.CheckErr(err)

		csvRows, headers, err := csv.ParseForAddress(address, db, format, conf)
		cobra.CheckErr(err)

		buffer := csv.ToCsv(csvRows, headers)

		err = os.WriteFile(output, buffer.Bytes(), 0644)
		cobra.CheckErr(err)
	},
}

var (
	address string //flag storage for the address to query on
	output  string //flag storage for the output file location
	format  string //flag storage for the output format
)

func init() {
	//Setup Logger
	logLevel := conf.Log.Level
	logPath := conf.Log.Path
	config.DoConfigureLogger(logPath, logLevel)

	validFormats := parsers.GetParserKeys()
	if len(validFormats) == 0 {
		config.Log.Fatal("Error during intialization, no CSV parsers found.")
	}

	queryCmd.Flags().StringVar(&address, "address", "", "The address to query for")
	err := queryCmd.MarkFlagRequired("address")
	if err != nil {
		config.Log.Fatal("Error marking address field as required during query init. Err: ", zap.Error(err))
	}
	queryCmd.Flags().StringVar(&output, "output", "./output.csv", "The output location")
	queryCmd.Flags().StringVar(&format, "format", validFormats[0], "The format to output")

	rootCmd.AddCommand(queryCmd)
}
