package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// GetTxHash takes in the Base64, hashed string found in Blocks and returns the transaction hash
func GetTxHash(encodedTx string) string {

	data, _ := base64.StdEncoding.DecodeString(encodedTx)
	h := sha256.New()
	h.Write([]byte(data))
	p := []uint8(h.Sum(nil))

	return strings.ToUpper(hex.EncodeToString(p[0:32]))

}
