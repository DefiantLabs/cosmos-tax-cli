//nolint:unused
package csv

import (
	"crypto/rand"
	"fmt"
	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/DefiantLabs/cosmos-tax-cli/csv/parsers/accointing"
	"github.com/DefiantLabs/cosmos-tax-cli/db"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis"
	"github.com/DefiantLabs/cosmos-tax-cli/osmosis/modules/gamm"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// TODO: Write test to assert that osmosis rewards (aka taxable events) are tagged as deposits and classified as 'liquidity_pool'
// TODO: Write tests for the two swap msgs not currently supported (after adding support)

func TestOsmoLPParsing(t *testing.T) {
	// setup parser
	parser := GetParser(accointing.ParserKey)
	cfg := config.Config{}
	cfg.Lens.ChainID = osmosis.ChainID
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
		assert.Contains(t, cols[10], "USD", "comment should say value of gam at that point in time")
	}

	// TODO: validate the output from the process func
}

func getTestTransferTXs(t *testing.T, targetAddress db.Address, targetChain db.Chain) []db.TaxableTransaction {
	// TODO: create test transaction
	randoAddress := mkAddress(t, 2)

	// create some blocks to put the transactions in
	block1 := mkBlk(1, 1, targetChain)
	block2 := mkBlk(2, 2, targetChain)

	// create the transfer msg type
	//swapExactAmountIn := mkMsgType(1, gamm.MsgSwapExactAmountIn) // FIXME: We don't currently handle these in IsOsmosisJoin/IsOsmosisExit
	//swapExactAmountOut := mkMsgType(2, gamm.MsgSwapExactAmountOut)
	// joins
	joinSwapExternAmountIn := mkMsgType(3, gamm.MsgJoinSwapExternAmountIn)
	//joinSwapShareAmountOut := mkMsgType(4, gamm.MsgJoinSwapShareAmountOut) // FIXME: I need an example transaction for this....
	joinPool := mkMsgType(5, gamm.MsgJoinPool)
	// leaves
	//exitSwapShareAmountIn := mkMsgType(6, gamm.MsgExitSwapShareAmountIn)     // FIXME: I need an example transaction for this....
	//exitSwapExternAmountOut := mkMsgType(7, gamm.MsgExitSwapExternAmountOut) // FIXME: I need an example transaction for this....
	exitPool := mkMsgType(8, gamm.MsgExitPool)

	// TxTimes
	oneYearAgo := time.Now().Add(-1 * time.Hour * 24 * 365)
	sixMonthAgo := time.Now().Add(-1 * time.Hour * 24 * 182)

	//FIXME: add fees

	// create TXs
	joinPoolTX := mkTx(1, oneYearAgo, "somehash", 0, block1, randoAddress, nil)
	leavePoolTX := mkTx(2, sixMonthAgo, "somehash", 0, block2, randoAddress, nil)

	// create Msgs
	joinSwapExternAmountInMsg := mkMsg(1, joinPoolTX, joinSwapExternAmountIn, 0)
	//joinSwapShareAmountOutMsg := mkMsg(2, joinPoolTX, joinSwapShareAmountOut, 1)
	joinPoolMsg := mkMsg(3, joinPoolTX, joinPool, 2)

	//exitSwapShareAmountInMsg := mkMsg(4, leavePoolTX, exitSwapShareAmountIn, 0)
	//exitSwapExternAmountOutMsg := mkMsg(5, leavePoolTX, exitSwapExternAmountOut, 1)
	exitPoolMsg := mkMsg(6, leavePoolTX, exitPool, 2)

	// create denoms
	osmo, osmoDenomUnit := mkDenom(1, "uosmo", "Osmosis", "OSMO")
	terraClassicUSD, terraClassicUSDDenomUnit := mkDenom(2, "ibc/BE1BB42D4BE3C30D50B68D7C41DB4DFCE9678E8EF8C539F6E6A9345048894FCC", "TerraClassicUSD", "USTC")
	gamm560, gamm560DenomUnit := mkDenom(704, "gamm/pool/560", "UNKNOWN", "UNKNOWN")

	// populate denom cache
	db.CachedDenomUnits = []db.DenomUnit{osmoDenomUnit, gamm560DenomUnit, terraClassicUSDDenomUnit}

	// create taxable transactions
	joinSwapExternAmountInTaxableTX := mkTaxableTransaction(1, joinSwapExternAmountInMsg, decimal.NewFromInt(12200000), decimal.NewFromInt(1384385853426963652), osmo, gamm560, targetAddress, randoAddress)
	//joinSwapShareAmountOutTX := mkTaxableTransaction(2, joinSwapShareAmountOutMsg, decimal.NewFromInt(12200000), decimal.NewFromInt(1384385853426963652), OSMO, gamm560, targetAddress, randoAddress)
	joinPoolTaxableTX1 := mkTaxableTransaction(3, joinPoolMsg, decimal.NewFromInt(116419), decimal.NewFromFloat(25650208942808158220694), terraClassicUSD, gamm560, targetAddress, randoAddress)
	joinPoolTaxableTX2 := mkTaxableTransaction(4, joinPoolMsg, decimal.NewFromInt(29999999), decimal.NewFromFloat(25650208942808158220695), osmo, gamm560, targetAddress, randoAddress)

	exitPoolTaxableTX1 := mkTaxableTransaction(5, exitPoolMsg, decimal.NewFromInt(549069370049685844), decimal.NewFromInt(31167087), gamm560, terraClassicUSD, targetAddress, targetAddress)
	exitPoolTaxableTX2 := mkTaxableTransaction(6, exitPoolMsg, decimal.NewFromInt(549069370049685844), decimal.NewFromInt(4839721), gamm560, osmo, targetAddress, targetAddress)

	return []db.TaxableTransaction{joinSwapExternAmountInTaxableTX, joinPoolTaxableTX1, joinPoolTaxableTX2, exitPoolTaxableTX1, exitPoolTaxableTX2}
}

func mkTaxableTransaction(id uint, msg db.Message, amntSent, amntReceived decimal.Decimal, denomSent db.Denom, denomReceived db.Denom, senderAddr db.Address, ReceiverAddr db.Address) db.TaxableTransaction {
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
		ReceiverAddressID:      &ReceiverAddr.ID,
		ReceiverAddress:        ReceiverAddr,
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

func mkTx(id uint, time time.Time, hash string, code uint32, block db.Block, signerAddr db.Address, fees []db.Fee) db.Tx {
	signerAddrID := int(signerAddr.ID)
	return db.Tx{
		ID:              id,
		TimeStamp:       time,
		Hash:            hash,
		Code:            code,
		BlockID:         block.ID,
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
