package config

import (
	"os"

	"github.com/spf13/cobra"
)

type IndexGenesisConfig struct {
	Database           database
	ConfigFileLocation string
	Base               indexGenesisBase
	Log                log
	Lens               lens
}

type indexGenesisBase struct {
	throttlingBase
	retryBase
	GenesisFileLocation string `mapstructure:"genesis-file-location"`
}

func SetupIndexGenesisSpecificFlags(conf *IndexGenesisConfig, cmd *cobra.Command) {
	// genesis indexing
	cmd.PersistentFlags().StringVar(&conf.Base.GenesisFileLocation, "base.genesis-file-location", "", "A file location containing a Gensis file. Can be raw JSON or a tar.gz file. The command will skip reaching out to the RPC node if this is set.")
}

func (conf *IndexGenesisConfig) Validate() error {
	err := validateDatabaseConf(conf.Database)
	if err != nil {
		return err
	}

	// Lens setup required if Genesis file not provided
	if conf.Base.GenesisFileLocation == "" {

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
	} else {
		if _, err := os.Stat(conf.Base.GenesisFileLocation); err != nil {
			return err
		}
	}

	return nil
}
