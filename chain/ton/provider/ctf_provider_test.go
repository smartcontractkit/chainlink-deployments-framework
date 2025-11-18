package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/testutils"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/ton"
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
				Once: testutils.DefaultNetworkOnce,
			},
			wantErr: "",
		},
		{
			name: "missing sync.Once instance",
			config: CTFChainProviderConfig{
				Once: nil,
			},
			wantErr: "sync.Once instance is required",
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

	var chainSelector = chainsel.TEST_1000.Selector

	tests := []struct {
		name         string
		giveSelector uint64
		giveConfig   CTFChainProviderConfig
		wantErr      string
	}{
		{
			name:         "valid initialization",
			giveSelector: chainSelector,
			giveConfig: CTFChainProviderConfig{
				Once: testutils.DefaultNetworkOnce,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewCTFChainProvider(t, tt.giveSelector, tt.giveConfig)

			got, err := p.Initialize(t.Context())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, p.chain)

				// Check that the chain is of type ton.Chain and has the expected fields
				gotChain, ok := got.(ton.Chain)
				require.True(t, ok, "expected got to be of type ton.Chain")
				assert.Equal(t, tt.giveSelector, gotChain.Selector)
				assert.NotEmpty(t, gotChain.Client)
				assert.NotEmpty(t, gotChain.Wallet)
				assert.NotEmpty(t, gotChain.WalletAddress)
			}
		})
	}
}

func Test_CTFChainProvider_Name(t *testing.T) {
	t.Parallel()

	p := &CTFChainProvider{}
	assert.Equal(t, "Ton CTF Chain Provider", p.Name())
}

func Test_CTFChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	p := &CTFChainProvider{selector: chainsel.TEST_1000.Selector}
	assert.Equal(t, chainsel.TEST_1000.Selector, p.ChainSelector())
}

func Test_CTFChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	chain := &ton.Chain{}

	p := &CTFChainProvider{
		chain: chain,
	}

	assert.Equal(t, *chain, p.BlockChain())
}

func Test_CTFChainProvider_getImage(t *testing.T) {
	t.Parallel()

	// Test default image
	p1 := &CTFChainProvider{config: CTFChainProviderConfig{}}
	assert.Equal(t, "ghcr.io/neodix42/mylocalton-docker:v3.7", p1.getImage())

	// Test custom image
	p2 := &CTFChainProvider{config: CTFChainProviderConfig{Image: "ghcr.io/neodix42/mylocalton-docker:latest"}}
	assert.Equal(t, "ghcr.io/neodix42/mylocalton-docker:latest", p2.getImage())
}
