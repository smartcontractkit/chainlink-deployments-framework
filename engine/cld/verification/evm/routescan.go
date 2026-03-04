package evm

// routescanChainIDs - see https://routescan.notion.site
var routescanChainIDs = map[string]map[uint64]string{
	"testnet": {
		21000001: "21000001", 3636: "3636", 80069: "80069", 9746: "9746_5", 43113: "43113",
	},
	"mainnet": {
		21000000: "21000000", 3637: "3637", 80094: "80094", 9745: "9745", 43114: "43114",
	},
}

func IsChainSupportedOnRouteScan(chainID uint64) (networkType string, ok bool) {
	for nt, ids := range routescanChainIDs {
		if _, found := ids[chainID]; found {
			return nt, true
		}
	}

	return "", false
}
