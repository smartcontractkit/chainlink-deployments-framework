package environment

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/cre"
	crecli "github.com/smartcontractkit/chainlink-deployments-framework/cre/cli"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/test/internal/testutils"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/test/onchain"
)

func Test_withChainLoader(t *testing.T) {
	t.Parallel()

	var (
		selector  = uint64(1)
		stubChain = testutils.NewStubChain(selector)
	)

	tests := []struct {
		name            string
		chainLoaderFunc func(t *testing.T, selector uint64) (fchain.BlockChain, error)
		wantErr         string
		wantChainCount  int
		wantChains      []fchain.BlockChain
	}{
		{
			name: "success",
			chainLoaderFunc: func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
				t.Helper()
				return stubChain, nil
			},
			wantChainCount: 1,
			wantChains:     []fchain.BlockChain{stubChain},
		},
		{
			name: "error - chain factory fails",
			chainLoaderFunc: func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
				t.Helper()
				return nil, errors.New("error")
			},
			wantErr:        "error",
			wantChainCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			components := newComponents()

			loader := onchain.NewChainLoader([]uint64{selector}, tt.chainLoaderFunc)
			option := withChainLoader(t, loader, []uint64{selector})
			err := option(components)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, tt.wantChains, components.Chains)
			}
		})
	}
}

func Test_withChainLoaderN(t *testing.T) {
	t.Parallel()

	var (
		selector  = uint64(1)
		stubChain = testutils.NewStubChain(selector)
	)

	tests := []struct {
		name            string
		availableChains []uint64
		chainLoaderFunc func(t *testing.T, selector uint64) (fchain.BlockChain, error)
		requestedCount  int
		wantErr         string
		wantChains      []fchain.BlockChain
	}{
		{
			name:            "success",
			availableChains: []uint64{1, 2, 3},
			chainLoaderFunc: func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
				t.Helper()
				return stubChain, nil
			},
			requestedCount: 1,
			wantChains:     []fchain.BlockChain{stubChain},
		},
		{
			name:            "error - chain factory fails",
			availableChains: []uint64{1, 2, 3},
			chainLoaderFunc: func(t *testing.T, selector uint64) (fchain.BlockChain, error) {
				t.Helper()
				return nil, errors.New("error")
			},
			requestedCount: 1,
			wantErr:        "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			components := newComponents()

			loader := onchain.NewChainLoader(tt.availableChains, tt.chainLoaderFunc)
			option := withChainLoaderN(t, loader, tt.requestedCount)
			err := option(components)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, tt.wantChains, components.Chains)
			}
		})
	}
}

func Test_WithStellarContainer(t *testing.T) {
	// This test mutates the package-level newStellarContainerLoader var (restored via
	// t.Cleanup), so the parent must NOT run in parallel: a non-parallel parent's entire
	// run — including its parallel subtests and cleanup — completes in an isolated window
	// before any other top-level test starts, which keeps the global mutation race-free.
	// The subtests below call t.Parallel() and run concurrently once the stub is installed.
	orig := newStellarContainerLoader
	t.Cleanup(func() { newStellarContainerLoader = orig })

	const selector = uint64(17301180955411967724) // chainsel.STELLAR_LOCALNET
	stubChain := testutils.NewStubChain(selector)
	newStellarContainerLoader = func() *onchain.ChainLoader {
		return onchain.NewChainLoader([]uint64{selector}, func(t *testing.T, _ uint64) (fchain.BlockChain, error) {
			t.Helper()
			return stubChain, nil
		})
	}

	t.Run("WithStellarContainer", func(t *testing.T) {
		t.Parallel()
		cmps := newComponents()
		require.NoError(t, WithStellarContainer(t, []uint64{selector})(cmps))
		require.ElementsMatch(t, []fchain.BlockChain{stubChain}, cmps.Chains)
	})

	t.Run("WithStellarContainerN", func(t *testing.T) {
		t.Parallel()
		cmps := newComponents()
		require.NoError(t, WithStellarContainerN(t, 1)(cmps))
		require.ElementsMatch(t, []fchain.BlockChain{stubChain}, cmps.Chains)
	})
}

func Test_WithCRERunner(t *testing.T) {
	t.Parallel()

	cmps := newComponents()
	require.Nil(t, cmps.CRERunner)

	runner := crecli.NewCLIRunner("/path/to/cre", "")
	creR := cre.NewRunner(cre.WithCLI(runner))
	option := WithCRERunner(creR)
	err := option(cmps)

	require.NoError(t, err)
	require.Equal(t, creR, cmps.CRERunner)
	require.Equal(t, runner, cmps.CRERunner.CLI())
}
