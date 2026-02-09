package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/stellar"
)

func Test_RPCChainProviderConfig_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  RPCChainProviderConfig
		wantErr string
	}{
		{
			name: "valid config",
			config: RPCChainProviderConfig{
				NetworkPassphrase: "Test SDF Network ; September 2015",
				FriendbotURL:      "https://friendbot.stellar.org",
				SorobanRPCURL:     "https://soroban-testnet.stellar.org",
			},
			wantErr: "",
		},
		{
			name: "missing network passphrase",
			config: RPCChainProviderConfig{
				NetworkPassphrase: "",
				FriendbotURL:      "https://friendbot.stellar.org",
				SorobanRPCURL:     "https://soroban-testnet.stellar.org",
			},
			wantErr: "network passphrase is required",
		},
		{
			name: "missing friendbot URL",
			config: RPCChainProviderConfig{
				NetworkPassphrase: "Test SDF Network ; September 2015",
				FriendbotURL:      "",
				SorobanRPCURL:     "https://soroban-testnet.stellar.org",
			},
			wantErr: "friendbot URL is required",
		},
		{
			name: "missing soroban RPC URL",
			config: RPCChainProviderConfig{
				NetworkPassphrase: "Test SDF Network ; September 2015",
				FriendbotURL:      "https://friendbot.stellar.org",
				SorobanRPCURL:     "",
			},
			wantErr: "soroban RPC URL is required",
		},
		{
			name: "all fields missing",
			config: RPCChainProviderConfig{
				NetworkPassphrase: "",
				FriendbotURL:      "",
				SorobanRPCURL:     "",
			},
			wantErr: "network passphrase is required",
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

func Test_RPCChainProvider_Initialize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		giveSelector uint64
		giveConfig   RPCChainProviderConfig
		giveChain    *stellar.Chain // pre-existing chain for re-initialization test
		wantErr      string
	}{
		{
			name:         "valid initialization",
			giveSelector: 12345,
			giveConfig: RPCChainProviderConfig{
				NetworkPassphrase: "Test SDF Network ; September 2015",
				FriendbotURL:      "https://friendbot.stellar.org",
				SorobanRPCURL:     "https://soroban-testnet.stellar.org",
			},
		},
		{
			name:         "re-initialize returns existing chain",
			giveSelector: 67890,
			giveConfig: RPCChainProviderConfig{
				NetworkPassphrase: "Test SDF Network ; September 2015",
				FriendbotURL:      "https://friendbot.stellar.org",
				SorobanRPCURL:     "https://soroban-testnet.stellar.org",
			},
			giveChain: &stellar.Chain{
				ChainMetadata: stellar.ChainMetadata{Selector: 67890},
			},
		},
		{
			name:         "fails config validation - missing network passphrase",
			giveSelector: 12345,
			giveConfig: RPCChainProviderConfig{
				NetworkPassphrase: "",
				FriendbotURL:      "https://friendbot.stellar.org",
				SorobanRPCURL:     "https://soroban-testnet.stellar.org",
			},
			wantErr: "network passphrase is required",
		},
		{
			name:         "fails config validation - missing friendbot URL",
			giveSelector: 12345,
			giveConfig: RPCChainProviderConfig{
				NetworkPassphrase: "Test SDF Network ; September 2015",
				FriendbotURL:      "",
				SorobanRPCURL:     "https://soroban-testnet.stellar.org",
			},
			wantErr: "friendbot URL is required",
		},
		{
			name:         "fails config validation - missing soroban RPC URL",
			giveSelector: 12345,
			giveConfig: RPCChainProviderConfig{
				NetworkPassphrase: "Test SDF Network ; September 2015",
				FriendbotURL:      "https://friendbot.stellar.org",
				SorobanRPCURL:     "",
			},
			wantErr: "soroban RPC URL is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewRPCChainProvider(tt.giveSelector, tt.giveConfig)
			if tt.giveChain != nil {
				p.chain = tt.giveChain
			}

			got, err := p.Initialize(context.Background())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, p.chain)

			gotChain, ok := got.(*stellar.Chain)
			require.True(t, ok, "expected got to be of type *stellar.Chain")
			assert.Equal(t, tt.giveSelector, gotChain.Selector)

			// If we had a pre-existing chain, verify it's the same instance
			if tt.giveChain != nil {
				assert.Equal(t, tt.giveChain, gotChain)
			}
		})
	}
}

func Test_RPCChainProvider_Name(t *testing.T) {
	t.Parallel()

	p := &RPCChainProvider{}
	assert.Equal(t, "Stellar RPC Chain Provider", p.Name())
}

func Test_RPCChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		giveSelector uint64
	}{
		{
			name:         "selector 12345",
			giveSelector: 12345,
		},
		{
			name:         "selector 0",
			giveSelector: 0,
		},
		{
			name:         "large selector",
			giveSelector: 999999999999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &RPCChainProvider{selector: tt.giveSelector}
			assert.Equal(t, tt.giveSelector, p.ChainSelector())
		})
	}
}

func Test_RPCChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when chain is nil", func(t *testing.T) {
		t.Parallel()

		p := &RPCChainProvider{
			chain: nil,
		}

		got := p.BlockChain()
		assert.Nil(t, got)
	})

	t.Run("returns chain when initialized", func(t *testing.T) {
		t.Parallel()

		chain := &stellar.Chain{
			ChainMetadata: stellar.ChainMetadata{Selector: 12345},
		}

		p := &RPCChainProvider{
			chain: chain,
		}

		got := p.BlockChain()
		assert.Equal(t, chain, got)
	})
}

func Test_NewRPCChainProvider(t *testing.T) {
	t.Parallel()

	selector := uint64(12345)
	config := RPCChainProviderConfig{
		NetworkPassphrase: "Test SDF Network ; September 2015",
		FriendbotURL:      "https://friendbot.stellar.org",
		SorobanRPCURL:     "https://soroban-testnet.stellar.org",
	}

	p := NewRPCChainProvider(selector, config)

	require.NotNil(t, p)
	assert.Equal(t, selector, p.selector)
	assert.Equal(t, config.NetworkPassphrase, p.config.NetworkPassphrase)
	assert.Equal(t, config.FriendbotURL, p.config.FriendbotURL)
	assert.Equal(t, config.SorobanRPCURL, p.config.SorobanRPCURL)
	assert.Nil(t, p.chain, "chain should not be initialized until Initialize() is called")
}
