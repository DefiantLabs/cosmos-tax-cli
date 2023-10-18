package chainregistry

type AssetList struct {
	ChainName string  `json:"chain_name"`
	Assets    []Asset `json:"assets"`
}

type Asset struct {
	Description string           `json:"description"`
	Base        string           `json:"base"`
	Symbol      string           `json:"symbol"`
	DenomUnits  []AssetDenomUnit `json:"denom_units"`
	ChainName   string           `json:"chain_name,omitempty"`
}

type AssetDenomUnit struct {
	Denom    string `json:"denom"`
	Exponent uint   `json:"exponent"`
}

var CachedAssetMap = map[string]Asset{}

func CacheAssetMap(newVal map[string]Asset) {
	CachedAssetMap = newVal
}

func GetCachedAssetEntry(denomBase string) (Asset, bool) {
	asset, ok := CachedAssetMap[denomBase]
	return asset, ok
}

func GetBaseDenomUnitForAsset(asset Asset) AssetDenomUnit {
	lowestDenomUnit := AssetDenomUnit{Exponent: 0}
	for _, denomUnit := range asset.DenomUnits {
		if denomUnit.Exponent <= lowestDenomUnit.Exponent {
			lowestDenomUnit = denomUnit
		}
	}

	return lowestDenomUnit
}

func GetHighestDenomUnitForAsset(asset Asset) AssetDenomUnit {
	highestDenomUnit := AssetDenomUnit{Exponent: 0}
	for _, denomUnit := range asset.DenomUnits {
		if denomUnit.Exponent >= highestDenomUnit.Exponent {
			highestDenomUnit = denomUnit
		}
	}

	return highestDenomUnit
}
