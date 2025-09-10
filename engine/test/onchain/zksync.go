// Package onchain provides chain loaders for testing infrastructure.
package onchain

import (
	"testing"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	evmprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider"
)

// NewZKSyncContainerLoader creates a new ZKSync EVM chain loader with predefined test selectors
// using CTF.
//
// ZKSync chains use dedicated test selectors starting from TEST_90000051 to avoid conflicts
// with standard EVM test selectors. The loader supports up to 10 concurrent ZKSync test chains.
func NewZKSyncContainerLoader() *ChainLoader {
	return &ChainLoader{
		// ZKSync selectors are defined here rather than in testSelectors to avoid bloating
		// the EVM family with ZKSync-specific complexity. Uses TEST_90000051+ range.
		selectors: zksyncSelectors,
		factory: func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
			t.Helper()

			return evmprov.NewZkSyncCTFChainProvider(t, selector,
				evmprov.ZkSyncCTFChainProviderConfig{
					Once: once,
				},
			).Initialize(t.Context())
		},
	}
}
