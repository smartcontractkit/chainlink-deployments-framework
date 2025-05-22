package chain_test

import (
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
)

var evmChain1 = evm.Chain{Selector: 1}
var evmChain2 = evm.Chain{Selector: 2}
var solanaChain1 = solana.Chain{Selector: 3}
var aptosChain1 = aptos.Chain{Selector: 4}

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
			name: "with exclusion",
			options: []chain.ChainSelectorsOption{chain.WithChainSelectorsExclusion(
				[]uint64{evmChain1.Selector, aptosChain1.Selector}),
			},
			expectedIDs: []uint64{evmChain2.Selector, solanaChain1.Selector},
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
			assert.Equal(t, tc.expectedIDs, selectors, tc.description)
		})
	}
}

// buildBlockChains creates a new BlockChains instance with the test chains.
// 2 evm chains, 1 solana chain and 1 aptos chain.
func buildBlockChains() chain.BlockChains {
	chains := chain.NewBlockChains(map[uint64]chain.BlockChain{
		evmChain1.ChainSelector():    evmChain1,
		solanaChain1.ChainSelector(): solanaChain1,
		evmChain2.ChainSelector():    evmChain2,
		aptosChain1.ChainSelector():  aptosChain1,
	})

	return chains
}
