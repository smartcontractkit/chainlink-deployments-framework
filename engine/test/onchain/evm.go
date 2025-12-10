// Package onchain provides chain loaders for testing infrastructure.
package onchain

import (
	"testing"
	"time"

	chainselectors "github.com/smartcontractkit/chain-selectors"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	evmprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider"
)

// EVMSimLoaderConfig holds configuration options for EVM simulated chain loading.
type EVMSimLoaderConfig struct {
	NumAdditionalAccounts uint          // Number of additional pre-funded accounts to create
	BlockTime             time.Duration // Time interval between automatic block mining (0 = manual)
}

// NewEVMSimLoader creates a new EVM chain loader with default simulated backend configuration.
// Uses go-ethereum's simulated backend with default settings for fast test execution.
func NewEVMSimLoader() *ChainLoader {
	selectors := getTestSelectorsByFamily(chainselectors.FamilyEVM)
	selectors = append([]uint64{chainselectors.GETH_TESTNET.Selector}, selectors...)

	return &ChainLoader{
		selectors: selectors,
		factory: func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
			t.Helper()

			return evmprov.NewSimChainProvider(t, selector, evmprov.SimChainProviderConfig{}).
				Initialize(t.Context())
		},
	}
}

// NewEVMSimLoaderWithConfig creates a new EVM chain loader with custom configuration.
// Allows specification of additional accounts and block mining intervals for advanced testing scenarios.
func NewEVMSimLoaderWithConfig(cfg EVMSimLoaderConfig) *ChainLoader {
	l := NewEVMSimLoader()

	l.factory = func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
		t.Helper()

		return evmprov.NewSimChainProvider(t, selector, evmprov.SimChainProviderConfig{
			NumAdditionalAccounts: cfg.NumAdditionalAccounts,
			BlockTime:             cfg.BlockTime,
		}).Initialize(t.Context())
	}

	return l
}
