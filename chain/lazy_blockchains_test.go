package chain_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// Mock chain loader for testing lazy loading
type mockChainLoader struct {
	loadFunc  func(selector uint64) (chain.BlockChain, error)
	loadCalls []uint64
}

func (m *mockChainLoader) Load(ctx context.Context, selector uint64) (chain.BlockChain, error) {
	m.loadCalls = append(m.loadCalls, selector)
	return m.loadFunc(selector)
}

func TestLazyBlockChains_GetBySelector(t *testing.T) {
	t.Parallel()

	t.Run("loads chain on first access", func(t *testing.T) {
		t.Parallel()

		loader := &mockChainLoader{
			loadFunc: func(selector uint64) (chain.BlockChain, error) {
				if selector == evmChain1.Selector {
					return evmChain1, nil
				}

				return nil, chain.ErrBlockChainNotFound
			},
		}

		availableChains := map[uint64]string{
			evmChain1.Selector: chainsel.FamilyEVM,
		}
		loaders := map[string]chain.ChainLoader{
			chainsel.FamilyEVM: loader,
		}

		lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

		// First access should load the chain
		got, err := lazyChains.GetBySelector(evmChain1.Selector)
		require.NoError(t, err)
		assert.Equal(t, evmChain1, got)
		assert.Len(t, loader.loadCalls, 1, "chain should be loaded once")

		// Second access should use cache
		got, err = lazyChains.GetBySelector(evmChain1.Selector)
		require.NoError(t, err)
		assert.Equal(t, evmChain1, got)
		assert.Len(t, loader.loadCalls, 1, "chain should not be loaded again")
	})

	t.Run("returns error for unavailable chain", func(t *testing.T) {
		t.Parallel()

		loader := &mockChainLoader{
			loadFunc: func(selector uint64) (chain.BlockChain, error) {
				return evmChain1, nil
			},
		}

		availableChains := map[uint64]string{
			evmChain1.Selector: chainsel.FamilyEVM,
		}
		loaders := map[string]chain.ChainLoader{
			chainsel.FamilyEVM: loader,
		}

		lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

		// Accessing non-existent chain should return error
		_, err := lazyChains.GetBySelector(99999999)
		require.Error(t, err)
		require.ErrorIs(t, err, chain.ErrBlockChainNotFound)
		assert.Empty(t, loader.loadCalls, "loader should not be called for unavailable chains")
	})
}

func TestLazyBlockChains_Exists(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			return evmChain1, nil
		},
	}

	availableChains := map[uint64]string{
		evmChain1.Selector: chainsel.FamilyEVM,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyEVM: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Should return true for available chain without loading
	assert.True(t, lazyChains.Exists(evmChain1.Selector))
	assert.Empty(t, loader.loadCalls, "Exists should not load the chain")

	// Should return false for unavailable chain
	assert.False(t, lazyChains.Exists(99999999))
}

func TestLazyBlockChains_EVMChains(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			switch selector {
			case evmChain1.Selector:
				return evmChain1, nil
			case evmChain2.Selector:
				return evmChain2, nil
			default:
				return nil, chain.ErrBlockChainNotFound
			}
		},
	}

	availableChains := map[uint64]string{
		evmChain1.Selector:    chainsel.FamilyEVM,
		evmChain2.Selector:    chainsel.FamilyEVM,
		solanaChain1.Selector: chainsel.FamilySolana,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyEVM:    loader,
		chainsel.FamilySolana: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Get EVM chains should load only EVM chains
	evmChains := lazyChains.EVMChains()
	assert.Len(t, evmChains, 2, "should return 2 EVM chains")
	assert.Contains(t, evmChains, evmChain1.Selector)
	assert.Contains(t, evmChains, evmChain2.Selector)

	// Loader should be called for EVM chains only
	assert.ElementsMatch(t, []uint64{evmChain1.Selector, evmChain2.Selector}, loader.loadCalls)
}

func TestLazyBlockChains_All(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			switch selector {
			case evmChain1.Selector:
				return evmChain1, nil
			case solanaChain1.Selector:
				return solanaChain1, nil
			default:
				return nil, chain.ErrBlockChainNotFound
			}
		},
	}

	availableChains := map[uint64]string{
		evmChain1.Selector:    chainsel.FamilyEVM,
		solanaChain1.Selector: chainsel.FamilySolana,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyEVM:    loader,
		chainsel.FamilySolana: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Iterate through all chains
	count := 0
	for selector, c := range lazyChains.All() {
		count++
		assert.NotNil(t, c)
		assert.True(t, selector == evmChain1.Selector || selector == solanaChain1.Selector)
	}

	assert.Equal(t, 2, count, "should iterate over 2 chains")
	assert.Len(t, loader.loadCalls, 2, "should load all chains during iteration")
}

