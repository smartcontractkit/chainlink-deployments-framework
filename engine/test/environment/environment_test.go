package environment

import (
	"errors"
	"maps"
	"testing"
	"time"

	chainselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/stretchr/testify/require"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	fchainaptos "github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	fchainevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	fchainsolana "github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	fchainsui "github.com/smartcontractkit/chainlink-deployments-framework/chain/sui"
	fchainton "github.com/smartcontractkit/chainlink-deployments-framework/chain/ton"
	fchaintron "github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/test/onchain"
	foffchainjd "github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd"
)

func TestNew(t *testing.T) {
	t.Parallel()

	env, err := New(t.Context(), WithLogger(logger.Test(t)))
	require.NoError(t, err)
	require.NotNil(t, env)
}

func TestLoader_Load_Options(t *testing.T) {
	t.Parallel()

	// Helper functions for creating test options
	successOpt := func(cmps *components) error {
		return nil
	}

	errorOpt := func(msg string) LoadOpt {
		return func(cmps *components) error {
			return errors.New(msg)
		}
	}

	tests := []struct {
		name            string
		opts            []LoadOpt
		wantErrContains []string
	}{
		{
			name: "succeeds with no options",
			opts: []LoadOpt{},
		},
		{
			name: "succeeds with single success option",
			opts: []LoadOpt{successOpt},
		},
		{
			name: "succeeds with multiple success options",
			opts: []LoadOpt{successOpt, successOpt},
		},
		{
			name:            "returns error when single option fails",
			opts:            []LoadOpt{errorOpt("test error")},
			wantErrContains: []string{"test error"},
		},
		{
			name:            "returns combined errors when multiple options fail",
			opts:            []LoadOpt{errorOpt("error 1"), successOpt, errorOpt("error 2")},
			wantErrContains: []string{"error 1", "error 2"},
		},
		{
			name:            "returns error when all options fail",
			opts:            []LoadOpt{errorOpt("first error"), errorOpt("second error")},
			wantErrContains: []string{"first error", "second error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loader := NewLoader()
			env, err := loader.Load(t.Context(), tt.opts...)

			if len(tt.wantErrContains) > 0 {
				require.Error(t, err)
				require.Nil(t, env)
				for _, errMsg := range tt.wantErrContains {
					require.ErrorContains(t, err, errMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, env)
				require.NotNil(t, env.Offchain) // Offchain should always be initialized
			}
		})
	}
}

func TestLoader_Load_LoggerOption(t *testing.T) {
	t.Parallel()
	lggr := logger.Test(t)

	loader := NewLoader()
	env, err := loader.Load(t.Context(), WithLogger(lggr))
	require.NoError(t, err)
	require.NotNil(t, env)
	require.Equal(t, lggr, env.Logger)
}

func TestLoader_Load_DatastoreOption(t *testing.T) {
	t.Parallel()

	ds := fdatastore.NewMemoryDataStore().Seal()

	loader := NewLoader()
	env, err := loader.Load(t.Context(), WithDatastore(ds))
	require.NoError(t, err)
	require.NotNil(t, env)
	require.Equal(t, ds, env.DataStore)
}

func TestLoader_Load_OffchainClientOption(t *testing.T) {
	t.Parallel()

	oc := &foffchainjd.JobDistributor{}

	loader := NewLoader()
	env, err := loader.Load(t.Context(), WithOffchainClient(oc))
	require.NoError(t, err)
	require.NotNil(t, env)
	require.Equal(t, oc, env.Offchain)
}

func TestLoader_Load_NodeIDsOption(t *testing.T) {
	t.Parallel()

	nodeIDs := []string{"1", "2", "3"}

	loader := NewLoader()
	env, err := loader.Load(t.Context(), WithNodeIDs(nodeIDs))
	require.NoError(t, err)
	require.NotNil(t, env)
	require.Equal(t, nodeIDs, env.NodeIDs)
}

func TestLoader_Load_AddressBookOption(t *testing.T) {
	t.Parallel()

	ab := fdeployment.NewMemoryAddressBook()

	loader := NewLoader()
	env, err := loader.Load(t.Context(), WithAddressBook(ab))
	require.NoError(t, err)
	require.NotNil(t, env)
	require.Equal(t, ab, env.ExistingAddresses) //nolint:staticcheck // SA1019 (Deprecated): We still need to support AddressBook for now
}

func TestLoader_Load_ChainOptions(t *testing.T) { //nolint:paralleltest // We are replacing local variables here, so we can't run tests in parallel.
	// Stub out the container loaders to avoid having to spin up containers for each test
	resetLoadersFunc := stubContainerLoaders()
	t.Cleanup(resetLoadersFunc)

	tests := []struct {
		name               string
		opts               []LoadOpt
		wantBlockChainsLen int
		assert             func(t *testing.T, BlockChains fchain.BlockChains)
	}{
		{
			name:               "succeeds with no options resulting in no block chains",
			opts:               []LoadOpt{},
			wantBlockChainsLen: 0,
		},
		{
			name:               "EVMSimulated with selectors",
			opts:               []LoadOpt{WithEVMSimulated(t, []uint64{chainselectors.TEST_90000001.Selector})},
			wantBlockChainsLen: 1,
			assert: func(t *testing.T, BlockChains fchain.BlockChains) {
				t.Helper()

				require.Len(t, BlockChains.EVMChains(), 1)
			},
		},
		{
			name:               "EVMSimulatedN",
			opts:               []LoadOpt{WithEVMSimulatedN(t, 1)},
			wantBlockChainsLen: 1,
			assert: func(t *testing.T, BlockChains fchain.BlockChains) {
				t.Helper()

				require.Len(t, BlockChains.EVMChains(), 1)
			},
		},
		{
			name: "EVMSimulatedWithConfig with selectors",
			opts: []LoadOpt{WithEVMSimulatedWithConfig(t, []uint64{chainselectors.TEST_90000001.Selector}, onchain.EVMSimLoaderConfig{
				NumAdditionalAccounts: 1,
				BlockTime:             1 * time.Second,
			})},
			wantBlockChainsLen: 1,
			assert: func(t *testing.T, BlockChains fchain.BlockChains) {
				t.Helper()

				require.Len(t, BlockChains.EVMChains(), 1)
			},
		},
		{
			name: "EVMSimulatedWithConfigN",
			opts: []LoadOpt{WithEVMSimulatedWithConfigN(t, 1, onchain.EVMSimLoaderConfig{
				NumAdditionalAccounts: 1,
				BlockTime:             1 * time.Second,
			})},
			wantBlockChainsLen: 1,
			assert: func(t *testing.T, BlockChains fchain.BlockChains) {
				t.Helper()

				require.Len(t, BlockChains.EVMChains(), 1)
			},
		},
		{
			name: "Containerized chains with selectors",
			opts: []LoadOpt{
				WithZKSyncContainer(t, []uint64{chainselectors.TEST_90000051.Selector}),
				WithSolanaContainer(t, []uint64{chainselectors.TEST_22222222222222222222222222222222222222222222.Selector}, t.TempDir(), map[string]string{}),
				WithAptosContainer(t, []uint64{chainselectors.APTOS_LOCALNET.Selector}),
				WithTonContainer(t, []uint64{chainselectors.TON_LOCALNET.Selector}),
				WithTronContainer(t, []uint64{chainselectors.TRON_DEVNET.Selector}),
				WithSuiContainer(t, []uint64{chainselectors.SUI_LOCALNET.Selector}),
			},
			wantBlockChainsLen: 6,
			assert: func(t *testing.T, BlockChains fchain.BlockChains) {
				t.Helper()

				require.Len(t, BlockChains.EVMChains(), 1) // zksync is an EVM chain
				require.Len(t, BlockChains.SolanaChains(), 1)
				require.Len(t, BlockChains.AptosChains(), 1)
				require.Len(t, BlockChains.TonChains(), 1)
				require.Len(t, BlockChains.TronChains(), 1)
				require.Len(t, BlockChains.SuiChains(), 1)
			},
		},
		{
			name: "Containerized chains with n count",
			opts: []LoadOpt{
				WithZKSyncContainerN(t, 1),
				WithSolanaContainerN(t, 1, t.TempDir(), map[string]string{}),
				WithAptosContainerN(t, 1),
				WithTonContainerN(t, 1),
				WithTronContainerN(t, 1),
				WithSuiContainerN(t, 1),
			},
			wantBlockChainsLen: 6,
			assert: func(t *testing.T, BlockChains fchain.BlockChains) {
				t.Helper()

				require.Len(t, BlockChains.EVMChains(), 1) // zksync is an EVM chain
				require.Len(t, BlockChains.SolanaChains(), 1)
				require.Len(t, BlockChains.AptosChains(), 1)
				require.Len(t, BlockChains.TonChains(), 1)
				require.Len(t, BlockChains.TronChains(), 1)
				require.Len(t, BlockChains.SuiChains(), 1)
			},
		},
	}

	for _, tt := range tests { //nolint:paralleltest // We are replacing local variables here, so we can't run tests in parallel.
		t.Run(tt.name, func(t *testing.T) {
			loader := NewLoader()
			env, err := loader.Load(t.Context(), tt.opts...)
			require.NoError(t, err)
			require.NotNil(t, env)
			require.Len(t, maps.Collect(env.BlockChains.All()), tt.wantBlockChainsLen)

			if tt.assert != nil {
				tt.assert(t, env.BlockChains)
			}
		})
	}
}

// stubContainerLoaders stubs out the container loaders to avoid having to spin up containers for each test.
//
// It does this by assigning the container loader constructors to local variables to allow for stubbing in tests.
// Returns a function that can be used to reset the loaders to their original values.
func stubContainerLoaders() func() {
	var (
		oldTonContainerLoader    = newTonContainerLoader
		oldAptosContainerLoader  = newAptosContainerLoader
		oldSolanaContainerLoader = newSolanaContainerLoader
		oldZKSyncContainerLoader = newZKSyncContainerLoader
		oldTronContainerLoader   = newTronContainerLoader
		oldSuiContainerLoader    = newSuiContainerLoader
	)

	newTonContainerLoader = makeChainLoaderStub([]uint64{chainselectors.TON_LOCALNET.Selector}, fchainton.Chain{
		ChainMetadata: fchainton.ChainMetadata{
			Selector: chainselectors.TON_LOCALNET.Selector,
		},
	})
	newAptosContainerLoader = makeChainLoaderStub([]uint64{chainselectors.APTOS_LOCALNET.Selector}, fchainaptos.Chain{
		Selector: chainselectors.APTOS_LOCALNET.Selector,
	})

	// We have to use a custom function here because the Solana container loader constructor has arguments.
	newSolanaContainerLoader = func(programsPath string, programIDs map[string]string) *onchain.ChainLoader {
		return onchain.NewChainLoader(
			[]uint64{chainselectors.TEST_22222222222222222222222222222222222222222222.Selector},
			func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
				t.Helper()

				return fchainsolana.Chain{}, nil
			})
	}
	newZKSyncContainerLoader = makeChainLoaderStub([]uint64{chainselectors.TEST_90000051.Selector}, fchainevm.Chain{
		Selector:   chainselectors.TEST_90000051.Selector,
		IsZkSyncVM: true,
	})
	newTronContainerLoader = makeChainLoaderStub([]uint64{chainselectors.TRON_DEVNET.Selector}, fchaintron.Chain{
		ChainMetadata: fchaintron.ChainMetadata{
			Selector: chainselectors.TRON_DEVNET.Selector,
		},
	})
	newSuiContainerLoader = makeChainLoaderStub([]uint64{chainselectors.SUI_LOCALNET.Selector}, fchainsui.Chain{
		ChainMetadata: fchainsui.ChainMetadata{
			Selector: chainselectors.SUI_LOCALNET.Selector,
		},
	})

	return func() {
		newTonContainerLoader = oldTonContainerLoader
		newAptosContainerLoader = oldAptosContainerLoader
		newSolanaContainerLoader = oldSolanaContainerLoader
		newZKSyncContainerLoader = oldZKSyncContainerLoader
		newTronContainerLoader = oldTronContainerLoader
		newSuiContainerLoader = oldSuiContainerLoader
	}
}

// stubChainLoader is a helper function to create a chain loader that returns a stub chain. This covers the simple
// case when a chain loader constructor has no arguments.
func makeChainLoaderStub(selectors []uint64, chain fchain.BlockChain) func() *onchain.ChainLoader {
	return func() *onchain.ChainLoader {
		return onchain.NewChainLoader(selectors, func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
			t.Helper()

			return chain, nil
		})
	}
}
