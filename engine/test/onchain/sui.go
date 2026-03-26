// Package onchain provides chain loaders for testing infrastructure.
package onchain

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"

	chainselectors "github.com/smartcontractkit/chain-selectors"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	suiprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/sui/provider"
)

// SuiContainerConfig is the configuration for the Sui container loader.
type SuiContainerConfig = suiprov.CTFChainProviderConfig

// NewSuiContainerLoader creates a new Sui chain loader with default configuration using CTF.
func NewSuiContainerLoader() *ChainLoader {
	// Generate a random Sui Ed25519 private key for testing
	seeded := ed25519.NewKeyFromSeed(suiRandomSeed()) // 64 bytes: seed||pub
	seed := seeded[:32]                               // or: seeded.Seed() if available
	testPrivateKey := hex.EncodeToString(seed)        // 64 hex chars

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

// NewSuiContainerLoaderWithConfig creates a new Sui chain loader with the given configuration using CTF.
func NewSuiContainerLoaderWithConfig(cfg SuiContainerConfig) *ChainLoader {
	return &ChainLoader{
		selectors: getTestSelectorsByFamily(chainselectors.FamilySui),
		factory: func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
			t.Helper()

			return suiprov.NewCTFChainProvider(t, selector, cfg).Initialize(t.Context())
		},
	}
}

// randomSeed generates a random seed for the Sui Ed25519 private key.
func suiRandomSeed() []byte {
	seed := make([]byte, ed25519.SeedSize)
	_, err := rand.Read(seed)
	if err != nil {
		panic(fmt.Sprintf("failed to generate random seed: %+v", err)) // This should never happen unless using a legacy Linux system
	}

	return seed
}
