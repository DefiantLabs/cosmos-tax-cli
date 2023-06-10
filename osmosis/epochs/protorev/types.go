package protorev

// protorevDeveloperAddress is the address of the developer account that receives rewards on the weekly Epoch.
// This will need to be prepopulated by the indexer before module startup
var protorevDeveloperAddress string

func SetDeveloperAddress(address string) {
	protorevDeveloperAddress = address
}
