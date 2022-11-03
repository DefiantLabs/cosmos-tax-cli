package denoms

type GetDenomsMetadatasResponse struct {
	Metadatas  []Metadata `json:"metadatas"`
	Pagination Pagination `json:"pagination"`
}

type Metadata struct {
	Description string      `json:"description"`
	DenomUnits  []DenomUnit `json:"denom_units"`
	Base        string      `json:"base"`
	Display     string      `json:"display"`
	Name        string      `json:"name"`
	Symbol      string      `json:"symbol"`
}

type DenomUnit struct {
	Denom    string   `json:"denom"`
	Exponent int      `json:"exponent"`
	Aliases  []string `json:"aliases"`
}

type Pagination struct {
	NextKey string `json:"next_key"`
	Total   string `json:"total"`
}
