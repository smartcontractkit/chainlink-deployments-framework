// Package onchain provides chain loaders for testing infrastructure.
package onchain

import (
	"testing"

	chainselectors "github.com/smartcontractkit/chain-selectors"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	tonprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/ton/provider"
)

// TonContainerConfig is the configuration for the TON container loader.
// See https://github.com/neodix42/mylocalton-docker/wiki/Genesis-setup-parameters for available options.
type TonContainerConfig = tonprov.CTFChainProviderConfig

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

// NewTonContainerLoaderWithConfig creates a new TON chain loader with the given configuration using CTF.
func NewTonContainerLoaderWithConfig(cfg TonContainerConfig) *ChainLoader {
	return &ChainLoader{
		selectors: getTestSelectorsByFamily(chainselectors.FamilyTon),
		factory: func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
			t.Helper()

			return tonprov.NewCTFChainProvider(t, selector, cfg).Initialize(t.Context())
		},
	}
}
