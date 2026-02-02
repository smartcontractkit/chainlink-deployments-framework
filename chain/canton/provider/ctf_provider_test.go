package provider

import (
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/canton"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/testutils"
)

func Test_CTFChainProviderConfig_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  CTFChainProviderConfig
		wantErr string
	}{
		{
			name: "valid config",
			config: CTFChainProviderConfig{
				NumberOfValidators: 4,
				Once:               testutils.DefaultNetworkOnce,
			},
			wantErr: "",
		},
		{
			name: "missing sync.Once",
			config: CTFChainProviderConfig{
				NumberOfValidators: 3,
				Once:               nil,
			},
			wantErr: "sync.Once instance is required",
		},
		{
			name: "invalid number of validators",
			config: CTFChainProviderConfig{
				NumberOfValidators: -99,
				Once:               testutils.DefaultNetworkOnce,
			},
			wantErr: "number of validators must be greater than zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.validate()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_CTFChainProvider_Initialize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		giveSelector uint64
		giveConfig   CTFChainProviderConfig
		wantErr      string
	}{
		{
			name:         "valid initialization",
			giveSelector: chainsel.CANTON_LOCALNET.Selector,
			giveConfig: CTFChainProviderConfig{
				NumberOfValidators: 1,
				Once:               testutils.DefaultNetworkOnce,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := NewCTFChainProvider(t, tt.giveSelector, tt.giveConfig)
			chain, err := provider.Initialize(t.Context())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, chain)

				gotChain, ok := chain.(*canton.Chain)
				require.True(t, ok, "expected chain to be of type *canton.Chain")
				assert.Equal(t, tt.giveSelector, gotChain.Selector)
				assert.Len(t, gotChain.Participants, tt.giveConfig.NumberOfValidators)
				// Test that we can retrieve JWTs for each participant
				for _, participant := range gotChain.Participants {
					jwt, err := participant.JWTProvider.Token(t.Context())
					require.NoError(t, err)
					assert.NotEmpty(t, jwt)
				}

				// Check that subsequent calls to Initialize don't re-initialize the chain
				chainBefore := provider.chain
				chain2, err := provider.Initialize(t.Context())
				require.NoError(t, err)
				require.Equal(t, chain, chain2)
				require.Same(t, chainBefore, provider.chain)
			}
		})
	}
}

func Test_CTFChainProvider_Name(t *testing.T) {
	t.Parallel()

	provider := NewCTFChainProvider(t, chainsel.CANTON_LOCALNET.Selector, CTFChainProviderConfig{
		NumberOfValidators: 3,
		Once:               testutils.DefaultNetworkOnce,
	})

	require.Equal(t, "Canton CTF Chain Provider", provider.Name())
}

func Test_CTFChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	selector := chainsel.CANTON_LOCALNET.Selector
	provider := NewCTFChainProvider(t, selector, CTFChainProviderConfig{
		NumberOfValidators: 3,
		Once:               testutils.DefaultNetworkOnce,
	})

	require.Equal(t, selector, provider.ChainSelector())
}

func Test_CTFChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	chain := &canton.Chain{
		ChainMetadata: canton.ChainMetadata{Selector: chainsel.CANTON_LOCALNET.Selector},
		Participants: []canton.Participant{
			{Name: "Participant 1"},
			{Name: "Participant 2"},
		},
	}

	provider := &CTFChainProvider{
		chain: chain,
	}

	require.Equal(t, *chain, provider.BlockChain())
}
