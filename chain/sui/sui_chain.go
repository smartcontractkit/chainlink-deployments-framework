package sui

import (
	"crypto/ed25519"

	"github.com/pattonkan/sui-go/suiclient"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

// Chain represents an Sui chain.
type Chain struct {
	Selector uint64
	Client   *suiclient.ClientImpl
	// TODO: sui-go currently does not have a working Signer interface, so we
	// have the raw private key for now.
	DeployerKey ed25519.PrivateKey
	URL         string

	Confirm func(txHash string, opts ...any) error
}

// ChainSelector returns the chain selector of the chain
func (c Chain) ChainSelector() uint64 {
	return c.Selector
}

// String returns chain name and selector "<name> (<selector>)"
func (c Chain) String() string {
	return common.ChainMetadata{Selector: c.Selector}.String()
}

// Name returns the name of the chain
func (c Chain) Name() string {
	return common.ChainMetadata{Selector: c.Selector}.Name()
}

// Family returns the family of the chain
func (c Chain) Family() string {
	return common.ChainMetadata{Selector: c.Selector}.Family()
}