func TestLazyBlockChains_ListChainSelectors(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			return evmChain1, nil // Return a valid chain instead of nil, nil
		},
	}

	availableChains := map[uint64]string{
		evmChain1.Selector:    chainsel.FamilyEVM,
		evmChain2.Selector:    chainsel.FamilyEVM,
		solanaChain1.Selector: chainsel.FamilySolana,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyEVM:    loader,
		chainsel.FamilySolana: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// List all selectors
	selectors := lazyChains.ListChainSelectors()
	assert.Len(t, selectors, 3, "should list 3 selectors")
	assert.Empty(t, loader.loadCalls, "ListChainSelectors should not load chains")

	// Filter by family
	evmSelectors := lazyChains.ListChainSelectors(chain.WithFamily(chainsel.FamilyEVM))
	assert.Len(t, evmSelectors, 2, "should list 2 EVM selectors")
	assert.ElementsMatch(t, []uint64{evmChain1.Selector, evmChain2.Selector}, evmSelectors)
}

func TestLazyBlockChains_ToBlockChains(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			switch selector {
			case evmChain1.Selector:
				return evmChain1, nil
			case solanaChain1.Selector:
				return solanaChain1, nil
			default:
				return nil, chain.ErrBlockChainNotFound
			}
		},
	}

	availableChains := map[uint64]string{
		evmChain1.Selector:    chainsel.FamilyEVM,
		solanaChain1.Selector: chainsel.FamilySolana,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyEVM:    loader,
		chainsel.FamilySolana: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Convert to regular BlockChains
	blockChains, err := lazyChains.ToBlockChains()
	require.NoError(t, err)

	// Should load all chains
	assert.Len(t, loader.loadCalls, 2, "should load all chains")

	// Verify chains are accessible
	got, err := blockChains.GetBySelector(evmChain1.Selector)
	require.NoError(t, err)
	assert.Equal(t, evmChain1, got)

	got, err = blockChains.GetBySelector(solanaChain1.Selector)
	require.NoError(t, err)
	assert.Equal(t, solanaChain1, got)
}

func TestLazyBlockChains_ToBlockChains_WithError(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			if selector == evmChain1.Selector {
				return evmChain1, nil
			}
			// Simulate load error for other chains
			return nil, assert.AnError
		},
	}

	availableChains := map[uint64]string{
		evmChain1.Selector:    chainsel.FamilyEVM,
		solanaChain1.Selector: chainsel.FamilySolana,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyEVM:    loader,
		chainsel.FamilySolana: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// ToBlockChains should fail if any chain fails to load
	_, err := lazyChains.ToBlockChains()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load chain")
}

func TestLazyBlockChains_EVMChains_LoadError(t *testing.T) {
	t.Parallel()

	// Create a logger that we can check for error logs
	lggr, logs := logger.TestObserved(t, zapcore.DebugLevel)

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			if selector == evmChain1.Selector {
				return evmChain1, nil
			}
			// Simulate a load error for evmChain2
			return nil, assert.AnError
		},
	}

	availableChains := map[uint64]string{
		evmChain1.Selector: chainsel.FamilyEVM,
		evmChain2.Selector: chainsel.FamilyEVM,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyEVM: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, lggr)

	// Get EVM chains - should get the successful one and skip the failed one
	evmChains := lazyChains.EVMChains()
	assert.Len(t, evmChains, 1, "should return only successfully loaded chains")
	assert.Contains(t, evmChains, evmChain1.Selector)
	assert.NotContains(t, evmChains, evmChain2.Selector)

	// Verify error was logged
	assert.Equal(t, 1, logs.FilterMessage("Failed to load one or more EVM chains").Len(), "should log error for failed chain")
}

