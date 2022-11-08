// nolint:unused
package csv

import (
	"crypto/rand"
	"fmt"
	"github.com/DefiantLabs/cosmos-tax-cli-private/csv/parsers/koinly"
	"strings"
	"testing"
	"time"

	"github.com/DefiantLabs/cosmos-tax-cli-private/config"
	"github.com/DefiantLabs/cosmos-tax-cli-private/csv/parsers/accointing"
	"github.com/DefiantLabs/cosmos-tax-cli-private/db"
	"github.com/DefiantLabs/cosmos-tax-cli-private/osmosis"
	"github.com/DefiantLabs/cosmos-tax-cli-private/osmosis/modules/gamm"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

var zeroTime time.Time

func TestKoinlyOsmoLPParsing(t *testing.T) {
	cfg := config.Config{}
	cfg.Lens.ChainID = osmosis.ChainID
	parser := GetParser(koinly.ParserKey)
	parser.InitializeParsingGroups(cfg)

	// setup user and chain
	targetAddress := mkAddress(t, 1)
	chain := mkChain(1, osmosis.ChainID, osmosis.Name)

	// make transactions for this user entering and leaving LPs
	transferTxs := getTestTransferTXs(t, targetAddress, chain)

	// attempt to parse
	err := parser.ProcessTaxableTx(targetAddress.Address, transferTxs)
	assert.Nil(t, err, "should not get error from parsing these transactions")

	// validate output
	rows := parser.GetRows()
	assert.Equalf(t, len(transferTxs), len(rows), "you should have one row for each transfer transaction ")

	// all transactions should be orders classified as liquidity_pool
	for i, row := range rows {
		cols := row.GetRowForCsv()
		// first column should parse as a time and not be zero-time
		time, err := time.Parse(koinly.TimeLayout, cols[0])
		assert.Nil(t, err, "time should parse properly")
		assert.NotEqual(t, time, zeroTime)

		// make sure transactions are properly labeled
		if i < 4 {
			assert.Equal(t, cols[9], koinly.LiquidityIn.String())
		} else {
			assert.Equal(t, cols[9], koinly.LiquidityOut.String())
		}
		// TODO: add more tests
	}
}

func TestKoinlyOsmoRewardParsing(t *testing.T) {
	cfg := config.Config{}
	cfg.Lens.ChainID = osmosis.ChainID
	parser := GetParser(koinly.ParserKey)
	parser.InitializeParsingGroups(cfg)

	// setup user and chain
	targetAddress := mkAddress(t, 1)
	chain := mkChain(1, osmosis.ChainID, osmosis.Name)

	// make transactions for this user entering and leaving LPs
	taxableEvents := getTestTaxableEvents(t, targetAddress, chain)

	// attempt to parse
	err := parser.ProcessTaxableEvent(taxableEvents)
	assert.Nil(t, err, "should not get error from parsing these transactions")

	// validate output
	rows := parser.GetRows()
	assert.Equalf(t, len(taxableEvents), len(rows), "you should have one row for each transfer transaction ")

	// all transactions should be orders classified as liquidity_pool
	for _, row := range rows {
		cols := row.GetRowForCsv()
		// first column should parse as a time
		_, err := time.Parse(koinly.TimeLayout, cols[0])
		assert.Nil(t, err, "time should parse properly")

		// make sure transactions are properly labeled
		assert.Equal(t, cols[9], "reward")

		// TODO: add more tests
	}
}

func TestAccointingOsmoLPParsing(t *testing.T) {
	cfg := config.Config{}
	cfg.Lens.ChainID = osmosis.ChainID
	parser := GetParser(accointing.ParserKey)
	parser.InitializeParsingGroups(cfg)

	// setup user and chain
	targetAddress := mkAddress(t, 1)
	chain := mkChain(1, osmosis.ChainID, osmosis.Name)

	// make transactions for this user entering and leaving LPs
	transferTxs := getTestTransferTXs(t, targetAddress, chain)

	// attempt to parse
	err := parser.ProcessTaxableTx(targetAddress.Address, transferTxs)
	assert.Nil(t, err, "should not get error from parsing these transactions")

	// validate output
	rows := parser.GetRows()
	assert.Equalf(t, len(transferTxs), len(rows), "you should have one row for each transfer transaction ")

	// all transactions should be orders classified as liquidity_pool
	for _, row := range rows {
		cols := row.GetRowForCsv()
		// assert on gamms being present
		assert.Equal(t, cols[0], "order", "transaction type should be an order")
		assert.Equal(t, cols[8], "", "transaction should not have a classification")
		// should either contain gamm value or a message about how to find it
		if !strings.Contains(cols[10], "USD") && !strings.Contains(cols[10], "") {
			t.Log("comment should say value of gamm")
			t.Fail()
		}
	}
}

func TestAccointingOsmoRewardParsing(t *testing.T) {
	cfg := config.Config{}
	cfg.Lens.ChainID = osmosis.ChainID
	parser := GetParser(accointing.ParserKey)
	parser.InitializeParsingGroups(cfg)

	// setup user and chain
	targetAddress := mkAddress(t, 1)
	chain := mkChain(1, osmosis.ChainID, osmosis.Name)

	// make transactions for this user entering and leaving LPs
	taxableEvents := getTestTaxableEvents(t, targetAddress, chain)

	// attempt to parse
	err := parser.ProcessTaxableEvent(taxableEvents)
	assert.Nil(t, err, "should not get error from parsing these transactions")

	// validate output
	rows := parser.GetRows()
	assert.Equalf(t, len(taxableEvents), len(rows), "you should have one row for each transfer transaction ")

	// all transactions should be orders classified as liquidity_pool
	for _, row := range rows {
		cols := row.GetRowForCsv()
		// assert on gamms being present
		assert.Equal(t, cols[0], "deposit", "transaction type should be a deposit")
		assert.Equal(t, cols[8], "liquidity_pool", "transaction should be classified as liquidity_pool")
	}
}

func getTestTaxableEvents(t *testing.T, targetAddress db.Address, targetChain db.Chain) []db.TaxableEvent {
	// BlockTimes
	oneYearAgo := time.Now().Add(-1 * time.Hour * 24 * 365)
	sixMonthAgo := time.Now().Add(-1 * time.Hour * 24 * 182)

	// create some blocks to put the transactions in
	block1 := mkBlk(1, 1, oneYearAgo, targetChain)
	block2 := mkBlk(2, 2, sixMonthAgo, targetChain)

	// create denom
	coinDenom, _ := mkDenom(1, "coin", "Some Coin", "SC")

	event1 := mkTaxableEvent(1, decimal.NewFromInt(10), coinDenom, targetAddress, block1)
	event2 := mkTaxableEvent(2, decimal.NewFromInt(10), coinDenom, targetAddress, block2)

	return []db.TaxableEvent{event1, event2}
}

func mkTaxableEvent(id uint, amount decimal.Decimal, denom db.Denom, address db.Address, block db.Block) db.TaxableEvent {
	return db.TaxableEvent{
		ID:             id,
		Source:         db.OsmosisRewardDistribution,
		Amount:         amount,
		DenominationID: denom.ID,
		Denomination:   denom,
		AddressID:      address.ID,
		EventAddress:   address,
		BlockID:        block.ID,
		Block:          block,
	}
}

func getTestTransferTXs(t *testing.T, targetAddress db.Address, targetChain db.Chain) []db.TaxableTransaction {
	randoAddress := mkAddress(t, 2)

	// BlockTimes
	oneYearAgo := time.Now().Add(-1 * time.Hour * 24 * 365)
	sixMonthAgo := time.Now().Add(-1 * time.Hour * 24 * 182)

	// create some blocks to put the transactions in
	block1 := mkBlk(1, 1, oneYearAgo, targetChain)
	block2 := mkBlk(2, 2, sixMonthAgo, targetChain)

	// create the transfer msg type
	// joins
	joinSwapExternAmountIn := mkMsgType(1, gamm.MsgJoinSwapExternAmountIn)
	joinSwapShareAmountOut := mkMsgType(2, gamm.MsgJoinSwapShareAmountOut)
	joinPool := mkMsgType(3, gamm.MsgJoinPool)
	// exits
	exitSwapShareAmountIn := mkMsgType(4, gamm.MsgExitSwapShareAmountIn)
	exitSwapExternAmountOut := mkMsgType(5, gamm.MsgExitSwapExternAmountOut)
	exitPool := mkMsgType(6, gamm.MsgExitPool)

	// FIXME: add fees

	// create TXs
	joinPoolTX1 := mkTx(1, "somehash1", 0, block1, randoAddress, nil)
	joinPoolTX2 := mkTx(2, "somehash2", 0, block1, randoAddress, nil)

	leavePoolTX1 := mkTx(3, "somehash4", 0, block2, randoAddress, nil)
	leavePoolTX2 := mkTx(4, "somehash5", 0, block2, randoAddress, nil)

	// create Msgs
	joinSwapExternAmountInMsg := mkMsg(1, joinPoolTX1, joinSwapExternAmountIn, 0)
	joinSwapShareAmountOutMsg := mkMsg(2, joinPoolTX1, joinSwapShareAmountOut, 1)
	joinPoolMsg := mkMsg(3, joinPoolTX2, joinPool, 0)

	exitSwapShareAmountInMsg := mkMsg(4, leavePoolTX1, exitSwapShareAmountIn, 0)
	exitSwapExternAmountOutMsg := mkMsg(5, leavePoolTX1, exitSwapExternAmountOut, 1)
	exitPoolMsg := mkMsg(6, leavePoolTX2, exitPool, 2)

	// create denoms
	coin1, coin1DenomUnit := mkDenom(1, "coin1", "Some Coin", "SC1")
	coin2, coin2DenomUnit := mkDenom(2, "coin2", "Another Coin", "AC2")
	gamm1, gamm1DenomUnit := mkDenom(3, "gamm/pool/1", "UNKNOWN", "UNKNOWN")

	// populate denom cache
	db.CachedDenomUnits = []db.DenomUnit{coin1DenomUnit, coin2DenomUnit, gamm1DenomUnit}

	// create taxable transactions
	// joins
	taxableTX1 := mkTaxableTransaction(1, joinSwapExternAmountInMsg, decimal.NewFromInt(12000), decimal.NewFromInt(24000038), coin1, gamm1, targetAddress, targetAddress)
	taxableTX2 := mkTaxableTransaction(2, joinSwapShareAmountOutMsg, decimal.NewFromInt(11999), decimal.NewFromInt(24000000), coin1, gamm1, targetAddress, targetAddress)
	taxableTX3 := mkTaxableTransaction(3, joinPoolMsg, decimal.NewFromInt(6000), decimal.NewFromFloat(12000019), coin1, gamm1, targetAddress, targetAddress)
	taxableTX4 := mkTaxableTransaction(4, joinPoolMsg, decimal.NewFromInt(3000), decimal.NewFromFloat(12000019), coin2, gamm1, targetAddress, targetAddress)
	// exits
	taxableTX5 := mkTaxableTransaction(5, exitSwapShareAmountInMsg, decimal.NewFromInt(24000038), decimal.NewFromInt(12438), gamm1, coin1, targetAddress, targetAddress)
	taxableTX6 := mkTaxableTransaction(6, exitSwapExternAmountOutMsg, decimal.NewFromInt(23152955), decimal.NewFromInt(11999), gamm1, coin1, targetAddress, targetAddress)
	taxableTX7 := mkTaxableTransaction(7, exitPoolMsg, decimal.NewFromInt(12000019), decimal.NewFromInt(6219), gamm1, coin1, targetAddress, targetAddress)
	taxableTX8 := mkTaxableTransaction(8, exitPoolMsg, decimal.NewFromInt(12000019), decimal.NewFromInt(2853), gamm1, coin2, targetAddress, targetAddress)

	return []db.TaxableTransaction{taxableTX1, taxableTX2, taxableTX3, taxableTX4, taxableTX5, taxableTX6, taxableTX7, taxableTX8}
}

func mkTaxableTransaction(id uint, msg db.Message, amntSent, amntReceived decimal.Decimal, denomSent db.Denom, denomReceived db.Denom, senderAddr db.Address, receiverAddr db.Address) db.TaxableTransaction {
	return db.TaxableTransaction{
		ID:                     id,
		MessageID:              msg.ID,
		Message:                msg,
		AmountSent:             amntSent,
		AmountReceived:         amntReceived,
		DenominationSentID:     &denomSent.ID,
		DenominationSent:       denomSent,
		DenominationReceivedID: &denomReceived.ID,
		DenominationReceived:   denomReceived,
		SenderAddressID:        &senderAddr.ID,
		SenderAddress:          senderAddr,
		ReceiverAddressID:      &receiverAddr.ID,
		ReceiverAddress:        receiverAddr,
	}
}

func mkDenom(id uint, base, name, symbol string) (denom db.Denom, denomUnit db.DenomUnit) {
	denom = db.Denom{
		ID:     id,
		Base:   base,
		Name:   name,
		Symbol: symbol,
	}

	denomUnit = db.DenomUnit{
		ID:       id,
		DenomID:  id,
		Denom:    denom,
		Exponent: 0,
		Name:     denom.Base,
	}

	return
}

func mkMsg(id uint, tx db.Tx, msgType db.MessageType, msgIdx int) db.Message {
	return db.Message{
		ID:            id,
		TxID:          tx.ID,
		Tx:            tx,
		MessageTypeID: msgType.ID,
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

func mkTx(id uint, hash string, code uint32, block db.Block, signerAddr db.Address, fees []db.Fee) db.Tx {
	signerAddrID := int(signerAddr.ID)
	return db.Tx{
		ID:              id,
		Hash:            hash,
		Code:            code,
		BlockID:         block.ID,
		Block:           block,
		SignerAddressID: &signerAddrID,
		SignerAddress:   signerAddr,
		Fees:            fees,
	}
}

func mkBlk(id uint, height int64, timestamp time.Time, chain db.Chain) db.Block {
	return db.Block{
		ID:           id,
		Height:       height,
		TimeStamp:    timestamp,
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
