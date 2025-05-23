package common

import (
	"fmt"
	"strconv"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/utils"
)

// ChainMetadata provides metadata about a chain.
type ChainMetadata struct {
	Selector uint64
}

// ChainSelector returns the chain selector of the chain
func (c ChainMetadata) ChainSelector() uint64 {
	return c.Selector
}

// String returns chain name and selector "<name> (<selector>)"
func (c ChainMetadata) String() string {
	chainInfo, err := utils.ChainInfo(c.Selector)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s (%d)", chainInfo.ChainName, chainInfo.ChainSelector)
}

// Name returns the name of the chain
func (c ChainMetadata) Name() string {
	chainInfo, err := utils.ChainInfo(c.Selector)
	if err != nil {
		return ""
	}
	if chainInfo.ChainName == "" {
		return strconv.FormatUint(c.Selector, 10)
	}

	return chainInfo.ChainName
}