func TestLazyBlockChains_SolanaChains(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			switch selector {
			case solanaChain1.Selector:
				return solanaChain1, nil
			case evmChain1.Selector:
				return evmChain1, nil
			default:
				return nil, chain.ErrBlockChainNotFound
			}
		},
	}

	availableChains := map[uint64]string{
		solanaChain1.Selector: chainsel.FamilySolana,
		evmChain1.Selector:    chainsel.FamilyEVM,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilySolana: loader,
		chainsel.FamilyEVM:    loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Get Solana chains should load only Solana chains
	solanaChains := lazyChains.SolanaChains()
	assert.Len(t, solanaChains, 1, "should return 1 Solana chain")
	assert.Contains(t, solanaChains, solanaChain1.Selector)

	// Loader should be called for Solana chain only
	assert.ElementsMatch(t, []uint64{solanaChain1.Selector}, loader.loadCalls)
}

func TestLazyBlockChains_SolanaChains_LoadError(t *testing.T) {
	t.Parallel()

	lggr, logs := logger.TestObserved(t, zapcore.DebugLevel)

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			return nil, assert.AnError
		},
	}

	availableChains := map[uint64]string{
		solanaChain1.Selector: chainsel.FamilySolana,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilySolana: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, lggr)

	// Get Solana chains - should return empty map and log error
	solanaChains := lazyChains.SolanaChains()
	assert.Empty(t, solanaChains, "should return empty map when load fails")

	// Verify error was logged
	assert.Equal(t, 1, logs.FilterMessage("Failed to load one or more Solana chains").Len(), "should log error for failed chain")
}

func TestLazyBlockChains_AptosChains(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			switch selector {
			case aptosChain1.Selector:
				return aptosChain1, nil
			default:
				return nil, chain.ErrBlockChainNotFound
			}
		},
	}

	availableChains := map[uint64]string{
		aptosChain1.Selector: chainsel.FamilyAptos,
		evmChain1.Selector:   chainsel.FamilyEVM,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyAptos: loader,
		chainsel.FamilyEVM:   loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Get Aptos chains should load only Aptos chains
	aptosChains := lazyChains.AptosChains()
	assert.Len(t, aptosChains, 1, "should return 1 Aptos chain")
	assert.Contains(t, aptosChains, aptosChain1.Selector)

	// Loader should be called for Aptos chain only
	assert.ElementsMatch(t, []uint64{aptosChain1.Selector}, loader.loadCalls)
}

func TestLazyBlockChains_AptosChains_LoadError(t *testing.T) {
	t.Parallel()

	lggr, logs := logger.TestObserved(t, zapcore.DebugLevel)

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			return nil, assert.AnError
		},
	}

	availableChains := map[uint64]string{
		aptosChain1.Selector: chainsel.FamilyAptos,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyAptos: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, lggr)

	// Get Aptos chains - should return empty map and log error
	aptosChains := lazyChains.AptosChains()
	assert.Empty(t, aptosChains, "should return empty map when load fails")

	// Verify error was logged
	assert.Equal(t, 1, logs.FilterMessage("Failed to load one or more Aptos chains").Len(), "should log error for failed chain")
}

func TestLazyBlockChains_SuiChains(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			switch selector {
			case suiChain1.Selector:
				return suiChain1, nil
			default:
				return nil, chain.ErrBlockChainNotFound
			}
		},
	}

	availableChains := map[uint64]string{
		suiChain1.Selector: chainsel.FamilySui,
		evmChain1.Selector: chainsel.FamilyEVM,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilySui: loader,
		chainsel.FamilyEVM: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Get Sui chains should load only Sui chains
	suiChains := lazyChains.SuiChains()
	assert.Len(t, suiChains, 1, "should return 1 Sui chain")
	assert.Contains(t, suiChains, suiChain1.Selector)

	// Loader should be called for Sui chain only
	assert.ElementsMatch(t, []uint64{suiChain1.Selector}, loader.loadCalls)
}

func TestLazyBlockChains_SuiChains_LoadError(t *testing.T) {
	t.Parallel()

	lggr, logs := logger.TestObserved(t, zapcore.DebugLevel)

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			return nil, assert.AnError
		},
	}

	availableChains := map[uint64]string{
		suiChain1.Selector: chainsel.FamilySui,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilySui: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, lggr)

	// Get Sui chains - should return empty map and log error
	suiChains := lazyChains.SuiChains()
	assert.Empty(t, suiChains, "should return empty map when load fails")

	// Verify error was logged
	assert.Equal(t, 1, logs.FilterMessage("Failed to load one or more Sui chains").Len(), "should log error for failed chain")
}

