package evm

var oklinkChainShortNames = map[uint64]string{
	196: "XLAYER",
}

func IsChainSupportedOnOkLink(chainID uint64) bool {
	_, ok := oklinkChainShortNames[chainID]
	return ok
}

func GetOkLinkShortName(chainID uint64) (string, bool) {
	name, ok := oklinkChainShortNames[chainID]
	return name, ok
}
