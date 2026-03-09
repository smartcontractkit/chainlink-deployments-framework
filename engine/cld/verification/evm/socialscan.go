package evm

// IsChainSupportedOnSocialScanV2 returns true if the chain is supported by SocialScan.
// Note: SocialScan verifier is not yet implemented; factory returns an error for this strategy.
var socialscanV2ChainIDs = map[uint64]string{
	688688: "pharos-testnet",
}

func IsChainSupportedOnSocialScanV2(chainID uint64) bool {
	_, ok := socialscanV2ChainIDs[chainID]
	return ok
}
