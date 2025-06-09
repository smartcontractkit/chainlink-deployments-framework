package provider

import (
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
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
				DeployerSignerGen: AccountGenCTFDefault(),
				Once:              testutils.DefaultNetworkOnce,
			},
			wantErr: "",
		},
		{
			name: "missing deployer signer generator",
			config: CTFChainProviderConfig{
				DeployerSignerGen: nil,
				Once:              testutils.DefaultNetworkOnce,
			},
			wantErr: "deployer signer generator is required",
		},
		{
			name: "missing sync.Once instance",
			config: CTFChainProviderConfig{
				DeployerSignerGen: AccountGenCTFDefault(),
				Once:              nil,
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

	tests := []struct {
		name         string
		giveSelector uint64
		giveConfig   CTFChainProviderConfig
		wantErr      string
	}{
		{
			name:         "valid initialization",
			giveSelector: chain_selectors.APTOS_LOCALNET.Selector,
			giveConfig: CTFChainProviderConfig{
				DeployerSignerGen: AccountGenCTFDefault(),
				Once:              testutils.DefaultNetworkOnce,
			},
		},
		{
			name:         "fails config validation",
			giveSelector: chain_selectors.APTOS_LOCALNET.Selector,
			giveConfig: CTFChainProviderConfig{
				Once: testutils.DefaultNetworkOnce,
			},
			wantErr: "deployer signer generator is required",
		},
		{
			name:         "fails to generate deployer account",
			giveSelector: chain_selectors.APTOS_LOCALNET.Selector,
			giveConfig: CTFChainProviderConfig{
				DeployerSignerGen: AccountGenPrivateKey("invalid_private_key"),
				Once:              testutils.DefaultNetworkOnce,
			},
			wantErr: "failed to generate deployer account",
		},
		{
			name:         "chain id not found for selector",
			giveSelector: 999999, // Invalid selector
			giveConfig: CTFChainProviderConfig{
				DeployerSignerGen: AccountGenCTFDefault(),
				Once:              testutils.DefaultNetworkOnce,
			},
			wantErr: "failed to get chain ID from selector 999999",
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

				// Check that the chain is of type *aptos.Chain and has the expected fields
				gotChain, ok := got.(aptos.Chain)
				require.True(t, ok, "expected got to be of type aptos.Chain")
				assert.Equal(t, tt.giveSelector, gotChain.Selector)
				assert.NotEmpty(t, gotChain.Client)
				assert.NotEmpty(t, gotChain.DeployerSigner)
				assert.NotEmpty(t, gotChain.URL)
				assert.NotEmpty(t, gotChain.Confirm)
			}
		})
	}
}

func Test_CTFChainProvider_Name(t *testing.T) {
	t.Parallel()

	p := &CTFChainProvider{}
	assert.Equal(t, "Aptos CTF Chain Provider", p.Name())
}

func Test_CTFChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	p := &CTFChainProvider{selector: chain_selectors.APTOS_LOCALNET.Selector}
	assert.Equal(t, chain_selectors.APTOS_LOCALNET.Selector, p.ChainSelector())
}

func Test_CTFChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	chain := &aptos.Chain{}

	p := &CTFChainProvider{
		chain: chain,
	}

	assert.Equal(t, chain, p.BlockChain())
}
