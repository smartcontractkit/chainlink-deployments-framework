package ccip

var tokenPoolContractTypes = map[string]struct{}{
	"LockReleaseTokenPool":      {},
	"BurnMintTokenPool":         {},
	"BurnFromMintTokenPool":     {},
	"BurnWithFromMintTokenPool": {},
	"TokenPool":                 {},
}

func IsTokenPoolContract(contractType string) bool {
	_, ok := tokenPoolContractTypes[contractType]

	return ok
}
