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

func TestOsmoParsing(t *testing.T) {
	// setup parser
	parser := GetParser(accointing.ParserKey)
	cfg := config.Config{}
	cfg.Lens.ChainID = osmosis.OsmoChainID
	parser.InitializeParsingGroups(cfg)

	// setup user and chain
	targetAddress := mkAddress(t, 1)
	chain := mkChain(1, osmosis.OsmoChainID, osmosis.OsmoName)

	// make transactions like those you would get from the DB
	taxableTxs := getTestTXs(t, targetAddress, chain)

	// attempt to parse
	err := parser.ProcessTaxableTx(targetAddress.Address, taxableTxs)
	assert.Nil(t, err, "should not get error from parsing these transactions")

	// validate output
	// TODO: validate the output from the process func
}

func getTestTXs(t *testing.T, targetAddress db.Address, targetChain db.Chain) []db.TaxableTransaction {
	TXs := []db.TaxableTransaction{}
	// TODO: actually make some transactions
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

func mkBlk(id uint, height int64, chainID uint, chain db.Chain) db.Block {
	return db.Block{
		ID:           id,
		Height:       height,
		BlockchainID: chainID,
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
