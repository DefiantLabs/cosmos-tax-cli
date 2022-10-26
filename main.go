package main

import (
	"github.com/DefiantLabs/cosmos-tax-cli/cmd"
	"log"
)

func main() {
	//simplest main as recommended by the Cobra package
	err := cmd.Execute()
	if err != nil {
		log.Fatalf("Failed to exectute. Err: %v", err)
	}
}
