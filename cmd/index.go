package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(indexCmd)
}

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Indexes the blockchain according to the configuration defined.",
	Long: `Indexes the Cosmos-based blockchain according to the configurations found on the command line
	or in the specified config file. Indexes taxable events into a database for easy querying. It is
	highly recommended to keep this command running as a background service to keep your index up to date.`,
	Run: func(cmd *cobra.Command, args []string) {
		//TODO: transition old main.go code to this subcommand
		//TODO: split out setup methods and only call necessary ones
		fmt.Println("Index stub")
		_, _, _, err := setup(conf)
		cobra.CheckErr(err)
	},
}
