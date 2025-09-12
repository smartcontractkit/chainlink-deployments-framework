// Package onchain provides chain loaders for testing infrastructure.
package onchain

import (
	"testing"

	chainselectors "github.com/smartcontractkit/chain-selectors"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	aptosprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos/provider"
)

// NewAptosContainerLoader creates a new Aptos chain loader with default configuration using CTF.
func NewAptosContainerLoader() *ChainLoader {
	return &ChainLoader{
		selectors: getTestSelectorsByFamily(chainselectors.FamilyAptos),
		factory: func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
			t.Helper()

			return aptosprov.NewCTFChainProvider(t, selector, aptosprov.CTFChainProviderConfig{
				Once:              once,
				DeployerSignerGen: aptosprov.AccountGenCTFDefault(),
			}).Initialize(t.Context())
		},
	}
}
