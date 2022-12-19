package gamm

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO: Write test to assert that osmosis rewards (aka taxable events) are tagged as deposits and classified as 'liquidity_pool'

func TestGammCalc(t *testing.T) {
	bignum := big.NewInt(int64(100))
	nthGamms, remainderGamms := calcNthGams(bignum, 3)
	assert.Equalf(t, nthGamms, big.NewInt(int64(33)), "1/3 of 100 rounds to 33")
	assert.Equalf(t, remainderGamms, big.NewInt(int64(34)), "the 3rd 3rd will get 34")
}
