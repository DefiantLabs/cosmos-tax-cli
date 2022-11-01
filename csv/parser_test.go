//nolint:unused
package csv

import (
	"crypto/rand"
	"fmt"
	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/csv/parsers/accointing"
	"github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestOsmoTransferParsing(t *testing.T) {
	// setup parser
	parser := GetParser(accointing.ParserKey)
	cfg := config.Config{}
	cfg.Lens.ChainID = osmosis.ChainID
	parser.InitializeParsingGroups(cfg)

	// setup user and chain
	targetAddress := mkAddress(t, 1)
	chain := mkChain(1, osmosis.ChainID, osmosis.Name)

	// make transactions like those you would get from the DB
	transferTxs := getTestTransferTXs(t, targetAddress, chain)

	// attempt to parse
	err := parser.ProcessTaxableTx(targetAddress.Address, transferTxs)
	assert.Nil(t, err, "should not get error from parsing these transactions")

	// validate output
	rows := parser.GetRows()
	assert.Equalf(t, len(transferTxs), len(rows), "you should have one row for each transfer transaction ")
	// TODO: validate the output from the process func
}

func getTestTransferTXs(t *testing.T, targetAddress db.Address, targetChain db.Chain) []db.TaxableTransaction {
	// TODO: create test transaction

	TXs := []db.TaxableTransaction{}
	// create some blocks to put the transactions in
	//block1 := mkBlk(1, 1, targetChain)
	//block2 := mkBlk(2, 2, targetChain)

	// create the transfer msg type
	//mkMsgType(1, gamm.MsgSwapExactAmountIn)

	return TXs
}

func mkTaxableTransaction(id, msgID uint, msg db.Message, amntSent, amntReceived decimal.Decimal, denomSentID *uint, denomSent db.Denom, denomReceivedID *uint, denomReceived db.Denom, senderAddrID *uint, senderAddr db.Address, ReceiverAddrID *uint, ReceiverAddr db.Address) db.TaxableTransaction {
	return db.TaxableTransaction{
		ID:                     id,
		MessageID:              msgID,
		Message:                msg,
		AmountSent:             amntSent,
		AmountReceived:         amntReceived,
		DenominationSentID:     denomSentID,
		DenominationSent:       denomSent,
		DenominationReceivedID: denomReceivedID,
		DenominationReceived:   denomReceived,
		SenderAddressID:        senderAddrID,
		SenderAddress:          senderAddr,
		ReceiverAddressID:      ReceiverAddrID,
		ReceiverAddress:        ReceiverAddr,
	}
}

func mkDenom(id uint, base, name, symbol string) db.Denom {
	return db.Denom{
		ID:     id,
		Base:   base,
		Name:   name,
		Symbol: symbol,
	}
}

func mkMsg(id, txID uint, tx db.Tx, msgTypeID uint, msgType db.MessageType, msgIdx int) db.Message {
	return db.Message{
		ID:            id,
		TxID:          txID,
		Tx:            tx,
		MessageTypeID: msgTypeID,
		MessageType:   msgType,
		MessageIndex:  msgIdx,
	}
}

func mkMsgType(id uint, msgType string) db.MessageType {
	return db.MessageType{
		ID:          id,
		MessageType: msgType,
	}
}

func mkTx(id uint, time time.Time, hash string, code uint32, blockID uint, block db.Block, signerAddrID int, signerAddr db.Address, fees []db.Fee) db.Tx {
	return db.Tx{
		ID:              id,
		TimeStamp:       time,
		Hash:            hash,
		Code:            code,
		BlockID:         blockID,
		Block:           block,
		SignerAddressID: &signerAddrID,
		SignerAddress:   signerAddr,
		Fees:            fees,
	}
}

func mkBlk(id uint, height int64, chain db.Chain) db.Block {
	return db.Block{
		ID:           id,
		Height:       height,
		BlockchainID: chain.ID,
		Chain:        chain,
	}
}

func mkChain(id uint, chainID, chainName string) db.Chain {
	return db.Chain{
		ID:      id,
		ChainID: chainID,
		Name:    chainName,
	}
}

func mkAddress(t *testing.T, id uint) db.Address {
	rand32 := make([]byte, 32)
	_, err := rand.Read(rand32)
	assert.Nil(t, err)
	address := fmt.Sprintf("osmo%v", rand32)
	return db.Address{
		ID:      id,
		Address: address,
	}
}
