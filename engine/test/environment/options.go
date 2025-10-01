package environment

import (
	"testing"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/test/onchain"
)

// Assign the chain container loader constructors to local variables to allow for stubbing in tests.
var (
	newAptosContainerLoader  = onchain.NewAptosContainerLoader
	newSolanaContainerLoader = onchain.NewSolanaContainerLoader
	newSuiContainerLoader    = onchain.NewSuiContainerLoader
	newTonContainerLoader    = onchain.NewTonContainerLoader
	newTronContainerLoader   = onchain.NewTronContainerLoader
	newZKSyncContainerLoader = onchain.NewZKSyncContainerLoader
)

// LoadOpt is a configuration function that sets environment components during loading.
type LoadOpt func(*components) error

// WithTonContainer loads TON blockchain container instances for specified chain selectors.
func WithTonContainer(t *testing.T, selectors []uint64) LoadOpt {
	t.Helper()

	return withChainLoader(t, newTonContainerLoader(), selectors)
}

// WithTonContainerN loads n TON blockchain container instances.
func WithTonContainerN(t *testing.T, n int) LoadOpt {
	t.Helper()

	return withChainLoaderN(t, newTonContainerLoader(), n)
}

// WithEVMSimulated loads simulated EVM blockchain instances for specified chain selectors.
//
// Uses in-memory simulation without Docker containers for faster test execution.
func WithEVMSimulated(t *testing.T, selectors []uint64) LoadOpt {
	t.Helper()

	return withChainLoader(t, onchain.NewEVMSimLoader(), selectors)
}

// WithEVMSimulatedN loads n simulated EVM blockchain instances.
func WithEVMSimulatedN(t *testing.T, n int) LoadOpt {
	t.Helper()

	return withChainLoaderN(t, onchain.NewEVMSimLoader(), n)
}

// WithEVMSimulatedWithConfig loads simulated EVM blockchain instances with custom configuration
// for specified chain selectors.
func WithEVMSimulatedWithConfig(t *testing.T, selectors []uint64, cfg onchain.EVMSimLoaderConfig) LoadOpt {
	t.Helper()

	return withChainLoader(t, onchain.NewEVMSimLoaderWithConfig(cfg), selectors)
}

// WithEVMSimulatedWithConfigN loads n simulated EVM blockchain instances with custom configuration.
func WithEVMSimulatedWithConfigN(t *testing.T, n int, cfg onchain.EVMSimLoaderConfig) LoadOpt {
	t.Helper()

	return withChainLoaderN(t, onchain.NewEVMSimLoaderWithConfig(cfg), n)
}

// WithAptosContainer loads Aptos blockchain container instances for specified chain selectors.
func WithAptosContainer(t *testing.T, selectors []uint64) LoadOpt {
	t.Helper()

	return withChainLoader(t, newAptosContainerLoader(), selectors)
}

// WithAptosContainerN loads n Aptos blockchain container instances.
func WithAptosContainerN(t *testing.T, n int) LoadOpt {
	t.Helper()

	return withChainLoaderN(t, newAptosContainerLoader(), n)
}

// WithSolanaContainer loads Solana blockchain container instances for specified chain selectors.
//
// Requires programsPath and programIDs for initial container setup.
func WithSolanaContainer(
	t *testing.T, selectors []uint64, programsPath string, programIDs map[string]string,
) LoadOpt {
	t.Helper()

	return withChainLoader(t, newSolanaContainerLoader(programsPath, programIDs), selectors)
}

// WithSolanaContainerN loads n Solana blockchain instances using Docker containers.
//
// Requires programsPath and programIDs for initial container setup.
func WithSolanaContainerN(
	t *testing.T, n int, programsPath string, programIDs map[string]string,
) LoadOpt {
	t.Helper()

	return withChainLoaderN(t, newSolanaContainerLoader(programsPath, programIDs), n)
}

// WithZKSyncContainer loads ZKSync blockchain container instances for specified chain selectors.
func WithZKSyncContainer(t *testing.T, selectors []uint64) LoadOpt {
	t.Helper()

	return withChainLoader(t, newZKSyncContainerLoader(), selectors)
}

// WithZKSyncContainerN loads n ZKSync blockchain container instances.
func WithZKSyncContainerN(t *testing.T, n int) LoadOpt {
	t.Helper()

	return withChainLoaderN(t, newZKSyncContainerLoader(), n)
}

// WithTronContainer loads Tron blockchain container instances for specified chain selectors.
func WithTronContainer(t *testing.T, selectors []uint64) LoadOpt {
	t.Helper()

	return withChainLoader(t, newTronContainerLoader(), selectors)
}

// WithTronContainerN loads n Tron blockchain container instances.
func WithTronContainerN(t *testing.T, n int) LoadOpt {
	t.Helper()

	return withChainLoaderN(t, newTronContainerLoader(), n)
}

// WithSuiContainer loads Sui blockchain container instances for specified chain selectors.
func WithSuiContainer(t *testing.T, selectors []uint64) LoadOpt {
	t.Helper()

	return withChainLoader(t, newSuiContainerLoader(), selectors)
}

// WithSuiContainerN loads n Sui blockchain container instances.
func WithSuiContainerN(t *testing.T, n int) LoadOpt {
	t.Helper()

	return withChainLoaderN(t, newSuiContainerLoader(), n)
}

// WithLogger sets the logger for the environment.
func WithLogger(lggr logger.Logger) LoadOpt {
	return func(cmps *components) error {
		cmps.Logger = lggr
		return nil
	}
}

// withChainLoader creates a LoadOpt that loads chains using the provided loader and selectors.
func withChainLoader(t *testing.T, loader *onchain.ChainLoader, selectors []uint64) LoadOpt {
	t.Helper()

	return func(cmps *components) error {
		chains, err := loader.Load(t, selectors)
		if err != nil {
			return err
		}

		cmps.AddChains(chains...)

		return nil
	}
}

// withChainLoaderN creates a LoadOpt that loads n chains using the provided loader.
func withChainLoaderN(t *testing.T, loader *onchain.ChainLoader, n int) LoadOpt {
	t.Helper()

	return func(cmps *components) error {
		chains, err := loader.LoadN(t, n)
		if err != nil {
			return err
		}

		cmps.AddChains(chains...)

		return nil
	}
}
