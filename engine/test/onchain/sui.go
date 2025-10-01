// Package onchain provides chain loaders for testing infrastructure.
package onchain

import (
	"testing"

	chainselectors "github.com/smartcontractkit/chain-selectors"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	suiprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/sui/provider"
)

// NewSuiContainerLoader creates a new Sui chain loader with default configuration using CTF.
func NewSuiContainerLoader() *ChainLoader {
	// testPrivateKey is a valid Sui Ed25519 private key for testing purposes (32 bytes, 64 hex chars)
	testPrivateKey := "E4FD0E90D32CB98DC6AD64516A421E8C2731870217CDBA64203CEB158A866304"

	return &ChainLoader{
		selectors: getTestSelectorsByFamily(chainselectors.FamilySui),
		factory: func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
			t.Helper()

			return suiprov.NewCTFChainProvider(t, selector, suiprov.CTFChainProviderConfig{
				Once:              once,
				DeployerSignerGen: suiprov.AccountGenPrivateKey(testPrivateKey),
			}).Initialize(t.Context())
		},
	}
}
