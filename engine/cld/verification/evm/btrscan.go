package evm

var btrScanChainIDs = map[uint64]struct{}{
	200901: {},
}

func IsChainSupportedOnBtrScan(chainID uint64) bool {
	_, ok := btrScanChainIDs[chainID]
	return ok
}
