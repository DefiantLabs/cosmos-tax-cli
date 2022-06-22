package util

import (
	"math/big"
	"regexp"

	"github.com/jackc/pgtype"
)

func ToNumeric(i *big.Int) pgtype.Numeric {
	num := pgtype.Numeric{}
	num.Set(i)
	return num
}

func FromNumeric(num pgtype.Numeric) *big.Int {
	return num.Int
}

func NumericToString(num pgtype.Numeric) string {
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
