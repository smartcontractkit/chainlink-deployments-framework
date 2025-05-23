package utils

import chain_selectors "github.com/smartcontractkit/chain-selectors"

// ChainInfo returns the chain info for the given selector.
// It returns an error if the selector is invalid or if the chain info cannot be retrieved.
func ChainInfo(cs uint64) (chain_selectors.ChainDetails, error) {
	id, err := chain_selectors.GetChainIDFromSelector(cs)
	if err != nil {
		return chain_selectors.ChainDetails{}, err
	}
	family, err := chain_selectors.GetSelectorFamily(cs)
	if err != nil {
		return chain_selectors.ChainDetails{}, err
	}
	info, err := chain_selectors.GetChainDetailsByChainIDAndFamily(id, family)
	if err != nil {
		return chain_selectors.ChainDetails{}, err
	}

	return info, nil
}
