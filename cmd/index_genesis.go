package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"

	"github.com/DefiantLabs/cosmos-indexer/config"
	"github.com/DefiantLabs/cosmos-indexer/core"
	dbTypes "github.com/DefiantLabs/cosmos-indexer/db"
	"github.com/spf13/cobra"
	tmtypes "github.com/tendermint/tendermint/types"
	"gorm.io/gorm"

	tmjson "github.com/tendermint/tendermint/libs/json"

	authModule "github.com/cosmos/cosmos-sdk/x/auth/types"
)

var (
	indexGenesisConfig       config.IndexGenesisConfig
	indexGenesisDbConnection *gorm.DB
)

func init() {
	config.SetupLogFlags(&indexGenesisConfig.Log, indexGenesisCmd)
	config.SetupDatabaseFlags(&indexGenesisConfig.Database, indexGenesisCmd)
	config.SetupLensFlags(&indexGenesisConfig.Lens, indexGenesisCmd)
	config.SetupIndexGenesisSpecificFlags(&indexGenesisConfig, indexGenesisCmd)
	rootCmd.AddCommand(indexGenesisCmd)
}

var indexGenesisCmd = &cobra.Command{
	Use:   "index-genesis",
	Short: "Indexes the blockchain genesis from either the RPC node or a downloaded genesis file.",
	Long: `Indexes the Cosmos-based blockchain according to the configurations found on the command line
	or in the specified config file. Indexes taxable events into a database for easy querying. It is
	highly recommended to keep this command running as a background service to keep your index up to date.`,
	PreRunE: setupIndexGenesis,
	Run:     indexGenesis,
}

func setupIndexGenesis(cmd *cobra.Command, args []string) error {
	bindFlags(cmd, viperConf)

	err := indexGenesisConfig.Validate()
	if err != nil {
		return err
	}

	// Logger
	logLevel := indexGenesisConfig.Log.Level
	logPath := indexGenesisConfig.Log.Path
	prettyLogging := indexGenesisConfig.Log.Pretty
	config.DoConfigureLogger(logPath, logLevel, prettyLogging)

	db, err := dbTypes.PostgresDbConnect(indexGenesisConfig.Database.Host, indexGenesisConfig.Database.Port, indexGenesisConfig.Database.Database,
		indexGenesisConfig.Database.User, indexGenesisConfig.Database.Password, strings.ToLower(indexGenesisConfig.Database.LogLevel))
	if err != nil {
		config.Log.Fatal("Could not establish connection to the database", err)
	}

	sqldb, _ := db.DB()
	sqldb.SetMaxIdleConns(10)
	sqldb.SetMaxOpenConns(100)
	sqldb.SetConnMaxLifetime(time.Hour)

	err = dbTypes.MigrateModels(db)
	if err != nil {
		config.Log.Error("Error running DB migrations", err)
		return err
	}

	core.SetupAddressRegex(indexer.cfg.Lens.AccountPrefix + "(valoper)?1[a-z0-9]{38}")
	core.SetupAddressPrefix(indexer.cfg.Lens.AccountPrefix)

	config.SetChainConfig(indexGenesisConfig.Lens.AccountPrefix)

	indexGenesisDbConnection = db

	return nil
}

func indexGenesis(cmd *cobra.Command, args []string) {
	// Since we are still stuck on Osmosis' version of the Cosmos SDK, we need to use the base Tendermint codebase for unmarsalling the Genesis file
	// Latest versions of Cosmos SDK provide wrappers for these types of operations in the genutil module
	// We also skip validations due to some testing where some chains had invalid Genesis files when passing items to tmtypes.GenesisDocFromJson which runs ValidateAndComplete
	var genDoc tmtypes.GenesisDoc
	if indexGenesisConfig.Base.GenesisFileLocation != "" {
		// open and read the Genesis File
		switch {
		case strings.HasSuffix(indexGenesisConfig.Base.GenesisFileLocation, ".tar.gz"):
			genDoc = getGenesisFromTar(indexGenesisConfig.Base.GenesisFileLocation)
		case strings.HasSuffix(indexGenesisConfig.Base.GenesisFileLocation, ".json"):
			genDoc = getGenesisFromFile(indexGenesisConfig.Base.GenesisFileLocation)
		default:
			config.Log.Fatal("Genesis file must be either a tar.gz or a json file")
		}
	}

	config.Log.Infof("Genesis file found for chain %s\n", genDoc.ChainID)

	var appState map[string]json.RawMessage
	if err := json.Unmarshal(genDoc.AppState, &appState); err != nil {
		config.Log.Fatal("Could not unmarshal app state", err)
	}

	// Pull accounts into account table from auth module app state
	client := config.GetLensClient(indexGenesisConfig.Lens)

	var authModuleState authModule.GenesisState
	err := client.Codec.Marshaler.UnmarshalJSON(appState[authModule.ModuleName], &authModuleState)
	if err != nil {
		config.Log.Fatal("Could not unmarshal auth module state", err)
	}

	authAccounts, err := authModule.UnpackAccounts(authModuleState.Accounts)
	if err != nil {
		config.Log.Fatal("Could not unpack auth module accounts", err)
	}

	var addresses []dbTypes.Address
	// create db entries for each account
	for _, genesisAccount := range authAccounts {
		address := dbTypes.Address{Address: genesisAccount.GetAddress().String()}

		// The genesisAccount.GetAddress function does not return an error.
		// The only way to know if it failed is to check for empty string.
		if address.Address == "" {
			config.Log.Fatal("Could not get address from account", err)
		}

		addresses = append(addresses, address)
	}

	err = dbTypes.UpsertAddresses(indexGenesisDbConnection, addresses)

	if err != nil {
		config.Log.Fatal("Could not create addresses from auth state", err)
	}
}

func getGenesisFromTar(tarfileLocation string) tmtypes.GenesisDoc {
	var genesisContents []byte
	genDoc := tmtypes.GenesisDoc{}
	reader, err := os.Open(indexGenesisConfig.Base.GenesisFileLocation)
	if err != nil {
		config.Log.Fatal("Error opening Genesis file", err)
	}
	defer reader.Close()
	genesisContents, err = extractGenesisFromTarGz(reader)

	if err != nil {
		config.Log.Fatal("Error extracting Genesis file", err)
	}

	err = tmjson.Unmarshal(genesisContents, &genDoc)

	if err != nil {
		config.Log.Fatal("Error unmarshalling Genesis file", err)
	}
	return genDoc
}

func getGenesisFromFile(jsonFileLocation string) tmtypes.GenesisDoc {
	var genesisContents []byte
	genDoc := tmtypes.GenesisDoc{}
	var err error
	genesisContents, err = os.ReadFile(indexGenesisConfig.Base.GenesisFileLocation)
	if err != nil {
		config.Log.Fatal("Error reading Genesis file", err)
	}
	err = tmjson.Unmarshal(genesisContents, &genDoc)

	if err != nil {
		config.Log.Fatal("Error reading Genesis file", err)
	}
	return genDoc
}

func extractGenesisFromTarGz(gzipStream io.Reader) ([]byte, error) {
	var out []byte
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return nil, err
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		// this function currently only returns the first entry in a tar, the use case we have for genesis files
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		if header.Typeflag == tar.TypeReg && strings.HasSuffix(header.Name, ".json") {
			config.Log.Infof("Found Genesis file in tar: %s", header.Name)
			// return entire file as string
			out, err = io.ReadAll(tarReader)
			if err != nil && err != io.EOF {
				return out, err
			}

			return out, nil

		}

	}
	return out, nil
}
