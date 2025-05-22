package internal

import (
	"fmt"
	"strconv"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
)

// ChainBase is a base struct for all chains.
// It should be embedded in all chain structs.
// Note: It is not embedded in EVM, Solana and Aptos chains to maintain backward compatibility.
// However new Chains should embed it.
type ChainBase struct {
	Selector uint64
}

// ChainSelector returns the chain selector of the chain
func (c ChainBase) ChainSelector() uint64 {
	return c.Selector
}

// String returns chain name and selector "<name> (<selector>)"
func (c ChainBase) String() string {
	chainInfo, err := ChainInfo(c.Selector)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s (%d)", chainInfo.ChainName, chainInfo.ChainSelector)
}

// Name returns the name of the chain
func (c ChainBase) Name() string {
	chainInfo, err := ChainInfo(c.Selector)
	if err != nil {
		return ""
	}
	if chainInfo.ChainName == "" {
		return strconv.FormatUint(c.Selector, 10)
	}

	return chainInfo.ChainName
}

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
