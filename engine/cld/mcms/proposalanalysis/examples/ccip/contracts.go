package ccip

var TokenPoolContractTypes = map[string]struct{}{
	"LockReleaseTokenPool":      {},
	"BurnMintTokenPool":         {},
	"BurnFromMintTokenPool":     {},
	"BurnWithFromMintTokenPool": {},
	"TokenPool":                 {},
}

func IsTokenPoolContract(contractType string) bool {
	_, ok := TokenPoolContractTypes[contractType]

	return ok
}
