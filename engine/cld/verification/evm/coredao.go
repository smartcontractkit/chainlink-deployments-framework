package evm

var coreDAOChainIDs = map[uint64]struct{}{
	1116: {}, 1115: {},
}

func IsChainSupportedOnCoreDAO(chainID uint64) bool {
	_, ok := coreDAOChainIDs[chainID]
	return ok
}
