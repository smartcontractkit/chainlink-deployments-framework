package chain

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"
	"reflect"
	"slices"
	"sync"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/sui"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/ton"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

var ErrBlockChainNotFound = errors.New("blockchain not found")

var _ BlockChain = evm.Chain{}
var _ BlockChain = solana.Chain{}
var _ BlockChain = aptos.Chain{}
var _ BlockChain = sui.Chain{}
var _ BlockChain = ton.Chain{}
var _ BlockChain = tron.Chain{}

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

// BlockChains represents a collection of chains that supports both eager and lazy loading.
// It provides querying capabilities for different types of chains.
// The struct can operate in two modes:
// - Eager mode: All chains are loaded upfront (original behavior, backward compatible)
// - Lazy mode: Chains are loaded on-demand when first accessed (new feature)
type BlockChains struct {
	// Eager loading fields (used when lazyState = nil)
	chains map[uint64]BlockChain

	// Lazy loading state (used when non-nil)
	lazyState *lazyLoadingState
}

// lazyLoadingState holds the state for lazy loading mode.
type lazyLoadingState struct {
	mu                 sync.RWMutex
	loadedChains       map[uint64]BlockChain
	loaders            map[string]ChainLoader // keyed by chain family
	supportedSelectors map[uint64]string      // maps selector to chain family
	ctx                context.Context        //nolint:containedctx // Context is needed for lazy loading operations
	lggr               logger.Logger
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
		chains:    chains,
		lazyState: nil, // nil indicates eager loading
	}
}