func TestLazyBlockChains_TonChains(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			switch selector {
			case tonChain1.Selector:
				return tonChain1, nil
			default:
				return nil, chain.ErrBlockChainNotFound
			}
		},
	}

	availableChains := map[uint64]string{
		tonChain1.Selector: chainsel.FamilyTon,
		evmChain1.Selector: chainsel.FamilyEVM,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyTon: loader,
		chainsel.FamilyEVM: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Get Ton chains should load only Ton chains
	tonChains := lazyChains.TonChains()
	assert.Len(t, tonChains, 1, "should return 1 Ton chain")
	assert.Contains(t, tonChains, tonChain1.Selector)

	// Loader should be called for Ton chain only
	assert.ElementsMatch(t, []uint64{tonChain1.Selector}, loader.loadCalls)
}

func TestLazyBlockChains_TonChains_LoadError(t *testing.T) {
	t.Parallel()

	lggr, logs := logger.TestObserved(t, zapcore.DebugLevel)

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			return nil, assert.AnError
		},
	}

	availableChains := map[uint64]string{
		tonChain1.Selector: chainsel.FamilyTon,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyTon: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, lggr)

	// Get Ton chains - should return empty map and log error
	tonChains := lazyChains.TonChains()
	assert.Empty(t, tonChains, "should return empty map when load fails")

	// Verify error was logged
	assert.Equal(t, 1, logs.FilterMessage("Failed to load one or more Ton chains").Len(), "should log error for failed chain")
}

func TestLazyBlockChains_TronChains(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			switch selector {
			case tronChain1.Selector:
				return tronChain1, nil
			default:
				return nil, chain.ErrBlockChainNotFound
			}
		},
	}

	availableChains := map[uint64]string{
		tronChain1.Selector: chainsel.FamilyTron,
		evmChain1.Selector:  chainsel.FamilyEVM,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyTron: loader,
		chainsel.FamilyEVM:  loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Get Tron chains should load only Tron chains
	tronChains := lazyChains.TronChains()
	assert.Len(t, tronChains, 1, "should return 1 Tron chain")
	assert.Contains(t, tronChains, tronChain1.Selector)

	// Loader should be called for Tron chain only
	assert.ElementsMatch(t, []uint64{tronChain1.Selector}, loader.loadCalls)
}

func TestLazyBlockChains_TronChains_LoadError(t *testing.T) {
	t.Parallel()

	lggr, logs := logger.TestObserved(t, zapcore.DebugLevel)

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			return nil, assert.AnError
		},
	}

	availableChains := map[uint64]string{
		tronChain1.Selector: chainsel.FamilyTron,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyTron: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, lggr)

	// Get Tron chains - should return empty map and log error
	tronChains := lazyChains.TronChains()
	assert.Empty(t, tronChains, "should return empty map when load fails")

	// Verify error was logged
	assert.Equal(t, 1, logs.FilterMessage("Failed to load one or more Tron chains").Len(), "should log error for failed chain")
}

func TestLazyBlockChains_All_LoadError(t *testing.T) {
	t.Parallel()

	lggr, logs := logger.TestObserved(t, zapcore.DebugLevel)

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			if selector == evmChain1.Selector {
				return evmChain1, nil
			}
			// Fail to load solana chain
			return nil, assert.AnError
		},
	}

	availableChains := map[uint64]string{
		evmChain1.Selector:    chainsel.FamilyEVM,
		solanaChain1.Selector: chainsel.FamilySolana,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyEVM:    loader,
		chainsel.FamilySolana: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, lggr)

	// Iterate through all chains - should skip the failed one
	count := 0
	for selector, c := range lazyChains.All() {
		count++
		assert.NotNil(t, c)
		assert.Equal(t, evmChain1.Selector, selector)
	}

	assert.Equal(t, 1, count, "should iterate over only successfully loaded chains")

	// Verify error was logged
	assert.Equal(t, 1, logs.FilterMessage("Failed to load chain during iteration").Len(), "should log error for failed chain")
}

