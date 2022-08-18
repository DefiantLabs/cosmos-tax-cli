package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(queryCmd)
}

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Queries the currently indexed data.",
	Long: `Queries the indexed data according to a configuration. Mainly addressed based. Apply
	your address to the command and a CSV export with your data for your address will be generated.`,
	Run: func(cmd *cobra.Command, args []string) {
		//TODO: transition old rest API querying methods to this subcommand
		fmt.Println("Query stub")
	},
}
