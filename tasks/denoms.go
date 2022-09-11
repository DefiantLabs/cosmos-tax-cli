package tasks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	dbTypes "github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/rest"
	"go.uber.org/zap"

	"gorm.io/gorm"
)

type OsmosisAssets struct {
	Assets []Asset
}

type Asset struct {
	Denoms []DenomUnit `json:"denom_units"`
	Symbol string
	Base   string
	Name   string
}

type DenomUnit struct {
	Denom    string
	Exponent int
	Aliases  []string
}

func DoChainSpecificUpsertDenoms(db *gorm.DB, chain string) {
	switch chain {
	case "osmosis-1":
		UpsertOsmosisDenoms(db)
	}
	//may want to move this elsewhere, or eliminate entirely
	//I would prefer we just grab the denoms when needed always
	//Current problem: we use the denom cache in various blocks later on
	dbTypes.CacheDenoms(db)
}

func UpsertOsmosisDenoms(db *gorm.DB) {

	url := "https://raw.githubusercontent.com/osmosis-labs/assetlists/main/osmosis-1/osmosis-1.assetlist.json"
	frontierUrl := "https://raw.githubusercontent.com/osmosis-labs/assetlists/main/osmosis-1/osmosis-frontier.assetlist.json"

	denomAssets, err := getOsmosisAssetsList(url)
	if err != nil {
		config.Logger.Error("Download Osmosis Denom Metadata", zap.Error(err))
		os.Exit(1)
	} else {
		denoms := toDenoms(denomAssets)
		err = dbTypes.UpsertDenoms(db, denoms)
		if err != nil {
			config.Logger.Error("Upsert Osmosis Denom Metadata", zap.Error(err))
			os.Exit(1)
		}
	}

	frontierDenomAssets, err := getOsmosisAssetsList(frontierUrl)
	if err != nil {
		config.Logger.Error("Download Osmosis Frontier Denom Metadata", zap.Error(err))
		os.Exit(1)
	} else {
		denoms := toDenoms(frontierDenomAssets)
		err = dbTypes.UpsertDenoms(db, denoms)
		if err != nil {
			config.Logger.Error("Upsert Osmosis Frontier Denom Metadata", zap.Error(err))
			os.Exit(1)
		}
	}
}

func toDenoms(assets *OsmosisAssets) []dbTypes.DenomDBWrapper {
	var denoms []dbTypes.DenomDBWrapper = make([]dbTypes.DenomDBWrapper, len(assets.Assets))
	for i, asset := range assets.Assets {
		denoms[i].Denom = dbTypes.Denom{Base: asset.Base, Name: asset.Name, Symbol: asset.Symbol}
		denoms[i].DenomUnits = make([]dbTypes.DenomUnitDBWrapper, len(asset.Denoms))

		for ii, denomUnit := range asset.Denoms {
			denoms[i].DenomUnits[ii].DenomUnit = dbTypes.DenomUnit{Exponent: uint(denomUnit.Exponent), Name: denomUnit.Denom}
			denoms[i].DenomUnits[ii].Aliases = make([]dbTypes.DenomUnitAlias, len(denomUnit.Aliases))

			for iii, denomUnitAlias := range denomUnit.Aliases {
				denoms[i].DenomUnits[ii].Aliases[iii] = dbTypes.DenomUnitAlias{Alias: denomUnitAlias}
			}
		}
	}

	return denoms
}

func getOsmosisAssetsList(assetsUrl string) (*OsmosisAssets, error) {
	assets := &OsmosisAssets{}
	err := getJson(assetsUrl, assets)
	if err != nil {
		return nil, err
	}

	return assets, nil
}

func getJson(url string, target interface{}) error {
	var myClient = &http.Client{Timeout: 10 * time.Second}

	r, err := myClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

func DenomUpsertTask(apiHost string, db *gorm.DB) {

	fmt.Println("Task started for DenomUpsertTask")
	denomsMetadata, err := rest.GetDenomsMetadatas(apiHost)
	if err != nil {
		fmt.Println("Error in DenomUpsertTask when reaching out to the API")
		fmt.Println(err)
		return
	}

	var denoms []dbTypes.DenomDBWrapper = make([]dbTypes.DenomDBWrapper, len(denomsMetadata.Metadatas))
	for i, denom := range denomsMetadata.Metadatas {
		denoms[i].Denom = dbTypes.Denom{Base: denom.Base, Name: denom.Name, Symbol: denom.Symbol}

		denoms[i].DenomUnits = make([]dbTypes.DenomUnitDBWrapper, len(denom.DenomUnits))

		for ii, denomUnit := range denom.DenomUnits {
			denoms[i].DenomUnits[ii].DenomUnit = dbTypes.DenomUnit{Exponent: uint(denomUnit.Exponent), Name: denomUnit.Denom}

			denoms[i].DenomUnits[ii].Aliases = make([]dbTypes.DenomUnitAlias, len(denomUnit.Aliases))

			for iii, denomUnitAlias := range denomUnit.Aliases {
				denoms[i].DenomUnits[ii].Aliases[iii] = dbTypes.DenomUnitAlias{Alias: denomUnitAlias}
			}
		}
	}

	err = dbTypes.UpsertDenoms(db, denoms)
	if err != nil {
		fmt.Println("Error upserting in DenomUpsertTask")
		fmt.Println(err)
		return
	}
	fmt.Println("Task ended for DenomUpsertTask")
}
