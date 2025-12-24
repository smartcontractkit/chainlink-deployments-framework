package chain

import (
	"context"
	"iter"
	"slices"
	"sync"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/sui"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/ton"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// ChainLoader is an interface for loading a blockchain instance lazily.
type ChainLoader interface {
	Load(ctx context.Context, selector uint64) (BlockChain, error)
}

// LazyBlockChains is a thread-safe wrapper around BlockChains that loads chains on-demand.
// It maintains a cache of loaded chains and uses ChainLoaders to initialize chains when first accessed.
type LazyBlockChains struct {
	mu              sync.RWMutex
	loadedChains    map[uint64]BlockChain
	loaders         map[string]ChainLoader // keyed by chain family
	availableChains map[uint64]string      // maps selector to chain family
	ctx             context.Context        //nolint:containedctx // Context is needed for lazy loading operations
	lggr            logger.Logger
}

// NewLazyBlockChains creates a new LazyBlockChains instance.
// availableChains maps chain selectors to their family (e.g., "evm", "solana", "aptos").
// loaders provides the ChainLoader for each family.
//
// Chains are loaded on-demand when first accessed. If a chain fails to load during access
// (via GetBySelector, EVMChains, SolanaChains, etc.), the error is logged using lggr and
// the failing chain is skipped. This ensures graceful degradation - successfully loaded
// chains remain accessible while failures are visible in logs.
func NewLazyBlockChains(
	ctx context.Context,
	availableChains map[uint64]string,
	loaders map[string]ChainLoader,
	lggr logger.Logger,
) *LazyBlockChains {
	return &LazyBlockChains{
		loadedChains:    make(map[uint64]BlockChain),
		loaders:         loaders,
		availableChains: availableChains,
		ctx:             ctx,
		lggr:            lggr,
	}
}

// GetBySelector returns a blockchain by its selector, loading it lazily if not already loaded.
func (l *LazyBlockChains) GetBySelector(selector uint64) (BlockChain, error) {
	// Fast path: check if already loaded
	l.mu.RLock()
	if chain, ok := l.loadedChains[selector]; ok {
		l.mu.RUnlock()
		return chain, nil
	}
	l.mu.RUnlock()

	// Slow path: need to load the chain
	l.mu.Lock()
	defer l.mu.Unlock()

	// Double-check after acquiring write lock
	if chain, ok := l.loadedChains[selector]; ok {
		return chain, nil
	}

	// Check if the chain is available
	family, ok := l.availableChains[selector]
	if !ok {
		return nil, ErrBlockChainNotFound
	}

	// Get the loader for this family
	loader, ok := l.loaders[family]
	if !ok {
		return nil, ErrBlockChainNotFound
	}

	// Load the chain
	chain, err := loader.Load(l.ctx, selector)
	if err != nil {
		return nil, err
	}

	// Cache the loaded chain
	l.loadedChains[selector] = chain

	return chain, nil
}

// Exists checks if a chain with the given selector is available (not necessarily loaded).
func (l *LazyBlockChains) Exists(selector uint64) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, ok := l.availableChains[selector]

	return ok
}

// ExistsN checks if all chains with the given selectors are available.
func (l *LazyBlockChains) ExistsN(selectors ...uint64) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, selector := range selectors {
		if _, ok := l.availableChains[selector]; !ok {
			return false
		}
	}

	return true
}

// All returns an iterator over all chains, loading them lazily as they are accessed.
// If a chain fails to load, the error is logged and the chain is skipped.
func (l *LazyBlockChains) All() iter.Seq2[uint64, BlockChain] {
	return func(yield func(uint64, BlockChain) bool) {
		l.mu.RLock()
		selectors := make([]uint64, 0, len(l.availableChains))
		for selector := range l.availableChains {
			selectors = append(selectors, selector)
		}
		l.mu.RUnlock()

		// Sort for consistent iteration order
		slices.Sort(selectors)

		for _, selector := range selectors {
			chain, err := l.GetBySelector(selector)
			if err != nil {
				l.lggr.Errorw("Failed to load chain during iteration",
					"selector", selector,
					"error", err,
				)
				// Skip chains that fail to load
				continue
			}
			if !yield(selector, chain) {
				return
			}
		}
	}
}

// EVMChains returns a map of all EVM chains, loading them lazily.
// If a chain fails to load, the error is logged and the chain is skipped.
func (l *LazyBlockChains) EVMChains() map[uint64]evm.Chain {
	l.mu.RLock()
	selectors := make([]uint64, 0)
	for selector, f := range l.availableChains {
		if f == chainsel.FamilyEVM {
			selectors = append(selectors, selector)
		}
	}
	l.mu.RUnlock()

	chains := make(map[uint64]evm.Chain)
	for _, selector := range selectors {
		chain, err := l.GetBySelector(selector)
		if err != nil {
			l.lggr.Errorw("Failed to load EVM chain",
				"selector", selector,
				"error", err,
			)

			continue
		}
		switch c := chain.(type) {
		case evm.Chain:
			chains[selector] = c
		case *evm.Chain:
			if c != nil {
				chains[selector] = *c
			}
		}
	}

	return chains
}

