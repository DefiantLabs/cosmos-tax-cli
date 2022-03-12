package csv

type AccointingRow struct {
	Date           string
	InBuyAmount    float64
	InBuyAsset     string
	OutSellAmount  float64
	OutSellAsset   float64
	FeeAmount      float64
	FeeAsset       string
	Classification string
	OperationId    string
}
