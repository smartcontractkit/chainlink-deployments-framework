package chain_test

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/sui"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/ton"
)

var evmChain1 = evm.Chain{Selector: chain_selectors.TEST_90000001.Selector}
var evmChain2 = evm.Chain{Selector: chain_selectors.TEST_90000002.Selector}
var solanaChain1 = solana.Chain{Selector: chain_selectors.TEST_22222222222222222222222222222222222222222222.Selector}
var aptosChain1 = aptos.Chain{Selector: chain_selectors.APTOS_LOCALNET.Selector}
var suiChain1 = sui.Chain{ChainMetadata: sui.ChainMetadata{Selector: chain_selectors.SUI_LOCALNET.Selector}}
var tonChain1 = ton.Chain{ChainMetadata: ton.ChainMetadata{Selector: chain_selectors.TON_LOCALNET.Selector}}

func TestNewBlockChains(t *testing.T) {
	t.Parallel()

	t.Run("nil map", func(t *testing.T) {
		t.Parallel()

		chains := chain.NewBlockChains(nil)

		require.NotNil(t, chains)
	})

	t.Run("populated map", func(t *testing.T) {
		t.Parallel()

		original := map[uint64]chain.BlockChain{
			evmChain1.Selector:    evmChain1,
			solanaChain1.Selector: solanaChain1,
		}
		chains := chain.NewBlockChains(original)

		require.NotNil(t, chains)
	})
}

func TestNewBlockChainsFromSlice(t *testing.T) {
	t.Parallel()

	t.Run("empty slice", func(t *testing.T) {
		t.Parallel()

		chains := chain.NewBlockChainsFromSlice(nil)

		require.NotNil(t, chains)
		assert.Empty(t, maps.Collect(chains.All()), "expected empty chains map")
	})

	t.Run("populated slice", func(t *testing.T) {
		t.Parallel()

		chains := chain.NewBlockChainsFromSlice([]chain.BlockChain{evmChain1, solanaChain1})

		require.NotNil(t, chains)
		assert.Len(t, maps.Collect(chains.All()), 2, "expected 2 chains")
	})
}

func Test_BlockChains_Exists(t *testing.T) {
	t.Parallel()

	chains := buildBlockChains()

	tests := []struct {
		name     string
		selector uint64
		expected bool
	}{
		{
			name:     "exists",
			selector: evmChain1.Selector,
			expected: true,
		},
		{
			name:     "does not exist",
			selector: 99999999,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			exists := chains.Exists(tt.selector)
			assert.Equal(t, tt.expected, exists)
		})
	}
}

func Test_BlockChains_ExistsN(t *testing.T) {
	t.Parallel()

	chains := buildBlockChains()

	tests := []struct {
		name      string
		selectors []uint64
		expected  bool
	}{
		{
			name:      "all exist",
			selectors: []uint64{evmChain1.Selector, solanaChain1.Selector},
			expected:  true,
		},
		{
			name:      "some do not exist",
			selectors: []uint64{evmChain1.Selector, 99999999},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			exists := chains.ExistsN(tt.selectors...)
			assert.Equal(t, tt.expected, exists)
		})
	}
}

func TestBlockChainsAllChains(t *testing.T) {
	t.Parallel()

	chains := buildBlockChains()

	allChains := maps.Collect(chains.All())

	expectedSelectors := []uint64{
		evmChain1.Selector, evmChain2.Selector,
		solanaChain1.Selector, aptosChain1.Selector,
		suiChain1.Selector, tonChain1.Selector,
	}

	assert.Len(t, allChains, len(expectedSelectors))

	for _, selector := range expectedSelectors {
		_, exists := allChains[selector]
		assert.True(t, exists, "expected chain with selector %d", selector)
	}
}