// SolanaChains returns a map of all Solana chains, loading them lazily.
// If a chain fails to load, the error is logged and the chain is skipped.
func (l *LazyBlockChains) SolanaChains() map[uint64]solana.Chain {
	l.mu.RLock()
	selectors := make([]uint64, 0)
	for selector, f := range l.availableChains {
		if f == chainsel.FamilySolana {
			selectors = append(selectors, selector)
		}
	}
	l.mu.RUnlock()

	chains := make(map[uint64]solana.Chain)
	for _, selector := range selectors {
		chain, err := l.GetBySelector(selector)
		if err != nil {
			l.lggr.Errorw("Failed to load Solana chain",
				"selector", selector,
				"error", err,
			)

			continue
		}
		switch c := chain.(type) {
		case solana.Chain:
			chains[selector] = c
		case *solana.Chain:
			if c != nil {
				chains[selector] = *c
			}
		}
	}

	return chains
}

// AptosChains returns a map of all Aptos chains, loading them lazily.
// If a chain fails to load, the error is logged and the chain is skipped.
func (l *LazyBlockChains) AptosChains() map[uint64]aptos.Chain {
	l.mu.RLock()
	selectors := make([]uint64, 0)
	for selector, f := range l.availableChains {
		if f == chainsel.FamilyAptos {
			selectors = append(selectors, selector)
		}
	}
	l.mu.RUnlock()

	chains := make(map[uint64]aptos.Chain)
	for _, selector := range selectors {
		chain, err := l.GetBySelector(selector)
		if err != nil {
			l.lggr.Errorw("Failed to load Aptos chain",
				"selector", selector,
				"error", err,
			)

			continue
		}
		switch c := chain.(type) {
		case aptos.Chain:
			chains[selector] = c
		case *aptos.Chain:
			if c != nil {
				chains[selector] = *c
			}
		}
	}

	return chains
}

// SuiChains returns a map of all Sui chains, loading them lazily.
// If a chain fails to load, the error is logged and the chain is skipped.
func (l *LazyBlockChains) SuiChains() map[uint64]sui.Chain {
	l.mu.RLock()
	selectors := make([]uint64, 0)
	for selector, f := range l.availableChains {
		if f == chainsel.FamilySui {
			selectors = append(selectors, selector)
		}
	}
	l.mu.RUnlock()

	chains := make(map[uint64]sui.Chain)
	for _, selector := range selectors {
		chain, err := l.GetBySelector(selector)
		if err != nil {
			l.lggr.Errorw("Failed to load Sui chain",
				"selector", selector,
				"error", err,
			)

			continue
		}
		switch c := chain.(type) {
		case sui.Chain:
			chains[selector] = c
		case *sui.Chain:
			if c != nil {
				chains[selector] = *c
			}
		}
	}

	return chains
}

// TonChains returns a map of all Ton chains, loading them lazily.
// If a chain fails to load, the error is logged and the chain is skipped.
func (l *LazyBlockChains) TonChains() map[uint64]ton.Chain {
	l.mu.RLock()
	selectors := make([]uint64, 0)
	for selector, f := range l.availableChains {
		if f == chainsel.FamilyTon {
			selectors = append(selectors, selector)
		}
	}
	l.mu.RUnlock()

	chains := make(map[uint64]ton.Chain)
	for _, selector := range selectors {
		chain, err := l.GetBySelector(selector)
		if err != nil {
			l.lggr.Errorw("Failed to load Ton chain",
				"selector", selector,
				"error", err,
			)

			continue
		}
		switch c := chain.(type) {
		case ton.Chain:
			chains[selector] = c
		case *ton.Chain:
			if c != nil {
				chains[selector] = *c
			}
		}
	}

	return chains
}

// TronChains returns a map of all Tron chains, loading them lazily.
// If a chain fails to load, the error is logged and the chain is skipped.
func (l *LazyBlockChains) TronChains() map[uint64]tron.Chain {
	l.mu.RLock()
	selectors := make([]uint64, 0)
	for selector, f := range l.availableChains {
		if f == chainsel.FamilyTron {
			selectors = append(selectors, selector)
		}
	}
	l.mu.RUnlock()

	chains := make(map[uint64]tron.Chain)
	for _, selector := range selectors {
		chain, err := l.GetBySelector(selector)
		if err != nil {
			l.lggr.Errorw("Failed to load Tron chain",
				"selector", selector,
				"error", err,
			)

			continue
		}
		switch c := chain.(type) {
		case tron.Chain:
			chains[selector] = c
		case *tron.Chain:
			if c != nil {
				chains[selector] = *c
			}
		}
	}

	return chains
}

// ListChainSelectors returns all available chain selectors with optional filtering.
func (l *LazyBlockChains) ListChainSelectors(options ...ChainSelectorsOption) []uint64 {
	opts := chainSelectorsOptions{}
	for _, option := range options {
		option(&opts)
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	selectors := make([]uint64, 0, len(l.availableChains))
	for selector, family := range l.availableChains {
		if opts.excludedChainSels != nil {
			if _, excluded := opts.excludedChainSels[selector]; excluded {
				continue
			}
		}
		if opts.includedFamilies != nil {
			if _, ok := opts.includedFamilies[family]; !ok {
				continue
			}
		}
		selectors = append(selectors, selector)
	}

	slices.Sort(selectors)

	return selectors
}

// ToBlockChains converts the LazyBlockChains to a regular BlockChains instance.
// This loads all available chains eagerly.
func (l *LazyBlockChains) ToBlockChains() (BlockChains, error) {
	l.mu.RLock()
	selectors := make([]uint64, 0, len(l.availableChains))
	for selector := range l.availableChains {
		selectors = append(selectors, selector)
	}
	l.mu.RUnlock()

	chains := make(map[uint64]BlockChain)
	for _, selector := range selectors {
		chain, err := l.GetBySelector(selector)
		if err != nil {
			return BlockChains{}, err
		}
		chains[selector] = chain
	}

	return NewBlockChains(chains), nil
}
