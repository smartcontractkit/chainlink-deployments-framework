package aptos

import (
	aptoslib "github.com/aptos-labs/aptos-go-sdk"
	chainsel "github.com/smartcontractkit/chain-selectors"

	chaincommon "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

// Chain represents an Aptos chain.
type Chain struct {
	Selector uint64

	Client         aptoslib.AptosRpcClient
	DeployerSigner aptoslib.TransactionSigner
	URL            string

	Confirm func(txHash string, opts ...any) error
}

// Author note: Have to implement the blockhain interface methods explicitly below
// instead of composing the ChainMetadata struct to avoid breaking change since there are existing usage.

// ChainSelector returns the chain selector of the chain
func (c Chain) ChainSelector() uint64 {
	return c.Selector
}

// String returns chain name and selector "<name> (<selector>)"
func (c Chain) String() string {
	return chaincommon.ChainMetadata{Selector: c.Selector}.String()
}

// Name returns the name of the chain
func (c Chain) Name() string {
	return chaincommon.ChainMetadata{Selector: c.Selector}.Name()
}

// Family returns the family of the chain
func (c Chain) Family() string {
	return chaincommon.ChainMetadata{Selector: c.Selector}.Family()
}

// NetworkType returns the type of network the chain is on (e.g. mainnet, testnet)
func (c Chain) NetworkType() (chainsel.NetworkType, error) {
	return chaincommon.ChainMetadata{Selector: c.Selector}.NetworkType()
}

// IsNetworkType checks if the chain is on the given network type
func (c Chain) IsNetworkType(networkType chainsel.NetworkType) bool {
	return chaincommon.ChainMetadata{Selector: c.Selector}.IsNetworkType(networkType)
}
