package utils //nolint:revive // var-naming: We need to keep this name for now for backwards compatibility.

import chainsel "github.com/smartcontractkit/chain-selectors"

// ChainInfo returns the chain info for the given selector.
// It returns an error if the selector is invalid or if the chain info cannot be retrieved.
func ChainInfo(cs uint64) (chainsel.ChainDetails, error) {
	id, err := chainsel.GetChainIDFromSelector(cs)
	if err != nil {
		return chainsel.ChainDetails{}, err
	}
	family, err := chainsel.GetSelectorFamily(cs)
	if err != nil {
		return chainsel.ChainDetails{}, err
	}
	info, err := chainsel.GetChainDetailsByChainIDAndFamily(id, family)
	if err != nil {
		return chainsel.ChainDetails{}, err
	}

	return info, nil
}
