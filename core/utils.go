package core

import "regexp"

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
