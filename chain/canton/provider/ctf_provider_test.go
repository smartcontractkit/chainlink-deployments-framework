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
			name: "valid config",
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
				NumberOfValidators: 5,
				Once:               testutils.DefaultNetworkOnce,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := NewCTFChainProvider(t, tt.giveSelector, tt.giveConfig)
			got, err := provider.Initialize(t.Context())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)

				gotChain, ok := got.(*canton.Chain)
				require.True(t, ok, "expected chain to be of type *canton.Chain")
				assert.Equal(t, tt.giveSelector, gotChain.Selector)
				assert.Len(t, gotChain.Participants, tt.giveConfig.NumberOfValidators)
				// Test that we can retrieve JWTs for each participant
				for _, participant := range gotChain.Participants {
					jwt, err := participant.JWT(t.Context())
					require.NoError(t, err)
					assert.NotEmpty(t, jwt)
				}
			}
		})
	}
}
