// Package onchain provides chain loaders for testing infrastructure.
package onchain

import (
	"testing"

	chainselectors "github.com/smartcontractkit/chain-selectors"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	stellarprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/stellar/provider"
)

// NewStellarContainerLoader creates a new Stellar chain loader with default configuration using CTF.
// The loader spins up a Stellar Quickstart localnet container and generates a random deployer keypair.
func NewStellarContainerLoader() *ChainLoader {
	return &ChainLoader{
		selectors: getTestSelectorsByFamily(chainselectors.FamilyStellar),
		factory: func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
			t.Helper()

			return stellarprov.NewCTFChainProvider(t, selector, stellarprov.CTFChainProviderConfig{
				Once:               once,
				DeployerKeypairGen: stellarprov.KeypairRandom(),
			}).Initialize(t.Context())
		},
	}
}
