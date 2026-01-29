package chain

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"
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
	mu                 sync.RWMutex
	loadedChains       map[uint64]BlockChain
	loaders            map[string]ChainLoader // keyed by chain family
	supportedSelectors map[uint64]string      // maps selector to chain family
	ctx                context.Context        //nolint:containedctx // Context is needed for lazy loading operations
	lggr               logger.Logger
}

// NewLazyBlockChains creates a new LazyBlockChains instance.
// supportedSelectors maps chain selectors to their family (e.g., "evm", "solana", "aptos").
// loaders provides the ChainLoader for each family.
//
// Chains are loaded on-demand when first accessed. If a chain fails to load during access
// (via GetBySelector, EVMChains, SolanaChains, etc.), the error is logged using lggr and
// the failing chain is skipped. This ensures graceful degradation - successfully loaded
// chains remain accessible while failures are visible in logs.
func NewLazyBlockChains(
	ctx context.Context,
	supportedSelectors map[uint64]string,
	loaders map[string]ChainLoader,
	lggr logger.Logger,
) *LazyBlockChains {
	return &LazyBlockChains{
		loadedChains:       make(map[uint64]BlockChain),
		loaders:            loaders,
		supportedSelectors: supportedSelectors,
		ctx:                ctx,
		lggr:               lggr,
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
	family, ok := l.supportedSelectors[selector]
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
	_, ok := l.supportedSelectors[selector]
	return ok
}

// ExistsN checks if all chains with the given selectors are available.
func (l *LazyBlockChains) ExistsN(selectors ...uint64) bool {
	for _, selector := range selectors {
		if _, ok := l.supportedSelectors[selector]; !ok {
			return false
		}
	}

	return true
}

// All returns an iterator over all chains, loading them lazily as they are accessed.
// If a chain fails to load, the error is logged and the chain is skipped.
//
// Note: This method loads chains sequentially during iteration. For faster loading when
// iterating over all chains, consider converting to BlockChains first using ToBlockChains(),
// which loads all chains in parallel, then call All() on the result:
//
//	blockChains, err := lazyChains.ToBlockChains()
//	if err != nil {
//	    // handle error
//	}
//	for selector, chain := range blockChains.All() {
//	    // chains are already loaded
//	}
func (l *LazyBlockChains) All() iter.Seq2[uint64, BlockChain] {
	return func(yield func(uint64, BlockChain) bool) {
		selectors := slices.Collect(maps.Keys(l.supportedSelectors))

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
	chains, err := l.TryEVMChains()
	if err != nil {
		l.lggr.Errorw("Failed to load one or more EVM chains", "error", err)
	}

	return chains
}

// SolanaChains returns a map of all Solana chains, loading them lazily.
// If a chain fails to load, the error is logged and the chain is skipped.
func (l *LazyBlockChains) SolanaChains() map[uint64]solana.Chain {
	chains, err := l.TrySolanaChains()
	if err != nil {
		l.lggr.Errorw("Failed to load one or more Solana chains", "error", err)
	}

	return chains
}

// AptosChains returns a map of all Aptos chains, loading them lazily.
// If a chain fails to load, the error is logged and the chain is skipped.
func (l *LazyBlockChains) AptosChains() map[uint64]aptos.Chain {
	chains, err := l.TryAptosChains()
	if err != nil {
		l.lggr.Errorw("Failed to load one or more Aptos chains", "error", err)
	}

	return chains
}

// SuiChains returns a map of all Sui chains, loading them lazily.
// If a chain fails to load, the error is logged and the chain is skipped.
func (l *LazyBlockChains) SuiChains() map[uint64]sui.Chain {
	chains, err := l.TrySuiChains()
	if err != nil {
		l.lggr.Errorw("Failed to load one or more Sui chains", "error", err)
	}

	return chains
}

// TonChains returns a map of all Ton chains, loading them lazily.
// If a chain fails to load, the error is logged and the chain is skipped.
func (l *LazyBlockChains) TonChains() map[uint64]ton.Chain {
	chains, err := l.TryTonChains()
	if err != nil {
		l.lggr.Errorw("Failed to load one or more Ton chains", "error", err)
	}

	return chains
}

// TronChains returns a map of all Tron chains, loading them lazily.
// If a chain fails to load, the error is logged and the chain is skipped.
func (l *LazyBlockChains) TronChains() map[uint64]tron.Chain {
	chains, err := l.TryTronChains()
	if err != nil {
		l.lggr.Errorw("Failed to load one or more Tron chains", "error", err)
	}

	return chains
}

// TryEVMChains attempts to load all EVM chains and returns any errors encountered.
// Unlike EVMChains, this method returns an error if any chain fails to load.
// The error may contain multiple chain load failures wrapped together.
// Successfully loaded chains are still returned in the map.
func (l *LazyBlockChains) TryEVMChains() (map[uint64]evm.Chain, error) {
	return tryChains[evm.Chain](l, chainsel.FamilyEVM)
}

// TrySolanaChains attempts to load all Solana chains and returns any errors encountered.
// Unlike SolanaChains, this method returns an error if any chain fails to load.
// The error may contain multiple chain load failures wrapped together.
// Successfully loaded chains are still returned in the map.
func (l *LazyBlockChains) TrySolanaChains() (map[uint64]solana.Chain, error) {
	return tryChains[solana.Chain](l, chainsel.FamilySolana)
}

// TryAptosChains attempts to load all Aptos chains and returns any errors encountered.
// Unlike AptosChains, this method returns an error if any chain fails to load.
// The error may contain multiple chain load failures wrapped together.
// Successfully loaded chains are still returned in the map.
func (l *LazyBlockChains) TryAptosChains() (map[uint64]aptos.Chain, error) {
	return tryChains[aptos.Chain](l, chainsel.FamilyAptos)
}

// TrySuiChains attempts to load all Sui chains and returns any errors encountered.
// Unlike SuiChains, this method returns an error if any chain fails to load.
// The error may contain multiple chain load failures wrapped together.
// Successfully loaded chains are still returned in the map.
func (l *LazyBlockChains) TrySuiChains() (map[uint64]sui.Chain, error) {
	return tryChains[sui.Chain](l, chainsel.FamilySui)
}

// TryTonChains attempts to load all Ton chains and returns any errors encountered.
// Unlike TonChains, this method returns an error if any chain fails to load.
// The error may contain multiple chain load failures wrapped together.
// Successfully loaded chains are still returned in the map.
func (l *LazyBlockChains) TryTonChains() (map[uint64]ton.Chain, error) {
	return tryChains[ton.Chain](l, chainsel.FamilyTon)
}

// TryTronChains attempts to load all Tron chains and returns any errors encountered.
// Unlike TronChains, this method returns an error if any chain fails to load.
// The error may contain multiple chain load failures wrapped together.
// Successfully loaded chains are still returned in the map.
func (l *LazyBlockChains) TryTronChains() (map[uint64]tron.Chain, error) {
	return tryChains[tron.Chain](l, chainsel.FamilyTron)
}

// ListChainSelectors returns all available chain selectors with optional filtering.
func (l *LazyBlockChains) ListChainSelectors(options ...ChainSelectorsOption) []uint64 {
	opts := chainSelectorsOptions{}
	for _, option := range options {
		option(&opts)
	}

	selectors := make([]uint64, 0, len(l.supportedSelectors))
	for selector, family := range l.supportedSelectors {
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
	selectors := make([]uint64, 0, len(l.supportedSelectors))
	for selector := range l.supportedSelectors {
		selectors = append(selectors, selector)
	}

	if len(selectors) == 0 {
		return NewBlockChains(make(map[uint64]BlockChain)), nil
	}

	// Load chains in parallel using helper
	results := l.loadChainsParallel(selectors)

	// Collect results
	chains := make(map[uint64]BlockChain)
	for res := range results {
		if res.err != nil {
			return BlockChains{}, fmt.Errorf("failed to load chain %d: %w", res.selector, res.err)
		}
		chains[res.selector] = res.chain
	}

	return NewBlockChains(chains), nil
}

// tryChains is a generic function that attempts to load all chains of a specific family in parallel.
// It returns a map of successfully loaded chains and an error containing all failures.
// Type parameters:
//   - T: the chain type (e.g., evm.Chain, solana.Chain)
//   - PT: pointer to the chain type (e.g., *evm.Chain)
func tryChains[T any, PT interface {
	*T
}](l *LazyBlockChains, family string) (map[uint64]T, error) {
	// Get all selectors for this chain family
	selectors := make([]uint64, 0)
	for selector, f := range l.supportedSelectors {
		if f == family {
			selectors = append(selectors, selector)
		}
	}

	if len(selectors) == 0 {
		return make(map[uint64]T), nil
	}

	// Load chains in parallel using helper
	results := l.loadChainsParallel(selectors)

	// Collect results
	chains := make(map[uint64]T)
	var errs []error

	for res := range results {
		if res.err != nil {
			errs = append(errs, fmt.Errorf("failed to load %s chain %d: %w", family, res.selector, res.err))
			continue
		}

		// Type assertion to convert BlockChain to the specific chain type
		switch c := res.chain.(type) {
		case T:
			chains[res.selector] = c
		case PT:
			if c != nil {
				chains[res.selector] = *c
			}
		}
	}

	if len(errs) > 0 {
		return chains, errors.Join(errs...)
	}

	return chains, nil
}

// chainLoadResult represents the result of loading a single chain.
type chainLoadResult struct {
	selector uint64
	chain    BlockChain
	err      error
}

// loadChainsParallel loads multiple chains in parallel and returns a channel of results.
// The channel is closed when all chains have been loaded.
func (l *LazyBlockChains) loadChainsParallel(selectors []uint64) <-chan chainLoadResult {
	results := make(chan chainLoadResult, len(selectors))
	var wg sync.WaitGroup

	for _, selector := range selectors {
		wg.Add(1)
		go func(sel uint64) {
			defer wg.Done()
			chain, err := l.GetBySelector(sel)
			results <- chainLoadResult{
				selector: sel,
				chain:    chain,
				err:      err,
			}
		}(selector)
	}

	// Close results channel when all goroutines are done
	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}
