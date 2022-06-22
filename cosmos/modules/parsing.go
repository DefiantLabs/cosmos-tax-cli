package parsing

import "math/big"

type MessageRelevantInformation struct {
	SenderAddress   string
	ReceiverAddress string
	Amount          *big.Int
	Denomination    string
}
