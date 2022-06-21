package db

import (
	"errors"
	"fmt"
	"math"
	"math/big"

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

	return DenomUnit{}, errors.New("no denom unit for the specified denom")
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
		return highestDenomUnit, fmt.Errorf("highest denom not found for denom %s", denomUnit.Name)
	}

	return highestDenomUnit, nil
}

//TODO unit test this function
func ConvertUnits(amount big.Int, denom string) (*big.Int, string, error) {

	//Try denom unit first
	denomUnit, err := GetDenomUnitForDenom(denom)

	if err != nil {
		fmt.Println("Error getting denom unit for denom", denom)
		return nil, "", fmt.Errorf("error getting denom unit for denom %s", denom)
	}

	highestDenomUnit, err := GetHighestDenomUnit(denomUnit, CachedDenomUnits)

	if err != nil {
		fmt.Println("Error getting highest denom unit for denom", denom)
		return nil, "", fmt.Errorf("error getting highest denom unit for denom %s", denom)
	}

	symbol := denomUnit.Denom.Symbol

	power := math.Pow(10, float64(highestDenomUnit.Exponent-denomUnit.Exponent))
	pw := big.NewInt(int64(power))
	convertedAmount := new(big.Int).Set(&amount)
	convertedAmount.Div(convertedAmount, pw)
	return convertedAmount, symbol, nil
}