func TestLazyBlockChains_TryEVMChains_Success(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			switch selector {
			case evmChain1.Selector:
				return evmChain1, nil
			case evmChain2.Selector:
				return evmChain2, nil
			default:
				return nil, chain.ErrBlockChainNotFound
			}
		},
	}

	availableChains := map[uint64]string{
		evmChain1.Selector:    chainsel.FamilyEVM,
		evmChain2.Selector:    chainsel.FamilyEVM,
		solanaChain1.Selector: chainsel.FamilySolana,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyEVM:    loader,
		chainsel.FamilySolana: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Try to get EVM chains - should succeed with no error
	evmChains, err := lazyChains.TryEVMChains()
	require.NoError(t, err)
	assert.Len(t, evmChains, 2, "should return 2 EVM chains")
	assert.Contains(t, evmChains, evmChain1.Selector)
	assert.Contains(t, evmChains, evmChain2.Selector)
}

func TestLazyBlockChains_TryEVMChains_PartialFailure(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			if selector == evmChain1.Selector {
				return evmChain1, nil
			}
			// Fail to load evmChain2
			return nil, assert.AnError
		},
	}

	availableChains := map[uint64]string{
		evmChain1.Selector: chainsel.FamilyEVM,
		evmChain2.Selector: chainsel.FamilyEVM,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyEVM: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Try to get EVM chains - should return error but also successful chains
	evmChains, err := lazyChains.TryEVMChains()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load evm chain")
	assert.Contains(t, err.Error(), strconv.FormatUint(evmChain2.Selector, 10))

	// Should still get the successfully loaded chain
	assert.Len(t, evmChains, 1, "should return successfully loaded chains")
	assert.Contains(t, evmChains, evmChain1.Selector)
}

func TestLazyBlockChains_TryEVMChains_AllFail(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			return nil, assert.AnError
		},
	}

	availableChains := map[uint64]string{
		evmChain1.Selector: chainsel.FamilyEVM,
		evmChain2.Selector: chainsel.FamilyEVM,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyEVM: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Try to get EVM chains - should return error with empty map
	evmChains, err := lazyChains.TryEVMChains()
	require.Error(t, err)

	// Error should contain both chain selectors
	assert.Contains(t, err.Error(), strconv.FormatUint(evmChain1.Selector, 10))
	assert.Contains(t, err.Error(), strconv.FormatUint(evmChain2.Selector, 10))

	assert.Empty(t, evmChains, "should return empty map when all fail")
}

func TestLazyBlockChains_TrySolanaChains_Success(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			if selector == solanaChain1.Selector {
				return solanaChain1, nil
			}

			return nil, chain.ErrBlockChainNotFound
		},
	}

	availableChains := map[uint64]string{
		solanaChain1.Selector: chainsel.FamilySolana,
		evmChain1.Selector:    chainsel.FamilyEVM,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilySolana: loader,
		chainsel.FamilyEVM:    loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Try to get Solana chains - should succeed
	solanaChains, err := lazyChains.TrySolanaChains()
	require.NoError(t, err)
	assert.Len(t, solanaChains, 1)
	assert.Contains(t, solanaChains, solanaChain1.Selector)
}

func TestLazyBlockChains_TrySolanaChains_Failure(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			return nil, assert.AnError
		},
	}

	availableChains := map[uint64]string{
		solanaChain1.Selector: chainsel.FamilySolana,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilySolana: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Try to get Solana chains - should return error
	solanaChains, err := lazyChains.TrySolanaChains()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load solana chain")
	assert.Empty(t, solanaChains)
}

func TestLazyBlockChains_TryAptosChains_Success(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			if selector == aptosChain1.Selector {
				return aptosChain1, nil
			}

			return nil, chain.ErrBlockChainNotFound
		},
	}

	availableChains := map[uint64]string{
		aptosChain1.Selector: chainsel.FamilyAptos,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyAptos: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Try to get Aptos chains - should succeed
	aptosChains, err := lazyChains.TryAptosChains()
	require.NoError(t, err)
	assert.Len(t, aptosChains, 1)
	assert.Contains(t, aptosChains, aptosChain1.Selector)
}

func TestLazyBlockChains_TryAptosChains_Failure(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			return nil, assert.AnError
		},
	}

	availableChains := map[uint64]string{
		aptosChain1.Selector: chainsel.FamilyAptos,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyAptos: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Try to get Aptos chains - should return error
	aptosChains, err := lazyChains.TryAptosChains()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load aptos chain")
	assert.Empty(t, aptosChains)
}

