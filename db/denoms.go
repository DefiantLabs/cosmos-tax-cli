package db

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"

	"gorm.io/gorm"
)

var CachedDenomUnits []DenomUnit
var denomCacheMutex sync.Mutex

func CacheDenoms(db *gorm.DB) {
	// TODO need to load aliases as well
	var denomUnits []DenomUnit
	db.Preload("Denom").Find(&denomUnits)
	denomCacheMutex.Lock()
	defer denomCacheMutex.Unlock()
	CachedDenomUnits = denomUnits
}

func GetDenomForBase(base string) (Denom, error) {
	denomCacheMutex.Lock()
	defer denomCacheMutex.Unlock()
	for _, denomUnit := range CachedDenomUnits {
		if denomUnit.Denom.Base == base {
			return denomUnit.Denom, nil
		}
	}

	return Denom{}, fmt.Errorf("GetDenomForBase: no denom unit for the specified denom %s", base)
}

func GetDenomUnitForDenom(denom Denom) (DenomUnit, error) {
	for _, denomUnit := range CachedDenomUnits {
		if denomUnit.DenomID == denom.ID {
			return denomUnit, nil
		}
	}

	return DenomUnit{}, errors.New("GetDenomUnitForDenom: no denom unit for the specified denom")
}

func GetBaseDenomUnitForDenom(denom Denom) (DenomUnit, error) {
	for _, denomUnit := range CachedDenomUnits {
		if denomUnit.DenomID == denom.ID && denomUnit.Exponent == 0 {
			return denomUnit, nil
		}
	}

	return DenomUnit{}, errors.New("GetDenomUnitForDenom: no denom unit for the specified denom")
}

func GetHighestDenomUnit(denomUnit DenomUnit, denomUnits []DenomUnit) (DenomUnit, error) {
	highestDenomUnit := DenomUnit{Exponent: 0, Name: "not found for denom"}

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

// TODO unit test this function
func ConvertUnits(amount *big.Int, denom Denom) (*big.Float, string, error) {
	// Try denom unit first
	// We were originally just using GetDenomUnitForDenom, but since CachedDenoms is an array, it would sometimes
	// return the non-Base denom unit (exponent != 0), which would break the power conversion process below i.e.
	// it would sometimes do highestDenomUnit.Exponent = 6, denomUnit.Exponent = 6 -> pow = 0
	denomUnit, err := GetBaseDenomUnitForDenom(denom)
	if err != nil {
		fmt.Println("Error getting denom unit for denom", denom)
		return nil, "", fmt.Errorf("error getting denom unit for denom %+v", denom)
	}

	highestDenomUnit, err := GetHighestDenomUnit(denomUnit, CachedDenomUnits)
	if err != nil {
		fmt.Println("Error getting highest denom unit for denom", denom)
		return nil, "", fmt.Errorf("error getting highest denom unit for denom %+v", denom)
	}

	symbol := denomUnit.Denom.Symbol

	// We were converting the units to big.Int, which would cause a Token to appear 0 if the conversion resulted in an amount < 1
	power := math.Pow(10, float64(highestDenomUnit.Exponent-denomUnit.Exponent))
	convertedAmount := new(big.Float).SetInt(amount)
	dividedAmount := new(big.Float).Quo(convertedAmount, new(big.Float).SetFloat64(power))
	if symbol == "UNKNOWN" || symbol == "" {
		symbol = highestDenomUnit.Name
	}
	return dividedAmount, symbol, nil
}

// This function assumes that the denom to be added is the base denom
// which will be correct in all cases that the missing denom was pulled from
// a transaction message and not found in our database during tx parsing
// Creates a single denom and a single denom_unit that fits our DB structure, adds them to the DB
func AddUnknownDenom(db *gorm.DB, denom string) (Denom, error) {
	denomToAdd := Denom{Base: denom, Name: "UNKNOWN", Symbol: "UNKNOWN"}
	singleDenomUnit := DenomUnit{Exponent: 0, Name: denom}
	denomUnitsToAdd := [...]DenomUnitDBWrapper{{DenomUnit: singleDenomUnit}}

	denomDbWrapper := [1]DenomDBWrapper{{Denom: denomToAdd}}
	denomDbWrapper[0].DenomUnits = denomUnitsToAdd[:]

	err := UpsertDenoms(db, denomDbWrapper[:])
	if err != nil {
		return denomToAdd, err
	}

	// recache the denoms (threadsafe due to mutex on read and write)
	CacheDenoms(db)

	return GetDenomForBase(denom)
}
