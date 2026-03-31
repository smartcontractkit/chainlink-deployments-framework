package evm

var sourcifyChainIDs = map[uint64]struct{}{
	295: {}, 296: {}, 2020: {}, 2021: {}, 4217: {}, 42431: {},
}

// sourcifyCustomServerURLs maps chain IDs to custom Sourcify-compatible server URLs.
// Chains not listed here fall back to the default sourcifyServerURL.
var sourcifyCustomServerURLs = map[uint64]string{
	4217:  "https://contracts.tempo.xyz",
	42431: "https://contracts.tempo.xyz",
}

func IsChainSupportedOnSourcify(chainID uint64) bool {
	_, ok := sourcifyChainIDs[chainID]
	return ok
}

func getSourcifyServerURL(chainID uint64) string {
	if url, ok := sourcifyCustomServerURLs[chainID]; ok {
		return url
	}
	return sourcifyServerURL
}
