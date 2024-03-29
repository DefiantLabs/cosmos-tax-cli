package db

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync"

	"github.com/DefiantLabs/cosmos-tax-cli/chainregistry"
	"gorm.io/gorm"
)

var (
	CachedDenomUnits []DenomUnit
	denomCacheMutex  sync.Mutex

	CachedIBCDenoms    []IBCDenom
	ibcDenomCacheMutex sync.Mutex
)

func CacheDenoms(db *gorm.DB) {
	var denomUnits []DenomUnit
	db.Preload("Denom").Find(&denomUnits)
	denomCacheMutex.Lock()
	defer denomCacheMutex.Unlock()
	CachedDenomUnits = denomUnits
}

func CacheIBCDenoms(db *gorm.DB) {
	var ibcDenoms []IBCDenom
	db.Preload("IBCDenom").Find(&ibcDenoms)
	ibcDenomCacheMutex.Lock()
	defer ibcDenomCacheMutex.Unlock()
	CachedIBCDenoms = ibcDenoms
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

func GetIBCDenom(denomTrace string) (IBCDenom, error) {
	ibcDenomCacheMutex.Lock()
	defer ibcDenomCacheMutex.Unlock()
	for _, denom := range CachedIBCDenoms {
		if denom.Hash == denomTrace {
			return denom, nil
		}
	}
	return IBCDenom{}, fmt.Errorf("no IBC denom found for the specified denom trace %s", denomTrace)
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
	// Code hack for IBC MsgAck denoms
	// MsgAcks have a denom of the form transfer/{channel/{base denom}. Attempt to use that to parse out the base denom unit first.
	if strings.HasPrefix(denom.Base, "transfer/") {
		splitString := strings.Split(denom.Base, "/")
		base := splitString[len(splitString)-1]
		searchDenom, err := GetDenomForBase(base)
		if err == nil {
			for _, denomUnit := range CachedDenomUnits {
				if denomUnit.DenomID == searchDenom.ID && denomUnit.Exponent == 0 {
					return denomUnit, nil
				}
			}
		}
	}

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

func ConvertUnits(amount *big.Int, denom Denom) (*big.Float, string, error) {
	convertedAmount := new(big.Float).SetInt(amount)

	// Handle gamm special case
	if strings.HasPrefix(denom.Base, "gamm/pool/") {
		power := math.Pow(10, float64(18))
		return new(big.Float).Quo(convertedAmount, new(big.Float).SetFloat64(power)), denom.Base, nil
	}

	// Try chainregistry asset lists first
	// We are experimenting with a full pull-down of the asset list entries in the chain registry to see if
	// they provide good coverage for parsing items into symbols.
	base := denom.Base
	if strings.HasPrefix(base, "transfer/") {
		splitString := strings.Split(denom.Base, "/")
		base = splitString[len(splitString)-1]
	}

	assetEntry, ok := chainregistry.GetCachedAssetEntry(base)

	var symbol string
	var highestExponent uint
	var baseExponent uint
	var highestExponentName string
	if ok {
		baseExponent = chainregistry.GetBaseDenomUnitForAsset(assetEntry).Exponent
		highestDenomUnit := chainregistry.GetHighestDenomUnitForAsset(assetEntry)
		highestExponent = highestDenomUnit.Exponent
		highestExponentName = highestDenomUnit.Denom
		symbol = assetEntry.Symbol
	} else {

		// Try denom unit second
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

		symbol = denomUnit.Denom.Symbol
		highestExponent = highestDenomUnit.Exponent
		baseExponent = denomUnit.Exponent
		highestExponentName = highestDenomUnit.Name
	}
	// We were converting the units to big.Int, which would cause a Token to appear 0 if the conversion resulted in an amount < 1
	power := math.Pow(10, float64(highestExponent-baseExponent))
	dividedAmount := new(big.Float).Quo(convertedAmount, new(big.Float).SetFloat64(power))
	if symbol == "UNKNOWN" || symbol == "" {
		symbol = highestExponentName
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
	CacheIBCDenoms(db)

	return GetDenomForBase(denom)
}
