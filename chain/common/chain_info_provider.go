package common

import (
	"fmt"
	"strconv"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
)

type ChainInfoProvider struct {
	Selector uint64
}

func (c ChainInfoProvider) ChainSelector() uint64 {
	return c.Selector
}

func (c ChainInfoProvider) String() string {
	chainInfo, err := ChainInfo(c.Selector)
	if err != nil {
		// we should never get here, if the selector is invalid it should not be in the environment
		panic(err)
	}

	return fmt.Sprintf("%s (%d)", chainInfo.ChainName, chainInfo.ChainSelector)
}

func (c ChainInfoProvider) Name() string {
	chainInfo, err := ChainInfo(c.Selector)
	if err != nil {
		// we should never get here, if the selector is invalid it should not be in the environment
		panic(err)
	}
	if chainInfo.ChainName == "" {
		return strconv.FormatUint(c.Selector, 10)
	}

	return chainInfo.ChainName
}

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
