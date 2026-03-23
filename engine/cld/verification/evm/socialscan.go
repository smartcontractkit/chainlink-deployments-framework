package evm

var socialscanV2ChainIDs = map[uint64]string{
	688688: "pharos-testnet",
}

func IsChainSupportedOnSocialScanV2(chainID uint64) bool {
	_, ok := socialscanV2ChainIDs[chainID]
	return ok
}

func getSocialScanChainName(chainID uint64) (string, bool) {
	name, ok := socialscanV2ChainIDs[chainID]
	return name, ok
}
