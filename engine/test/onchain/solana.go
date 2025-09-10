// Package onchain provides chain loaders for testing infrastructure.
package onchain

import (
	"testing"
	"time"

	chainselectors "github.com/smartcontractkit/chain-selectors"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	solanaprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/solana/provider"
)

// NewSolanaContainerLoader creates a new Solana chain loader with program configuration using CTF.
// The programsPath specifies the directory containing Solana programs, and programIDs maps
// program names to their deployment addresses.
func NewSolanaContainerLoader(
	programsPath string, programIDs map[string]string,
) *ChainLoader {
	return &ChainLoader{
		selectors: getTestSelectorsByFamily(chainselectors.FamilySolana),
		factory: func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
			t.Helper()

			return solanaprov.NewCTFChainProvider(t, selector, solanaprov.CTFChainProviderConfig{
				Once:                         once,
				DeployerKeyGen:               solanaprov.PrivateKeyRandom(),
				ProgramsPath:                 programsPath,
				ProgramIDs:                   programIDs,
				WaitDelayAfterContainerStart: 15 * time.Second, // we have slot errors that force retries if the chain is not given enough time to boot
			}).Initialize(t.Context())
		},
	}
}
