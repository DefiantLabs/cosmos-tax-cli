package db

import (
	"errors"
	"math"

	"gorm.io/gorm"
)

func ConvertUnits(db *gorm.DB, amount int64, denom string) (float64, string, error) {
	var denomUnit DenomUnit

	//Try denom unit first
	db.Where("name = ?", denom).Preload("Denom").First(&denomUnit)

	if denomUnit.ID != 0 {
		var highestDenomUnit DenomUnit
		db.Where("denom_id = ?", denomUnit.DenomID).Order("exponent desc").First(&highestDenomUnit)

		symbol := denomUnit.Denom.Symbol

		convertedAmount := float64(amount) / math.Pow(10, float64(highestDenomUnit.Exponent-denomUnit.Exponent))
		return convertedAmount, symbol, nil
	} else {
		//Try alias
		var denomUnitAlias DenomUnitAlias
		db.Where("alias = ?", denom).Preload("DenomUnit").Preload("DenomUnit.Denom").First(&denomUnitAlias)
		if denomUnitAlias.ID != 0 {
			var highestDenomUnit DenomUnit
			db.Where("denom_id = ?", denomUnit.DenomID).Order("exponent desc").First(&highestDenomUnit)
			symbol := denomUnitAlias.DenomUnit.Denom.Symbol
			convertedAmount := float64(amount) / math.Pow(10, float64(highestDenomUnit.Exponent-denomUnitAlias.DenomUnit.Exponent))
			return convertedAmount, symbol, nil
		}
	}

	return 0, "", errors.New("No denom unit or alias found for the given denom")
}
