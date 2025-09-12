// Package onchain provides chain loaders for testing infrastructure.
package onchain

import (
	"testing"

	chainselectors "github.com/smartcontractkit/chain-selectors"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	tonprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/ton/provider"
)

// NewTonContainerLoader creates a new TON chain loader with default configuration using CTF.
func NewTonContainerLoader() *ChainLoader {
	return &ChainLoader{
		selectors: getTestSelectorsByFamily(chainselectors.FamilyTon),
		factory: func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
			t.Helper()

			return tonprov.NewCTFChainProvider(t, selector, tonprov.CTFChainProviderConfig{
				Once: once,
			}).Initialize(t.Context())
		},
	}
}
