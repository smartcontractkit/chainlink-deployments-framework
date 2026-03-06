package evm

var l2scanChainIDs = map[uint64]struct{}{
	4200: {}, 686868: {}, 223: {},
}

func IsChainSupportedOnL2Scan(chainID uint64) bool {
	_, ok := l2scanChainIDs[chainID]
	return ok
}
