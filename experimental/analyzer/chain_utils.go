package analyzer

import (
	chainsel "github.com/smartcontractkit/chain-selectors"
)

// GetChainNameBySelector retrieves the chain name for a given chain selector.
func GetChainNameBySelector(selector uint64) (string, error) {
	chainID, err := chainsel.GetChainIDFromSelector(selector)
	if err != nil {
		return "", err
	}
	family, err := chainsel.GetSelectorFamily(selector)
	if err != nil {
		return "", err
	}
	chainInfo, err := chainsel.GetChainDetailsByChainIDAndFamily(chainID, family)
	if err != nil {
		return "", err
	}

	return chainInfo.ChainName, nil
}
