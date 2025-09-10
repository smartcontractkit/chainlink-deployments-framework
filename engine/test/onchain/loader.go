package onchain

import (
	"sync"
	"testing"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
)

// ChainFactory is a function type that creates a single chain for a given selector.
type ChainFactory func(t *testing.T, selector uint64) (fchain.BlockChain, error)

// ChainLoader provides functionality to load multiple chains in parallel.
type ChainLoader struct {
	selectors []uint64     // Available chain selectors
	factory   ChainFactory // Factory function to create individual chains
}

// NewChainLoader creates a new ChainLoader with the given selectors and factory.
func NewChainLoader(selectors []uint64, factory ChainFactory) *ChainLoader {
	return &ChainLoader{
		selectors: selectors,
		factory:   factory,
	}
}

// Load creates multiple chains for the specified selectors.
func (l *ChainLoader) Load(t *testing.T, selectors []uint64) ([]fchain.BlockChain, error) {
	t.Helper()

	return loadChainsParallel(t, selectors, l.factory)
}

// LoadN loads the first n available test Aptos chains.
func (l *ChainLoader) LoadN(t *testing.T, n int) ([]fchain.BlockChain, error) {
	t.Helper()

	if len(l.selectors) < n {
		return nil, errMaxSelectors(len(l.selectors))
	}

	return l.Load(t, l.selectors[:n])
}

// loadChainsParallel is a generic helper that loads multiple chains in parallel.
// It takes a slice of selectors and a factory function that creates a chain for each selector.
//
// Returns a slice of chains in the same order as the selectors, or an error if any chain fails to load.
func loadChainsParallel(t *testing.T, selectors []uint64, factory ChainFactory) ([]fchain.BlockChain, error) {
	t.Helper()

	chains := make([]fchain.BlockChain, len(selectors))

	var wg sync.WaitGroup
	errChan := make(chan error, len(selectors))

	for i, selector := range selectors {
		wg.Add(1)
		go func(index int, sel uint64) {
			defer wg.Done()

			c, err := factory(t, sel)
			if err != nil {
				errChan <- err
				return
			}

			chains[index] = c
		}(i, selector)
	}

	wg.Wait()
	close(errChan)

	// Check if any errors occurred
	if len(errChan) > 0 {
		return nil, <-errChan
	}

	return chains, nil
}
