package main

import (
	"regexp"
)

var addressRegex *regexp.Regexp

func setupAddressRegex(addressRegexPattern string) {
	addressRegex, _ = regexp.Compile(addressRegexPattern)
}

func ExtractTransactionAddresses(tx MergedTx) []string {
	messagesAddresses := WalkFindStrings(tx.Tx.Body.Messages, addressRegex)
	//Consider walking logs - needs benchmarking compared to whole string search on raw log
	logAddresses := addressRegex.FindAllString(tx.TxResponse.RawLog, -1)
	addresses := append(messagesAddresses, logAddresses...)
	addressMap := make(map[string]string)
	for _, v := range addresses {
		addressMap[v] = ""
	}
	uniqueAddresses := make([]string, len(addressMap))
	i := 0
	for k := range addressMap {
		uniqueAddresses[i] = k
		i++
	}
	return uniqueAddresses
}
