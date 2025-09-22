package onchain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/test/internal/testutils"
)

func TestLoader_Load(t *testing.T) {
	t.Parallel()

	var (
		csel1 uint64 = 1
		csel2 uint64 = 2
		csel3 uint64 = 3
	)

	// Create a mock factory that returns test chains
	mockFactory := func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
		t.Helper()
		return testutils.NewStubChain(selector), nil
	}

	// Create a mock factory that returns errors
	errorFactory := func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
		t.Helper()
		return nil, errors.New("mock factory error")
	}

	tests := []struct {
		name          string
		factory       ChainFactory
		selectors     []uint64
		wantCount     int
		wantSelectors []uint64
		wantErr       string
	}{
		{
			name:          "loads single chain",
			factory:       mockFactory,
			selectors:     []uint64{csel1},
			wantCount:     1,
			wantSelectors: []uint64{csel1},
		},
		{
			name:          "loads multiple chains",
			factory:       mockFactory,
			selectors:     []uint64{csel1, csel2, csel3},
			wantCount:     3,
			wantSelectors: []uint64{csel1, csel2, csel3},
		},
		{
			name:      "loads empty selector list",
			factory:   mockFactory,
			selectors: []uint64{},
			wantCount: 0,
		},
		{
			name:      "returns error when factory fails",
			factory:   errorFactory,
			selectors: []uint64{csel1},
			wantErr:   "mock factory error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loader := NewChainLoader(
				[]uint64{csel1, csel2, csel3},
				tt.factory,
			)
			chains, err := loader.Load(t, tt.selectors)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Len(t, chains, tt.wantCount)

				gotSelectors := make([]uint64, len(chains))
				for i, chain := range chains {
					gotSelectors[i] = chain.ChainSelector()
				}

				assert.ElementsMatch(t, tt.wantSelectors, gotSelectors)
			}
		})
	}
}

func TestLoader_LoadN(t *testing.T) {
	t.Parallel()

	var (
		csel1 uint64 = 1
		csel2 uint64 = 2
		csel3 uint64 = 3
	)

	// Create a mock factory that returns test chains
	mockFactory := func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
		t.Helper()
		return testutils.NewStubChain(selector), nil
	}

	// Create a mock factory that returns errors
	errorFactory := func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
		t.Helper()
		return nil, errors.New("mock factory error")
	}

	tests := []struct {
		name          string
		selectors     []uint64
		factory       ChainFactory
		n             int
		wantCount     int
		wantSelectors []uint64
		wantErr       string
	}{
		{
			name:          "loads single chain",
			selectors:     []uint64{csel1},
			factory:       mockFactory,
			n:             1,
			wantCount:     1,
			wantSelectors: []uint64{csel1},
		},
		{
			name:          "loads multiple chains",
			selectors:     []uint64{csel1, csel2, csel3},
			factory:       mockFactory,
			n:             3,
			wantCount:     3,
			wantSelectors: []uint64{csel1, csel2, csel3},
		},
		{
			name:          "loads subset of available chains",
			selectors:     []uint64{csel1, csel2, csel3},
			factory:       mockFactory,
			n:             2,
			wantCount:     2,
			wantSelectors: []uint64{csel1, csel2},
		},
		{
			name:      "returns error when n exceeds available selectors",
			selectors: []uint64{csel1, csel2},
			factory:   mockFactory,
			n:         3,
			wantErr:   "maximum of 2 selectors",
		},
		{
			name:      "returns error when factory fails",
			selectors: []uint64{csel1, csel2, csel3},
			factory:   errorFactory,
			n:         2,
			wantErr:   "mock factory error",
		},
		{
			name:          "handles zero chains request",
			selectors:     []uint64{csel1, csel2, csel3},
			factory:       mockFactory,
			n:             0,
			wantCount:     0,
			wantSelectors: []uint64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loader := NewChainLoader(
				tt.selectors,
				tt.factory,
			)

			chains, err := loader.LoadN(t, tt.n)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Len(t, chains, tt.wantCount)

				gotSelectors := make([]uint64, len(chains))
				for i, chain := range chains {
					gotSelectors[i] = chain.ChainSelector()
				}

				assert.ElementsMatch(t, tt.wantSelectors, gotSelectors)

				// Verify chains have correct selectors in order
				for i, chain := range chains {
					expectedSelector := tt.selectors[i]
					assert.Equal(t, expectedSelector, chain.ChainSelector())
				}
			}
		})
	}
}
