package db

import (
	"errors"
	"fmt"
	"math"

	"gorm.io/gorm"
)

var CachedDenomUnits []DenomUnit

func CacheDenoms(db *gorm.DB) {
	//TODO need to load aliases as well
	var denomUnits []DenomUnit
	db.Preload("Denom").Find(&denomUnits)
	CachedDenomUnits = denomUnits
}

func GetDenomUnitForDenom(denom string) (DenomUnit, error) {
	for _, denomUnit := range CachedDenomUnits {
		if denomUnit.Name == denom {
			return denomUnit, nil
		}
	}

	return DenomUnit{}, errors.New("No denom unit for the specified denom")
}

func GetHighestDenomUnit(denomUnit DenomUnit, denomUnits []DenomUnit) (DenomUnit, error) {
	var highestDenomUnit DenomUnit = DenomUnit{Exponent: 0, Name: "not found for denom"}

	for _, currDenomUnit := range denomUnits {
		if currDenomUnit.Denom.ID == denomUnit.Denom.ID {
			if highestDenomUnit.Exponent <= currDenomUnit.Exponent {
				highestDenomUnit = currDenomUnit
			}
		}
	}

	if highestDenomUnit.Name == "not found for denom" {
		return highestDenomUnit, errors.New(fmt.Sprintf("Highest denom not found for denom %s", denomUnit.Name))
	}

	return highestDenomUnit, nil
}

func ConvertUnits(amount int64, denom string) (float64, string, error) {

	//Try denom unit first
	denomUnit, err := GetDenomUnitForDenom(denom)

	if err != nil {
		fmt.Println("Error getting denom unit for denom", denom)
		return 0, "", errors.New(fmt.Sprintf("Error getting denom unit for denom %s", denom))
	}

	highestDenomUnit, err := GetHighestDenomUnit(denomUnit, CachedDenomUnits)

	if err != nil {
		fmt.Println("Error getting highest denom unit for denom", denom)
		return 0, "", errors.New(fmt.Sprintf("Error getting highest denom unit for denom %s", denom))
	}

	symbol := denomUnit.Denom.Symbol

	convertedAmount := float64(amount) / math.Pow(10, float64(highestDenomUnit.Exponent-denomUnit.Exponent))
	return convertedAmount, symbol, nil
}