func TestBlockChainsGetters(t *testing.T) {
	t.Parallel()

	valueChains := buildBlockChains()
	pointerChains := buildBlockChainsPointers()

	tests := []struct {
		name        string
		runTest     func(t *testing.T, chains chain.BlockChains)
		description string
	}{
		{
			name: "EVMChains",
			runTest: func(t *testing.T, chains chain.BlockChains) {
				t.Helper()
				evmChains := chains.EVMChains()
				expectedSelectors := []uint64{evmChain1.Selector, evmChain2.Selector}

				assert.Len(t, evmChains, len(expectedSelectors), "unexpected number of EVM chains")

				for _, selector := range expectedSelectors {
					_, exists := evmChains[selector]
					assert.True(t, exists, "expected EVM chain with selector %d", selector)
				}
			},
			description: "should return all EVM chains",
		},
		{
			name: "SolanaChains",
			runTest: func(t *testing.T, chains chain.BlockChains) {
				t.Helper()
				solanaChains := chains.SolanaChains()
				expectedSelectors := []uint64{solanaChain1.Selector}

				assert.Len(t, solanaChains, len(expectedSelectors), "unexpected number of Solana chains")

				for _, selector := range expectedSelectors {
					_, exists := solanaChains[selector]
					assert.True(t, exists, "expected Solana chain with selector %d", selector)
				}
			},
			description: "should return all Solana chains",
		},
		{
			name: "AptosChains",
			runTest: func(t *testing.T, chains chain.BlockChains) {
				t.Helper()
				aptosChains := chains.AptosChains()
				expectedSelectors := []uint64{aptosChain1.Selector}

				assert.Len(t, aptosChains, len(expectedSelectors), "unexpected number of Aptos chains")

				for _, selector := range expectedSelectors {
					_, exists := aptosChains[selector]
					assert.True(t, exists, "expected Aptos chain with selector %d", selector)
				}
			},
			description: "should return all Aptos chains",
		},
		{
			name: "SuiChains",
			runTest: func(t *testing.T, chains chain.BlockChains) {
				t.Helper()
				suiChains := chains.SuiChains()
				expectedSelectors := []uint64{suiChain1.Selector}

				assert.Len(t, suiChains, len(expectedSelectors), "unexpected number of Sui chains")

				for _, selector := range expectedSelectors {
					_, exists := suiChains[selector]
					assert.True(t, exists, "expected Sui chain with selector %d", selector)
				}
			},
			description: "should return all Sui chains",
		},
		{
			name: "TonChains",
			runTest: func(t *testing.T, chains chain.BlockChains) {
				t.Helper()
				tonChains := chains.TonChains()
				expectedSelectors := []uint64{tonChain1.Selector}

				assert.Len(t, tonChains, len(expectedSelectors), "unexpected number of Ton chains")

				for _, selector := range expectedSelectors {
					_, exists := tonChains[selector]
					assert.True(t, exists, "expected Ton chain with selector %d", selector)
				}
			},
			description: "should return all Ton chains",
		},
	}

	// Run tests for both value and pointer chains
	chainTypes := []struct {
		name   string
		chains chain.BlockChains
	}{
		{"value chains", valueChains},
		{"pointer chains", pointerChains},
	}

	for _, tc := range tests {
		for _, ct := range chainTypes {
			t.Run(tc.name+"_"+ct.name, func(t *testing.T) {
				t.Parallel()
				tc.runTest(t, ct.chains)
			})
		}
	}
}

