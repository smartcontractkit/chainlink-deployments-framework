package aptos

import (
	"github.com/aptos-labs/aptos-go-sdk"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal"
)

// Chain represents an Aptos chain.
type Chain struct {
	Selector uint64

	Client         aptos.AptosRpcClient
	DeployerSigner aptos.TransactionSigner
	URL            string

	Confirm func(txHash string, opts ...any) error
}

// ChainSelector returns the chain selector of the chain
func (c Chain) ChainSelector() uint64 {
	return c.Selector
}

// String returns chain name and selector "<name> (<selector>)"
func (c Chain) String() string {
	return internal.ChainBase{Selector: c.Selector}.String()
}

// Name returns the name of the chain
func (c Chain) Name() string {
	return internal.ChainBase{Selector: c.Selector}.Name()
}
