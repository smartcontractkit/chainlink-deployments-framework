package chain

import (
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

// EVMChains returns a map of all EVM chains with their selectors.
func (b BlockChains) EVMChains() (map[uint64]evm.Chain, error) {
	var evmChains = make(map[uint64]evm.Chain)
	for selector, chain := range b.chains {
		c, ok := chain.(evm.Chain)
		if !ok {
			continue
		}
		evmChains[selector] = c
	}

	return evmChains, nil
}

// SolanaChains returns a map of all Solana chains with their selectors.
func (b BlockChains) SolanaChains() (map[uint64]solana.Chain, error) {
	var solanaChains = make(map[uint64]solana.Chain)
	for selector, chain := range b.chains {
		c, ok := chain.(solana.Chain)
		if !ok {
			continue
		}
		solanaChains[selector] = c
	}

	return solanaChains, nil
}

// AptosChains returns a map of all Aptos chains with their selectors.
func (b BlockChains) AptosChains() (map[uint64]aptos.Chain, error) {
	var aptosChains = make(map[uint64]aptos.Chain)
	for selector, chain := range b.chains {
		c, ok := chain.(aptos.Chain)
		if !ok {
			continue
		}
		aptosChains[selector] = c
	}

	return aptosChains, nil
}

// SuiChains returns a map of all Sui chains with their selectors.
func (b BlockChains) SuiChains() (map[uint64]sui.Chain, error) {
	var suiChains = make(map[uint64]sui.Chain)
	for selector, chain := range b.chains {
		c, ok := chain.(sui.Chain)
		if !ok {
			continue
		}
		suiChains[selector] = c
	}

	return suiChains, nil
}

// TonChains returns a map of all Ton chains with their selectors.
func (b BlockChains) TonChains() (map[uint64]ton.Chain, error) {
	var tonChains = make(map[uint64]ton.Chain)
	for selector, chain := range b.chains {
		c, ok := chain.(ton.Chain)
		if !ok {
			continue
		}
		tonChains[selector] = c
	}

	return tonChains, nil
}

// ChainSelectorsOption defines a function type for configuring ChainSelectors
type ChainSelectorsOption func(*chainSelectorsOptions)

type chainSelectorsOptions struct {
	family    string
	excluding []uint64
}

// WithFamily returns an option to filter chains by family (evm, solana, aptos)
// Use constants from chain_selectors package eg WithFamily(chain_selectors.FamilySolana)
func WithFamily(family string) ChainSelectorsOption {
	return func(o *chainSelectorsOptions) {
		o.family = family
	}
}

// WithChainSelectorsExclusion returns an option to exclude specific chain selectors
func WithChainSelectorsExclusion(chainSelectors []uint64) ChainSelectorsOption {
	return func(o *chainSelectorsOptions) {
		o.excluding = chainSelectors
	}
}

// ListChainSelectors returns all chain selectors with optional filtering
// Options:
// - WithFamily: filter by family eg WithFamily(chain_selectors.FamilySolana)
// - WithChainSelectorsExclusion: exclude specific chain selectors
func (b BlockChains) ListChainSelectors(options ...ChainSelectorsOption) []uint64 {
	// Initialize default options
	opts := chainSelectorsOptions{
		family:    "",
		excluding: []uint64{},
	}

	// Apply all provided options
	for _, option := range options {
		option(&opts)
	}

	selectors := make([]uint64, 0, len(b.chains))

	for selector, chain := range b.chains {
		// Skip if in exclusion list
		if slices.Contains(opts.excluding, selector) {
			continue
		}

		// Apply family filter if specified
		if opts.family != "" {
			if opts.family != chain.Family() {
				continue
			}
		}

		selectors = append(selectors, selector)
	}

	// Sort for consistent output
	slices.Sort(selectors)

	return selectors
}
