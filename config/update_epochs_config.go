package config

import (
	"fmt"

	"github.com/spf13/cobra"
)

type UpdateEpochsConfig struct {
	Database database
	Lens     lens
	Base     updateEpochsBase
	Log      log
}

type updateEpochsBase struct {
	throttlingBase
	EpochIdentifier string `mapstructure:"epoch-identifier"`
}

func SetupUpdateEpochsSpecificFlags(conf *UpdateEpochsConfig, cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&conf.Base.EpochIdentifier, "base.epoch-identifier", "", "the epoch identifier to update")
}

func (conf *UpdateEpochsConfig) Validate() error {
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

	err = validateThrottlingConf(conf.Base.throttlingBase)

	if err != nil {
		return err
	}

	if conf.Base.EpochIdentifier == "" {
		return fmt.Errorf("epoch identifier must be set")
	}

	return nil
}
