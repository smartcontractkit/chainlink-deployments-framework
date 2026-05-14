package common

import (
	"fmt"
	"strconv"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/utils"
)

type BlockChain interface {
	// String returns chain name and selector "<name> (<selector>)"
	String() string

	// Name returns the name of the chain (e.g. Ethereum Mainnet, Solana Mainnet, Aptos Mainnet, etc.)
	Name() string

	// ChainSelector returns the chain's selector
	ChainSelector() uint64

	// Family returns the family of the chain (e.g. evm, solana, aptos, etc.)
	Family() string

	// NetworkType returns the type of network the chain is on (e.g. mainnet, testnet)
	NetworkType() (chainsel.NetworkType, error)

	// IsNetworkType checks if the chain is on the given network type
	IsNetworkType(networkType chainsel.NetworkType) bool

	ReadOnly() (BlockChain, error)
}

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

// Family returns the family of the chain
func (c ChainMetadata) Family() string {
	family, err := chainsel.GetSelectorFamily(c.Selector)
	if err != nil {
		return ""
	}

	return family
}

// NetworkType returns the type of network the chain represents.
func (c ChainMetadata) NetworkType() (chainsel.NetworkType, error) {
	networkType, err := chainsel.GetNetworkType(c.Selector)
	if err != nil {
		return "", err
	}

	return networkType, nil
}

// IsNetworkType checks if the chain is on the given network type
func (c ChainMetadata) IsNetworkType(networkType chainsel.NetworkType) bool {
	// Get the network type of the chain
	t, err := c.NetworkType()
	if err != nil {
		return false
	}

	return t == networkType
}
