package util

import (
	"math/big"
	"regexp"

	"github.com/shopspring/decimal"
)

func ToNumeric(i *big.Int) decimal.Decimal {
	num := decimal.NewFromBigInt(i, 0)
	return num
}

func FromNumeric(num decimal.Decimal) *big.Int {
	return num.BigInt()
}

func NumericToString(num decimal.Decimal) string {
	return FromNumeric(num).String()
}

func WalkFindStrings(data interface{}, regex *regexp.Regexp) []string {
	var ret []string

	//These are enough to walk the messages blocks, but we may want to build out the type switch more
	switch x := data.(type) {
	case []interface{}:
		for _, i := range x {
			ret = append(ret, WalkFindStrings(i, regex)...)
		}
		return ret

	case map[interface{}]interface{}:
		for _, i := range x {
			ret = append(ret, WalkFindStrings(i, regex)...)
		}
		return ret

	case map[string]interface{}:
		for _, i := range x {
			ret = append(ret, WalkFindStrings(i, regex)...)
		}
		return ret

	case string:
		return regex.FindAllString(x, -1)

	default:
		//unsupported type, returns empty Slice
		return ret
	}
}

// StrNotSet will return true if the string value provided is empty
func StrNotSet(value string) bool {
	if len(value) == 0 {
		return true
	}
	return false
}
