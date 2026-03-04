package evm

var blockscoutChainIDs = map[uint64]struct{}{
	1868: {}, 98866: {}, 7777777: {}, 1088: {}, 1329: {}, 43111: {}, 42793: {}, 185: {},
	57073: {}, 1135: {}, 177: {}, 60808: {}, 1750: {}, 47763: {}, 34443: {}, 5330: {},
	592: {}, 30: {}, 2818: {}, 2810: {}, 36888: {}, 26888: {}, 99999: {}, 36900: {},
}

func IsChainSupportedOnBlockscout(chainID uint64) bool {
	_, ok := blockscoutChainIDs[chainID]
	return ok
}
