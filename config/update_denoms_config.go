package config

import "github.com/spf13/cobra"

type UpdateDenomsConfig struct {
	Database database
	Lens     lens
	Log      log
	Base     updateDenomsBase
}

type updateDenomsBase struct {
	UpdateAll bool `mapstructure:"update-all"`
}

func SetupUpdateDenomsSpecificFlags(conf *UpdateDenomsConfig, cmd *cobra.Command) {
	cmd.Flags().BoolVar(&conf.Base.UpdateAll, "base.update-all", false, "If provided, the update script will ignore the config chain-id and update all denoms by reaching out to all assetlists supported.")
}

func (conf *UpdateDenomsConfig) Validate() error {

	err := validateDatabaseConf(conf.Database)

	if err != nil {
		return err
	}

	lensConf := conf.Lens

	lensConf, err = validateLensConf(lensConf)

	if err != nil {
		return err
	}

	conf.Lens = lensConf

	return nil
}
