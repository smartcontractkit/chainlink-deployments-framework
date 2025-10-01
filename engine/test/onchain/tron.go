// Package onchain provides chain loaders for testing infrastructure.
package onchain

import (
	"testing"

	chainselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	tronprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/tron/provider"
)

// NewTronContainerLoader creates a new Tron chain loader with default configuration using CTF.
func NewTronContainerLoader() *ChainLoader {
	return &ChainLoader{
		selectors: getTestSelectorsByFamily(chainselectors.FamilyTron),
		factory: func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
			t.Helper()

			signerGen, err := tronprov.SignerGenCTFDefault()
			require.NoError(t, err)

			return tronprov.NewCTFChainProvider(t, selector, tronprov.CTFChainProviderConfig{
				Once:              once,
				DeployerSignerGen: signerGen,
			}).Initialize(t.Context())
		},
	}
}