func TestLazyBlockChains_TrySuiChains_Success(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			if selector == suiChain1.Selector {
				return suiChain1, nil
			}

			return nil, chain.ErrBlockChainNotFound
		},
	}

	availableChains := map[uint64]string{
		suiChain1.Selector: chainsel.FamilySui,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilySui: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Try to get Sui chains - should succeed
	suiChains, err := lazyChains.TrySuiChains()
	require.NoError(t, err)
	assert.Len(t, suiChains, 1)
	assert.Contains(t, suiChains, suiChain1.Selector)
}

func TestLazyBlockChains_TrySuiChains_Failure(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			return nil, assert.AnError
		},
	}

	availableChains := map[uint64]string{
		suiChain1.Selector: chainsel.FamilySui,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilySui: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Try to get Sui chains - should return error
	suiChains, err := lazyChains.TrySuiChains()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load sui chain")
	assert.Empty(t, suiChains)
}

func TestLazyBlockChains_TryTonChains_Success(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			if selector == tonChain1.Selector {
				return tonChain1, nil
			}

			return nil, chain.ErrBlockChainNotFound
		},
	}

	availableChains := map[uint64]string{
		tonChain1.Selector: chainsel.FamilyTon,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyTon: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Try to get Ton chains - should succeed
	tonChains, err := lazyChains.TryTonChains()
	require.NoError(t, err)
	assert.Len(t, tonChains, 1)
	assert.Contains(t, tonChains, tonChain1.Selector)
}

func TestLazyBlockChains_TryTonChains_Failure(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			return nil, assert.AnError
		},
	}

	availableChains := map[uint64]string{
		tonChain1.Selector: chainsel.FamilyTon,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyTon: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Try to get Ton chains - should return error
	tonChains, err := lazyChains.TryTonChains()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load ton chain")
	assert.Empty(t, tonChains)
}

func TestLazyBlockChains_TryTronChains_Success(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			if selector == tronChain1.Selector {
				return tronChain1, nil
			}

			return nil, chain.ErrBlockChainNotFound
		},
	}

	availableChains := map[uint64]string{
		tronChain1.Selector: chainsel.FamilyTron,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyTron: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Try to get Tron chains - should succeed
	tronChains, err := lazyChains.TryTronChains()
	require.NoError(t, err)
	assert.Len(t, tronChains, 1)
	assert.Contains(t, tronChains, tronChain1.Selector)
}

func TestLazyBlockChains_TryTronChains_Failure(t *testing.T) {
	t.Parallel()

	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			return nil, assert.AnError
		},
	}

	availableChains := map[uint64]string{
		tronChain1.Selector: chainsel.FamilyTron,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyTron: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Try to get Tron chains - should return error
	tronChains, err := lazyChains.TryTronChains()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load tron chain")
	assert.Empty(t, tronChains)
}

// TestLazyBlockChains_TryEVMChains_WithPointers tests that the generic tryChains
// function correctly handles both value and pointer return types from loaders.
func TestLazyBlockChains_TryEVMChains_WithPointers(t *testing.T) {
	t.Parallel()

	// Test with loader that returns pointers
	loader := &mockChainLoader{
		loadFunc: func(selector uint64) (chain.BlockChain, error) {
			switch selector {
			case evmChain1.Selector:
				// Return pointer to test the PT case in tryChains type switch
				chainCopy := evmChain1
				return &chainCopy, nil
			case evmChain2.Selector:
				// Return value to test the T case in tryChains type switch
				return evmChain2, nil
			default:
				return nil, chain.ErrBlockChainNotFound
			}
		},
	}

	availableChains := map[uint64]string{
		evmChain1.Selector: chainsel.FamilyEVM,
		evmChain2.Selector: chainsel.FamilyEVM,
	}
	loaders := map[string]chain.ChainLoader{
		chainsel.FamilyEVM: loader,
	}

	lazyChains := chain.NewLazyBlockChains(t.Context(), availableChains, loaders, logger.Nop())

	// Try to get EVM chains - should handle both pointers and values
	evmChains, err := lazyChains.TryEVMChains()
	require.NoError(t, err)
	assert.Len(t, evmChains, 2, "should return 2 EVM chains regardless of pointer/value")
	assert.Contains(t, evmChains, evmChain1.Selector)
	assert.Contains(t, evmChains, evmChain2.Selector)

	// Verify the chains are properly dereferenced
	assert.Equal(t, evmChain1.Selector, evmChains[evmChain1.Selector].Selector)
	assert.Equal(t, evmChain2.Selector, evmChains[evmChain2.Selector].Selector)
}
