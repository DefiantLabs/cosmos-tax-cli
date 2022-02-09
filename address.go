package main

import (
	"regexp"
)

var addressRegex *regexp.Regexp

func setupAddressRegex(addressRegexPattern string) {
	addressRegex, _ = regexp.Compile(addressRegexPattern)
}

func ExtractTransactionAddresses(tx MergedTx) []string {
	//TODO: Need to walk messages blocks and extract addresses
	//Consider walking logs - needs benchmarking compared to whole string search on raw log
	addresses := addressRegex.FindAllString(tx.TxResponse.RawLog, -1)
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
