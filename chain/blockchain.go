package chain

import (
	"errors"
	"sort"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
)

var _ BlockChain = evm.Chain{}
var _ BlockChain = solana.Chain{}
var _ BlockChain = aptos.Chain{}

type BlockChains struct {
	chains map[uint64]BlockChain
}

type BlockChain interface {
	String() string
	Name() string
	ChainSelector() uint64
}

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

// ChainSelectorsOption defines a function type for configuring ChainSelectors
type ChainSelectorsOption func(*chainSelectorsOptions)

// chainSelectorsOptions holds configuration for chain selector filtering
type chainSelectorsOptions struct {
	family    string
	excluding []uint64
}

// WithFamily returns an option to filter chains by family (evm, solana, aptos)
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
		excluded := false
		for _, exclude := range opts.excluding {
			if selector == exclude {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		// Apply family filter if specified
		if opts.family != "" {
			switch chain.(type) {
			case evm.Chain:
				if opts.family != chain_selectors.FamilyEVM {
					continue
				}
			case solana.Chain:
				if opts.family != chain_selectors.FamilySolana {
					continue
				}
			case aptos.Chain:
				if opts.family != chain_selectors.FamilyAptos {
					continue
				}
			default:
				continue
			}
		}

		selectors = append(selectors, selector)
	}

	// Sort for consistent output
	sort.Slice(selectors, func(i, j int) bool {
		return selectors[i] < selectors[j]
	})

	return selectors
}

func (b BlockChains) chainBySelector(selector uint64) (BlockChain, error) {
	c, exists := b.chains[selector]
	if !exists {
		return nil, errors.New("chain does not exist")
	}

	return c, nil
}