// NewLazyBlockChains creates a BlockChains instance that defers chain loading until first access.
// This improves initialization performance by avoiding unnecessary chain connections.
// Chains are loaded on-demand when accessed via GetBySelector, EVMChains, SolanaChains, etc.
//
// Parameters:
//   - ctx: Context for chain loading operations
//   - supportedSelectors: Maps chain selectors to their family (e.g., "evm", "solana", "aptos")
//   - loaders: Provides the ChainLoader for each family
//   - lggr: Logger for recording chain loading events and errors
//
// If a chain fails to load during access, the error is logged and the failing chain is skipped.
// This ensures graceful degradation - successfully loaded chains remain accessible while failures
// are visible in logs.
func NewLazyBlockChains(
	ctx context.Context,
	supportedSelectors map[uint64]string,
	loaders map[string]ChainLoader,
	lggr logger.Logger,
) BlockChains {
	return BlockChains{
		lazyState: &lazyLoadingState{
			loadedChains:       make(map[uint64]BlockChain),
			loaders:            loaders,
			supportedSelectors: supportedSelectors,
			ctx:                ctx,
			lggr:               lggr,
		},
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

// GetBySelector returns a blockchain by its selector.
// In eager mode, returns the pre-loaded chain immediately.
// In lazy mode, loads the chain on-demand if not already loaded.
func (b *BlockChains) GetBySelector(selector uint64) (BlockChain, error) {
	if b.lazyState != nil {
		return b.getBySelectorLazy(selector)
	}

	return b.getBySelectorEager(selector)
}

// getBySelectorEager returns a pre-loaded chain (eager mode).
func (b *BlockChains) getBySelectorEager(selector uint64) (BlockChain, error) {
	if chain, ok := b.chains[selector]; ok {
		return chain, nil
	}

	return nil, ErrBlockChainNotFound
}

// getBySelectorLazy loads and returns a chain on-demand (lazy mode).
func (b *BlockChains) getBySelectorLazy(selector uint64) (BlockChain, error) {
	lazy := b.lazyState

	// Fast path: check if already loaded
	lazy.mu.RLock()
	if chain, ok := lazy.loadedChains[selector]; ok {
		lazy.mu.RUnlock()
		return chain, nil
	}
	lazy.mu.RUnlock()

	// Slow path: need to load the chain
	lazy.mu.Lock()
	defer lazy.mu.Unlock()

	// Double-check after acquiring write lock
	if chain, ok := lazy.loadedChains[selector]; ok {
		return chain, nil
	}

	// Check if the chain is available
	family, ok := lazy.supportedSelectors[selector]
	if !ok {
		return nil, ErrBlockChainNotFound
	}

	// Get the loader for this family
	loader, ok := lazy.loaders[family]
	if !ok {
		return nil, ErrBlockChainNotFound
	}

	// Load the chain
	chain, err := loader.Load(lazy.ctx, selector)
	if err != nil {
		return nil, err
	}

	// Cache the loaded chain
	lazy.loadedChains[selector] = chain

	return chain, nil
}

// Exists checks if a chain with the given selector exists (not necessarily loaded).
func (b BlockChains) Exists(selector uint64) bool {
	if b.lazyState != nil {
		_, ok := b.lazyState.supportedSelectors[selector]
		return ok
	}
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
// In eager mode, iterates over pre-loaded chains immediately.
// In lazy mode, loads chains on-demand during iteration. If a chain fails to load,
// the error is logged and the chain is skipped.
//
// Note: For lazy mode, this method loads chains sequentially during iteration.
// For faster loading when iterating over all chains, consider converting to eager mode first
// using ToEagerBlockChains(), which loads all chains in parallel.
func (b *BlockChains) All() iter.Seq2[uint64, BlockChain] {
	if b.lazyState != nil {
		return b.allLazy()
	}

	return maps.All(b.chains)
}

// allLazy returns an iterator that loads chains on-demand.
func (b *BlockChains) allLazy() iter.Seq2[uint64, BlockChain] {
	lazy := b.lazyState
	return func(yield func(uint64, BlockChain) bool) {
		selectors := slices.Collect(maps.Keys(lazy.supportedSelectors))

		// Sort for consistent iteration order
		slices.Sort(selectors)

		for _, selector := range selectors {
			chain, err := b.GetBySelector(selector)
			if err != nil {
				lazy.lggr.Errorw("Failed to load chain during iteration",
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

// EVMChains returns a map of all EVM chains with their selectors.
// In lazy mode, chains are loaded on-demand. If a chain fails to load,
// the error is logged and the chain is skipped.
func (b *BlockChains) EVMChains() map[uint64]evm.Chain {
	if b.lazyState != nil {
		chains, err := tryChainsByFamily[evm.Chain, *evm.Chain](b, "evm")
		if err != nil {
			b.lazyState.lggr.Errorw("Failed to load one or more EVM chains", "error", err)
		}

		return chains
	}

	return getChainsByType[evm.Chain, *evm.Chain](*b)
}

// SolanaChains returns a map of all Solana chains with their selectors.
// In lazy mode, chains are loaded on-demand. If a chain fails to load,
// the error is logged and the chain is skipped.
func (b *BlockChains) SolanaChains() map[uint64]solana.Chain {
	if b.lazyState != nil {
		chains, err := tryChainsByFamily[solana.Chain, *solana.Chain](b, "solana")
		if err != nil {
			b.lazyState.lggr.Errorw("Failed to load one or more Solana chains", "error", err)
		}

		return chains
	}

	return getChainsByType[solana.Chain, *solana.Chain](*b)
}

// AptosChains returns a map of all Aptos chains with their selectors.
// In lazy mode, chains are loaded on-demand. If a chain fails to load,
// the error is logged and the chain is skipped.
func (b *BlockChains) AptosChains() map[uint64]aptos.Chain {
	if b.lazyState != nil {
		chains, err := tryChainsByFamily[aptos.Chain, *aptos.Chain](b, "aptos")
		if err != nil {
			b.lazyState.lggr.Errorw("Failed to load one or more Aptos chains", "error", err)
		}

		return chains
	}

	return getChainsByType[aptos.Chain, *aptos.Chain](*b)
}

// SuiChains returns a map of all Sui chains with their selectors.
// In lazy mode, chains are loaded on-demand. If a chain fails to load,
// the error is logged and the chain is skipped.
func (b *BlockChains) SuiChains() map[uint64]sui.Chain {
	if b.lazyState != nil {
		chains, err := tryChainsByFamily[sui.Chain, *sui.Chain](b, "sui")
		if err != nil {
			b.lazyState.lggr.Errorw("Failed to load one or more Sui chains", "error", err)
		}

		return chains
	}

	return getChainsByType[sui.Chain, *sui.Chain](*b)
}

// TonChains returns a map of all Ton chains with their selectors.
// In lazy mode, chains are loaded on-demand. If a chain fails to load,
// the error is logged and the chain is skipped.
func (b *BlockChains) TonChains() map[uint64]ton.Chain {
	if b.lazyState != nil {
		chains, err := tryChainsByFamily[ton.Chain, *ton.Chain](b, "ton")
		if err != nil {
			b.lazyState.lggr.Errorw("Failed to load one or more Ton chains", "error", err)
		}

		return chains
	}

	return getChainsByType[ton.Chain, *ton.Chain](*b)
}

// TronChains returns a map of all Tron chains with their selectors.
// In lazy mode, chains are loaded on-demand. If a chain fails to load,
// the error is logged and the chain is skipped.
func (b *BlockChains) TronChains() map[uint64]tron.Chain {
	if b.lazyState != nil {
		chains, err := tryChainsByFamily[tron.Chain, *tron.Chain](b, "tron")
		if err != nil {
			b.lazyState.lggr.Errorw("Failed to load one or more Tron chains", "error", err)
		}

		return chains
	}

	return getChainsByType[tron.Chain, *tron.Chain](*b)
}

// ChainSelectorsOption defines a function type for configuring ChainSelectors
type ChainSelectorsOption func(*chainSelectorsOptions)

type chainSelectorsOptions struct {
	// Use map for faster lookups
	includedFamilies  map[string]struct{}
	excludedChainSels map[uint64]struct{}
}

// WithFamily returns an option to filter chains by family (evm, solana, aptos)
// Use constants from chainsel package eg WithFamily(chainsel.FamilySolana)
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
// - WithFamily: filter by family eg WithFamily(chainsel.FamilySolana)
// - WithChainSelectorsExclusion: exclude specific chain selectors
func (b BlockChains) ListChainSelectors(options ...ChainSelectorsOption) []uint64 {
	opts := chainSelectorsOptions{}

	// Apply all provided options
	for _, option := range options {
		option(&opts)
	}

	var selectors []uint64

	if b.lazyState != nil {
		selectors = make([]uint64, 0, len(b.lazyState.supportedSelectors))
		for selector, family := range b.lazyState.supportedSelectors {
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
	} else {
		selectors = make([]uint64, 0, len(b.chains))
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
	}

	// Sort for consistent output
	slices.Sort(selectors)

	return selectors
}

// IsLazy returns true if the BlockChains instance uses lazy loading.
func (b BlockChains) IsLazy() bool {
	return b.lazyState != nil
}

// ToEagerBlockChains converts lazy-loaded chains to eagerly-loaded chains.
// This loads all available chains in parallel and returns a new BlockChains instance.
// If already using eager loading, returns a copy of the current instance.
// This is useful for operations that need all chains loaded upfront.
func (b *BlockChains) ToEagerBlockChains() (BlockChains, error) {
	if b.lazyState == nil {
		// Already eager, return a copy
		return NewBlockChains(b.chains), nil
	}

	// Get all selectors
	selectors := make([]uint64, 0, len(b.lazyState.supportedSelectors))
	for selector := range b.lazyState.supportedSelectors {
		selectors = append(selectors, selector)
	}

	if len(selectors) == 0 {
		return NewBlockChains(make(map[uint64]BlockChain)), nil
	}

	// Load chains in parallel
	results := b.loadChainsParallel(selectors)

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

// chainLoadResult represents the result of loading a single chain.
type chainLoadResult struct {
	selector uint64
	chain    BlockChain
	err      error
}

// loadChainsParallel loads multiple chains in parallel and returns a channel of results.
// The channel is closed when all chains have been loaded.
func (b *BlockChains) loadChainsParallel(selectors []uint64) <-chan chainLoadResult {
	results := make(chan chainLoadResult, len(selectors))
	var wg sync.WaitGroup

	for _, selector := range selectors {
		wg.Add(1)
		go func(sel uint64) {
			defer wg.Done()
			chain, err := b.GetBySelector(sel)
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

// tryChainsByFamily loads all chains of a specific family in parallel (lazy mode only).
// It returns a map of successfully loaded chains and an error containing all failures.
// This is a standalone function because Go doesn't support generic methods.
func tryChainsByFamily[T any, PT interface {
	*T
}](b *BlockChains, family string) (map[uint64]T, error) {
	// Get all selectors for this chain family
	selectors := make([]uint64, 0)
	for selector, f := range b.lazyState.supportedSelectors {
		if f == family {
			selectors = append(selectors, selector)
		}
	}

	if len(selectors) == 0 {
		return make(map[uint64]T), nil
	}

	// Load chains in parallel
	results := b.loadChainsParallel(selectors)

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
				val := reflect.ValueOf(c)
				if val.Kind() == reflect.Ptr && !val.IsNil() {
					elem := val.Elem()
					if elem.CanInterface() {
						if v, ok := elem.Interface().(T); ok {
							chains[res.selector] = v
						}
					}
				}
			}
		}
	}

	if len(errs) > 0 {
		return chains, errors.Join(errs...)
	}

	return chains, nil
}

// getChainsByType is a helper function to extract chains of a specific type from BlockChains (eager mode).
// It accepts two type parameters: VT for the target type and PT for pointer types of the same chain type.
// eg getChainsByType[evm.Chain, *evm.Chain](b BlockChains) returns a map of uint64 to evm.Chain.
// It handles both value and pointer types, allowing for flexibility in how chains are stored.
func getChainsByType[VT any, PT any](b BlockChains) map[uint64]VT {
	chains := make(map[uint64]VT, len(b.chains))
	for sel, chain := range b.chains {
		switch c := any(chain).(type) {
		case VT:
			chains[sel] = c
		case PT:
			val := reflect.ValueOf(c)
			if val.Kind() == reflect.Ptr && !val.IsNil() {
				elem := val.Elem()
				if elem.CanInterface() {
					if v, ok := elem.Interface().(VT); ok {
						chains[sel] = v
					}
				}
			}
		}
	}

	return chains
}
