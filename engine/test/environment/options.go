package environment

import (
	"testing"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/test/onchain"
	"github.com/smartcontractkit/chainlink-deployments-framework/offchain"
)

// Assign the chain container loader constructors to local variables to allow for stubbing in tests.
var (
	newAptosContainerLoader         = onchain.NewAptosContainerLoader
	newSolanaContainerLoader        = onchain.NewSolanaContainerLoader
	newSuiContainerLoader           = onchain.NewSuiContainerLoader
	newSuiContainerLoaderWithConfig = onchain.NewSuiContainerLoaderWithConfig
	newTonContainerLoader           = onchain.NewTonContainerLoader
	newTonContainerLoaderWithConfig = onchain.NewTonContainerLoaderWithConfig
	newTronContainerLoader          = onchain.NewTronContainerLoader
	newZKSyncContainerLoader        = onchain.NewZKSyncContainerLoader
)

// LoadOpt is a configuration function that sets environment components during loading.
type LoadOpt func(*components) error

// WithChains adds pre-constructed blockchain instances to the environment.
//
// Use this option when you need to manually construct and configure chains before adding
// them to the environment. For most test scenarios, prefer using chain-specific loaders
// like WithEVMSimulated, WithSolanaContainer, etc., which handle chain setup automatically.
func WithChains(chains ...fchain.BlockChain) LoadOpt {
	return func(cmps *components) error {
		cmps.AddChains(chains...)

		return nil
	}
}

// WithTonContainer loads TON blockchain container instances for specified chain selectors.
func WithTonContainer(t *testing.T, selectors []uint64) LoadOpt {
	t.Helper()

	return withChainLoader(t, newTonContainerLoader(), selectors)
}

// WithTonContainerWithConfig loads TON blockchain container instances with custom configuration
// for specified chain selectors.
func WithTonContainerWithConfig(t *testing.T, selectors []uint64, cfg onchain.TonContainerConfig) LoadOpt {
	t.Helper()

	return withChainLoader(t, newTonContainerLoaderWithConfig(cfg), selectors)
}

// WithTonContainerN loads n TON blockchain container instances.
func WithTonContainerN(t *testing.T, n int) LoadOpt {
	t.Helper()

	return withChainLoaderN(t, newTonContainerLoader(), n)
}

// WithTonContainerNWithConfig loads n TON blockchain container instances with custom configuration.
func WithTonContainerNWithConfig(t *testing.T, n int, cfg onchain.TonContainerConfig) LoadOpt {
	t.Helper()

	return withChainLoaderN(t, newTonContainerLoaderWithConfig(cfg), n)
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

// WithSuiContainerWithConfig loads Sui blockchain container instances with custom configuration
// for specified chain selectors.
func WithSuiContainerWithConfig(t *testing.T, selectors []uint64, cfg onchain.SuiContainerConfig) LoadOpt {
	t.Helper()

	return withChainLoader(t, newSuiContainerLoaderWithConfig(cfg), selectors)
}

// WithSuiContainerN loads n Sui blockchain container instances.
func WithSuiContainerN(t *testing.T, n int) LoadOpt {
	t.Helper()

	return withChainLoaderN(t, newSuiContainerLoader(), n)
}

// WithSuiContainerNWithConfig loads n Sui blockchain container instances with custom configuration.
func WithSuiContainerNWithConfig(t *testing.T, n int, cfg onchain.SuiContainerConfig) LoadOpt {
	t.Helper()

	return withChainLoaderN(t, newSuiContainerLoaderWithConfig(cfg), n)
}

// WithLogger sets the logger for the environment.
func WithLogger(lggr logger.Logger) LoadOpt {
	return func(cmps *components) error {
		cmps.Logger = lggr
		return nil
	}
}

// WithDatastore sets a custom datastore for the environment.
func WithDatastore(ds fdatastore.DataStore) LoadOpt {
	return func(cmps *components) error {
		cmps.Datastore = ds
		return nil
	}
}

// WithAddressBook sets a custom address book for the environment.
func WithAddressBook(ab fdeployment.AddressBook) LoadOpt {
	return func(cmps *components) error {
		cmps.AddressBook = ab
		return nil
	}
}

// WithOffchainClient sets a custom offchain client for the environment.
func WithOffchainClient(oc offchain.Client) LoadOpt {
	return func(cmps *components) error {
		cmps.OffchainClient = oc
		return nil
	}
}

// WithNodeIDs sets a custom node IDs for the environment.
func WithNodeIDs(nodeIDs []string) LoadOpt {
	return func(cmps *components) error {
		cmps.NodeIDs = nodeIDs
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
