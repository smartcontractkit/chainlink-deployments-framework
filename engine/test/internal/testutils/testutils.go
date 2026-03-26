// Package testutils provides utility functions for testing the test engine.
package testutils

import (
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
)

// Ensure the StubChain implements the fchain.BlockChain interface.
var _ fchain.BlockChain = &StubChain{}

// StubChain is an lightweight implementation of the fchain.BlockChain interface for testing
// purposes.
type StubChain struct{ selector uint64 }

// NewStubChain creates a new StubChain with the given selector.
func NewStubChain(selector uint64) *StubChain {
	return &StubChain{selector: selector}
}

// Implement the fchain.BlockChain interface.
func (c *StubChain) ChainSelector() uint64 { return c.selector }
func (c *StubChain) Family() string        { return "test" }
func (c *StubChain) Name() string          { return "test" }
func (c *StubChain) String() string        { return fmt.Sprintf("testChain(%d)", c.selector) }
func (c *StubChain) NetworkType() (chainsel.NetworkType, error) {
	return chainsel.NetworkTypeTestnet, nil
}
func (c *StubChain) IsNetworkType(networkType chainsel.NetworkType) bool {
	return networkType == chainsel.NetworkTypeTestnet
}
