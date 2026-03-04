package evm

var sourcifyChainIDs = map[uint64]struct{}{
	295: {}, 296: {}, 2020: {}, 2021: {},
}

func IsChainSupportedOnSourcify(chainID uint64) bool {
	_, ok := sourcifyChainIDs[chainID]
	return ok
}
