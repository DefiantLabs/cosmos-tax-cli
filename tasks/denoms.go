package tasks

import (
	"fmt"

	dbTypes "github.com/DefiantLabs/cosmos-exporter/db"
	"github.com/DefiantLabs/cosmos-exporter/rest"

	"gorm.io/gorm"
)

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
