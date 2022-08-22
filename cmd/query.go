package cmd

import (
	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Queries the currently indexed data.",
	Long: `Queries the indexed data according to a configuration. Mainly address based. Apply
	your address to the command and a CSV export with your data for your address will be generated.`,
	Run: func(cmd *cobra.Command, args []string) {
		//TODO: transition old rest API querying methods to this subcommand
		//TODO: split out setup methods and only call necessary ones
		_, _, _, err := setup(conf)
		cobra.CheckErr(err)
	},
}

var (
	address string //flag storage for the address to query on
	output  string //flag storage for the output file location
)

func init() {
	queryCmd.Flags().StringVar(&address, "address", "", "The address to query for")
	queryCmd.MarkFlagRequired("address")
	queryCmd.Flags().StringVar(&output, "output", "./output.csv", "The output location")

	rootCmd.AddCommand(queryCmd)
}
