package environment

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/test/internal/testutils"
)

func TestComponents_AddChains(t *testing.T) {
	t.Parallel()

	var (
		stubChain1 = testutils.NewStubChain(1)
		stubChain2 = testutils.NewStubChain(2)
		stubChain3 = testutils.NewStubChain(3)
	)

	tests := []struct {
		name        string
		setupChains func() []fchain.BlockChain // Initial chains to add before test
		chainsToAdd func() []fchain.BlockChain // Chains to add during test
		wantCount   int
		assert      func(t *testing.T, c *components) // Custom validation function
	}{
		{
			name:        "adds single chain",
			setupChains: func() []fchain.BlockChain { return nil },
			chainsToAdd: func() []fchain.BlockChain { return []fchain.BlockChain{stubChain1} },
			wantCount:   1,
			assert: func(t *testing.T, c *components) {
				t.Helper()

				assert.Equal(t, uint64(1), c.Chains[0].ChainSelector())
			},
		},
		{
			name:        "adds multiple chains",
			setupChains: func() []fchain.BlockChain { return nil },
			chainsToAdd: func() []fchain.BlockChain {
				return []fchain.BlockChain{stubChain1, stubChain2, stubChain3}
			},
			wantCount: 3,
			assert: func(t *testing.T, c *components) {
				t.Helper()

				assert.Equal(t, uint64(1), c.Chains[0].ChainSelector())
				assert.Equal(t, uint64(2), c.Chains[1].ChainSelector())
				assert.Equal(t, uint64(3), c.Chains[2].ChainSelector())
			},
		},
		{
			name:        "adds chains to existing collection",
			setupChains: func() []fchain.BlockChain { return []fchain.BlockChain{stubChain1} },
			chainsToAdd: func() []fchain.BlockChain {
				return []fchain.BlockChain{stubChain2, stubChain3}
			},
			wantCount: 3,
			assert: func(t *testing.T, c *components) {
				t.Helper()

				assert.Equal(t, uint64(1), c.Chains[0].ChainSelector())
				assert.Equal(t, uint64(2), c.Chains[1].ChainSelector())
				assert.Equal(t, uint64(3), c.Chains[2].ChainSelector())
			},
		},
		{
			name:        "adds empty chains",
			setupChains: func() []fchain.BlockChain { return nil },
			chainsToAdd: func() []fchain.BlockChain { return []fchain.BlockChain{} },
			wantCount:   0,
		},
		{
			name:        "does not add nil chain",
			setupChains: func() []fchain.BlockChain { return nil },
			chainsToAdd: func() []fchain.BlockChain { return []fchain.BlockChain{nil} },
			wantCount:   0,
		},
		{
			name:        "preserves order",
			setupChains: func() []fchain.BlockChain { return nil },
			chainsToAdd: func() []fchain.BlockChain {
				chains := make([]fchain.BlockChain, 10)
				for i := range uint64(10) {
					chains[i] = testutils.NewStubChain(i)
				}

				return chains
			},
			wantCount: 10,
			assert: func(t *testing.T, c *components) {
				t.Helper()

				for i, chain := range c.Chains {
					assert.Equal(t, uint64(i), chain.ChainSelector()) //nolint:gosec // G115: this will not overflow from a range index
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newComponents()

			// Setup initial chains if any
			if setupChains := tt.setupChains(); len(setupChains) > 0 {
				c.AddChains(setupChains...)
			}

			// Add the test chains
			chainsToAdd := tt.chainsToAdd()
			c.AddChains(chainsToAdd...)

			// Verify expected count
			require.Len(t, c.Chains, tt.wantCount)

			// Run custom validation
			if tt.assert != nil {
				tt.assert(t, c)
			}
		})
	}

	// Keep the concurrent test separate as it has different structure
	t.Run("concurrently adds chains", func(t *testing.T) {
		t.Parallel()

		c := newComponents()
		const numGoroutines = 10
		const chainsPerGoroutine = 5

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Launch multiple goroutines that add chains concurrently
		for i := range numGoroutines {
			go func(goroutineID int) {
				defer wg.Done()

				chains := make([]fchain.BlockChain, chainsPerGoroutine)
				for j := range chainsPerGoroutine {
					selector := goroutineID*chainsPerGoroutine + j
					chains[j] = testutils.NewStubChain(uint64(selector)) //nolint:gosec // G115: this will not overflow from a range
				}

				c.AddChains(chains...)
			}(i)
		}

		wg.Wait()

		// Verify all chains were added
		wantTotal := numGoroutines * chainsPerGoroutine
		assert.Len(t, c.Chains, wantTotal)

		// Verify no chains are nil (indicating no race conditions corrupted the data)
		for i, chain := range c.Chains {
			assert.NotNil(t, chain, "Chain at index %d should not be nil", i)
		}
	})
}

func TestNewComponents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		assert func(t *testing.T, c *components)
	}{
		{
			name: "initializes empty chains",
			assert: func(t *testing.T, c *components) {
				t.Helper()

				assert.NotNil(t, c.Chains)
				assert.Empty(t, c.Chains)
				assert.Equal(t, 0, cap(c.Chains)) // Should start with zero capacity
			},
		},
		{
			name: "initializes with mutex",
			assert: func(t *testing.T, c *components) {
				t.Helper()

				// Verify mutex is properly initialized by attempting to lock/unlock
				c.mu.Lock()
				// Verify we can access the chains while locked
				initialLen := len(c.Chains)
				c.mu.Unlock() // Should not panic

				assert.Equal(t, 0, initialLen)
			},
		},
		{
			name: "initializes catalogEnabled to false",
			assert: func(t *testing.T, c *components) {
				t.Helper()

				assert.False(t, c.catalogEnabled, "catalogEnabled should be initialized to false")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newComponents()
			tt.assert(t, c)
		})
	}
}