func TestBlockChainsListChainSelectors(t *testing.T) {
	t.Parallel()

	chains := buildBlockChains()

	tests := []struct {
		name        string
		options     []chain.ChainSelectorsOption
		expectedIDs []uint64
		description string
	}{
		{
			name:    "no options",
			options: []chain.ChainSelectorsOption{},
			expectedIDs: []uint64{
				evmChain1.ChainSelector(), evmChain2.ChainSelector(),
				solanaChain1.ChainSelector(), aptosChain1.ChainSelector(),
				suiChain1.ChainSelector(), tonChain1.ChainSelector(),
			},
			description: "expected all chain selectors",
		},
		{
			name:        "with family filter - EVM",
			options:     []chain.ChainSelectorsOption{chain.WithFamily(chain_selectors.FamilyEVM)},
			expectedIDs: []uint64{evmChain1.ChainSelector(), evmChain2.ChainSelector()},
			description: "expected EVM chain selectors",
		},
		{
			name:        "with family filter - Solana",
			options:     []chain.ChainSelectorsOption{chain.WithFamily(chain_selectors.FamilySolana)},
			expectedIDs: []uint64{solanaChain1.Selector},
			description: "expected Solana chain selectors",
		},
		{
			name:        "with family filter - Aptos",
			options:     []chain.ChainSelectorsOption{chain.WithFamily(chain_selectors.FamilyAptos)},
			expectedIDs: []uint64{aptosChain1.Selector},
			description: "expected Aptos chain selectors",
		},
		{
			name:        "with family filter - Sui",
			options:     []chain.ChainSelectorsOption{chain.WithFamily(chain_selectors.FamilySui)},
			expectedIDs: []uint64{suiChain1.Selector},
			description: "expected Sui chain selectors",
		},
		{
			name:        "with family filter - Ton",
			options:     []chain.ChainSelectorsOption{chain.WithFamily(chain_selectors.FamilyTon)},
			expectedIDs: []uint64{tonChain1.Selector},
			description: "expected Ton chain selectors",
		},
		{
			name:        "with multiple families",
			options:     []chain.ChainSelectorsOption{chain.WithFamily(chain_selectors.FamilyEVM), chain.WithFamily(chain_selectors.FamilySolana)},
			expectedIDs: []uint64{evmChain1.Selector, evmChain2.Selector, solanaChain1.Selector},
			description: "expected EVM and Solana chain selectors",
		},
		{
			name: "with exclusion",
			options: []chain.ChainSelectorsOption{chain.WithChainSelectorsExclusion(
				[]uint64{evmChain1.Selector, aptosChain1.Selector}),
			},
			expectedIDs: []uint64{evmChain2.Selector, solanaChain1.Selector, suiChain1.Selector, tonChain1.Selector},
			description: "expected chain selectors excluding 1 and 4",
		},
		{
			name: "with family and exclusion",
			options: []chain.ChainSelectorsOption{
				chain.WithFamily(chain_selectors.FamilyEVM),
				chain.WithChainSelectorsExclusion([]uint64{evmChain1.Selector}),
			},
			expectedIDs: []uint64{evmChain2.Selector},
			description: "expected EVM chain selectors excluding 1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			selectors := chains.ListChainSelectors(tc.options...)
			assert.ElementsMatch(t, tc.expectedIDs, selectors, tc.description)
		})
	}
}

// buildBlockChains creates a new BlockChains instance with the test chains.
// 2 evm chains, 1 solana chain, 1 aptos chain, 1 sui chain, 1 ton chain.
func buildBlockChains() chain.BlockChains {
	chains := chain.NewBlockChains(map[uint64]chain.BlockChain{
		evmChain1.ChainSelector():    evmChain1,
		solanaChain1.ChainSelector(): solanaChain1,
		evmChain2.ChainSelector():    evmChain2,
		aptosChain1.ChainSelector():  aptosChain1,
		suiChain1.ChainSelector():    suiChain1,
		tonChain1.ChainSelector():    tonChain1,
	})

	return chains
}

// buildBlockChainsPointers creates a new BlockChains instance with the test chains as pointers.
func buildBlockChainsPointers() chain.BlockChains {
	chains := buildBlockChains()
	pointerChains := make(map[uint64]chain.BlockChain)
	for selector, c := range chains.All() {
		switch c := c.(type) {
		case evm.Chain:
			pointerChains[selector] = &c
		case solana.Chain:
			pointerChains[selector] = &c
		case aptos.Chain:
			pointerChains[selector] = &c
		case sui.Chain:
			pointerChains[selector] = &c
		case ton.Chain:
			pointerChains[selector] = &c
		default:
			continue // skip unsupported chains
		}
	}

	return chain.NewBlockChains(pointerChains)
}
