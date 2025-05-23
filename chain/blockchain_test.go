package chain_test

import (
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
var suiChain1 = sui.Chain{Selector: chain_selectors.SUI_LOCALNET.Selector}
var tonChain1 = ton.Chain{Selector: chain_selectors.TON_LOCALNET.Selector}

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

func TestBlockChainsEVMChains(t *testing.T) {
	t.Parallel()

	chains := buildBlockChains()

	evmChains, err := chains.EVMChains()
	require.NoError(t, err)

	assert.Len(t, evmChains, 2, "expected 2 EVM chains")

	_, exists := evmChains[evmChain1.Selector]
	assert.True(t, exists, "expected EVM chain with selector 1")

	_, exists = evmChains[evmChain2.Selector]
	assert.True(t, exists, "expected EVM chain with selector 3")
}

func TestBlockChainsSolanaChains(t *testing.T) {
	t.Parallel()

	chains := buildBlockChains()

	solanaChains, err := chains.SolanaChains()
	require.NoError(t, err)

	assert.Len(t, solanaChains, 1, "expected 1 Solana chain")

	_, exists := solanaChains[solanaChain1.Selector]
	assert.True(t, exists, "expected Solana chain with selector 2")
}

func TestBlockChainsAptosChains(t *testing.T) {
	t.Parallel()

	chains := buildBlockChains()

	aptosChains, err := chains.AptosChains()
	require.NoError(t, err)

	assert.Len(t, aptosChains, 1, "expected 1 Aptos chain")

	_, exists := aptosChains[aptosChain1.Selector]
	assert.True(t, exists, "expected Aptos chain with selector 4")
}

func TestBlockChainsSuiChains(t *testing.T) {
	t.Parallel()

	chains := buildBlockChains()

	suiChains, err := chains.SuiChains()
	require.NoError(t, err)

	assert.Len(t, suiChains, 1, "expected 1 Sui chain")

	_, exists := suiChains[suiChain1.Selector]
	assert.True(t, exists, "expected Sui chain with selector 5")
}

func TestBlockChainsTonChains(t *testing.T) {
	t.Parallel()

	chains := buildBlockChains()

	tonChains, err := chains.TonChains()
	require.NoError(t, err)

	assert.Len(t, tonChains, 1, "expected 1 Ton chain")

	_, exists := tonChains[tonChain1.Selector]
	assert.True(t, exists, "expected Ton chain with selector 6")
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
// 2 evm chains, 1 solana chain, 1 aptos chain, 1 sui chain
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
