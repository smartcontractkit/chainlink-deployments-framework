package chain

import (
	"iter"
	"maps"
	"reflect"
	"slices"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/sui"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/ton"
)

var _ BlockChain = evm.Chain{}
var _ BlockChain = solana.Chain{}
var _ BlockChain = aptos.Chain{}
var _ BlockChain = sui.Chain{}
var _ BlockChain = ton.Chain{}

// BlockChain is an interface that represents a chain.
// A chain can be an EVM chain, Solana chain Aptos chain or others.
type BlockChain interface {
	// String returns chain name and selector "<name> (<selector>)"
	String() string
	// Name returns the name of the chain
	Name() string
	ChainSelector() uint64
	Family() string
}

// BlockChains represents a collection of chains.
// It provides querying capabilities for different types of chains.
type BlockChains struct {
	chains map[uint64]BlockChain
}

// NewBlockChains initializes a new BlockChains instance
func NewBlockChains(chains map[uint64]BlockChain) BlockChains {
	// perform a copy of chains
	// to avoid mutating the original map
	if chains == nil {
		chains = make(map[uint64]BlockChain)
	} else {
		newChains := make(map[uint64]BlockChain, len(chains))
		for k, v := range chains {
			newChains[k] = v
		}
		chains = newChains
	}

	return BlockChains{
		chains: chains,
	}
}

// NewBlockChainsFromSlice initializes a new BlockChains instance from a slice of BlockChain.
func NewBlockChainsFromSlice(chains []BlockChain) BlockChains {
	// Create a new map to hold the chains
	chainsMap := make(map[uint64]BlockChain, len(chains))

	// Populate the map with chains
	for _, chain := range chains {
		chainsMap[chain.ChainSelector()] = chain
	}

	return NewBlockChains(chainsMap)
}

// Exists checks if a chain with the given selector exists.
func (b BlockChains) Exists(selector uint64) bool {
	_, ok := b.chains[selector]

	return ok
}

// ExistsN checks if all chains with the given selectors exist.
func (b BlockChains) ExistsN(selectors ...uint64) bool {
	for _, selector := range selectors {
		if !b.Exists(selector) {
			return false
		}
	}

	return true
}

// All returns an iterator over all chains with their selectors.
func (b BlockChains) All() iter.Seq2[uint64, BlockChain] {
	return maps.All(b.chains)
}

// EVMChains returns a map of all EVM chains with their selectors.
func (b BlockChains) EVMChains() map[uint64]evm.Chain {
	return getChainsByType[evm.Chain, *evm.Chain](b)
}

// SolanaChains returns a map of all Solana chains with their selectors.
func (b BlockChains) SolanaChains() map[uint64]solana.Chain {
	return getChainsByType[solana.Chain, *solana.Chain](b)
}

// AptosChains returns a map of all Aptos chains with their selectors.
func (b BlockChains) AptosChains() map[uint64]aptos.Chain {
	return getChainsByType[aptos.Chain, *aptos.Chain](b)
}

// SuiChains returns a map of all Sui chains with their selectors.
func (b BlockChains) SuiChains() map[uint64]sui.Chain {
	return getChainsByType[sui.Chain, *sui.Chain](b)
}

// TonChains returns a map of all Ton chains with their selectors.
func (b BlockChains) TonChains() map[uint64]ton.Chain {
	return getChainsByType[ton.Chain, *ton.Chain](b)
}

// ChainSelectorsOption defines a function type for configuring ChainSelectors
type ChainSelectorsOption func(*chainSelectorsOptions)

type chainSelectorsOptions struct {
	// Use map for faster lookups
	includedFamilies  map[string]struct{}
	excludedChainSels map[uint64]struct{}
}

// WithFamily returns an option to filter chains by family (evm, solana, aptos)
// Use constants from chain_selectors package eg WithFamily(chain_selectors.FamilySolana)
// This can be used more than once to include multiple families.
func WithFamily(family string) ChainSelectorsOption {
	return func(o *chainSelectorsOptions) {
		if o.includedFamilies == nil {
			o.includedFamilies = make(map[string]struct{})
		}
		o.includedFamilies[family] = struct{}{}
	}
}

// WithChainSelectorsExclusion returns an option to exclude specific chain selectors
func WithChainSelectorsExclusion(chainSelectors []uint64) ChainSelectorsOption {
	return func(o *chainSelectorsOptions) {
		if o.excludedChainSels == nil {
			o.excludedChainSels = make(map[uint64]struct{})
		}
		for _, selector := range chainSelectors {
			o.excludedChainSels[selector] = struct{}{}
		}
	}
}

// ListChainSelectors returns all chain selectors with optional filtering
// Options:
// - WithFamily: filter by family eg WithFamily(chain_selectors.FamilySolana)
// - WithChainSelectorsExclusion: exclude specific chain selectors
func (b BlockChains) ListChainSelectors(options ...ChainSelectorsOption) []uint64 {
	opts := chainSelectorsOptions{}

	// Apply all provided options
	for _, option := range options {
		option(&opts)
	}

	selectors := make([]uint64, 0, len(b.chains))

	for selector, chain := range b.chains {
		if opts.excludedChainSels != nil {
			if _, excluded := opts.excludedChainSels[selector]; excluded {
				continue
			}
		}
		if opts.includedFamilies != nil {
			if _, ok := opts.includedFamilies[chain.Family()]; !ok {
				continue
			}
		}

		selectors = append(selectors, selector)
	}

	// Sort for consistent output
	slices.Sort(selectors)

	return selectors
}

// getChainsByType is a helper function to extract chains of a specific type from BlockChains.
// It accepts two type parameters: VT for the target type and PT for pointer types of the same chain type.
// eg getChainsByType[evm.Chain, *evm.Chain](b BlockChains) returns a map of uint64 to evm.Chain.
// It handles both value and pointer types, allowing for flexibility in how chains are stored.
func getChainsByType[VT any, PT any](b BlockChains) map[uint64]VT {
	chains := make(map[uint64]VT, len(b.chains))
	for _, chain := range b.chains {
		switch c := any(chain).(type) {
		case VT:
			chains[chain.ChainSelector()] = c
		case PT:
			val := reflect.ValueOf(c)
			if val.Kind() == reflect.Ptr && !val.IsNil() {
				elem := val.Elem()
				if elem.CanInterface() {
					if v, ok := elem.Interface().(VT); ok {
						chains[chain.ChainSelector()] = v
					}
				}
			}
		}
	}

	return chains
}
