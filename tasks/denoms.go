package tasks

import (
	"github.com/DefiantLabs/cosmos-tax-cli/config"
	dbTypes "github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/rest"
	"go.uber.org/zap"

	"gorm.io/gorm"
)

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
	//Stubbed until we add Osmosis compatibility
	// switch chain {
	// case osmosis.ChainID:
	// 	UpsertOsmosisDenoms(db)
	// }

	//may want to move this elsewhere, or eliminate entirely
	//I would prefer we just grab the denoms when needed always
	//Current problem: we use the denom cache in various blocks later on
	dbTypes.CacheDenoms(db)
}

func UpsertOsmosisDenoms(db *gorm.DB) {
	//Stubbed for now
}

func DenomUpsertTask(apiHost string, db *gorm.DB) {
	config.Log.Debug("Task started for DenomUpsertTask")
	denomsMetadata, err := rest.GetDenomsMetadatas(apiHost)
	if err != nil {
		config.Log.Error("Error in DenomUpsertTask when reaching out to the API. ", zap.Error(err))
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
		config.Log.Error("Error upserting in DenomUpsertTask", zap.Error(err))
		return
	}
	config.Log.Info("Task ended for DenomUpsertTask")
}
