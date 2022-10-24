package parsers

// Check in your parsers here
var Parsers map[string]bool

func init() {
	Parsers = make(map[string]bool)
}

func RegisterParser(key string) {
	Parsers[key] = true
}

func GetParserKeys() []string {
	var parserKeys []string

	for i := range Parsers {
		parserKeys = append(parserKeys, i)
	}

	return parserKeys
}
