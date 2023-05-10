package cmd

import (
	"log"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/tasks"
	"github.com/spf13/cobra"
)

var updateAll bool

func init() {
	updateDenomsCmd.Flags().BoolVar(&updateAll, "update-all", false, "If provided, the update script will ignore the config chain-id and update all denoms by reaching out to all assetlists supported.")
	rootCmd.AddCommand(updateDenomsCmd)
}

var updateDenomsCmd = &cobra.Command{
	Use:   "update-denoms",
	Short: "Reach out to various assetlist locations to update the database with vetted denom information.",
	Long: `Reaches out to various Cosmos Denom assetlist registries and updates the values found in the database.
	Cosmos developers provide assetlists in a relatively standardized format (examples found for specific chains here https://github.com/cosmos/chain-registry).
	This command will prepopulate the Cosmos Tax CLI database with values found in regsitries for the specific chains we provide support for.
	It will either use the chain-id specified in the application configuration to update the specific assetlist, or update-all if provided.
	`,
	Run: updateDenoms,
}

func updateDenoms(cmd *cobra.Command, args []string) {
	cfg, _, db, _, err := setup(conf)
	if err != nil {
		log.Fatalf("Error during application setup. Err: %v", err)
	}

	switch {
	case updateAll:
		config.Log.Infof("Running denom update task for all supported chains")
		for chainID, function := range tasks.ChainSpecificDenomUpsertFunctions {
			config.Log.Infof("Running denom update task for chain %s", chainID)
			function(db)
		}
	case cfg.Lens.ChainID != "":
		function, ok := tasks.ChainSpecificDenomUpsertFunctions[cfg.Lens.ChainID]
		if ok {
			config.Log.Infof("Running denom update task for chain %s found in config", cfg.Lens.ChainID)
			function(db)
			config.Log.Info("Done")
		} else {
			config.Log.Fatalf("No denom update functionality for chain-id %s", cfg.Lens.ChainID)
		}
	default:
		config.Log.Fatal("Please pass the flag --update-all or provide a chain-id in your application configuration")
	}

	tasks.ValidateDenoms(db)
}
