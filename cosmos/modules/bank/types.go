package bank

import (
	tx "cosmos-exporter/cosmos/modules/tx"
	"encoding/json"
	"fmt"

	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

var IsMsgSend = map[string]bool{
	"MsgSend":                      true,
	"/cosmos.bank.v1beta1.MsgSend": true,
}

//CosmUnmarshal(): Unmarshal JSON for MsgSend.
//Note that MsgSend ignores the TxLogMessage because it isn't needed.
func (sf *WrapperMsgSend) CosmUnmarshal(msgType string, raw []byte, log *tx.TxLogMessage) error {
	sf.Type = msgType
	if err := json.Unmarshal(raw, &sf.CosmosMsgSend); err != nil {
		fmt.Println("Error parsing message: " + err.Error())
		return err
	}

	return nil
}

func (sf *WrapperMsgSend) String() string {
	return fmt.Sprintf("MsgSend: Address %s received %s from %s \n",
		sf.CosmosMsgSend.ToAddress, sf.CosmosMsgSend.Amount, sf.CosmosMsgSend.FromAddress)
}

type WrapperMsgSend struct {
	tx.Message
	CosmosMsgSend bankTypes.MsgSend
}
