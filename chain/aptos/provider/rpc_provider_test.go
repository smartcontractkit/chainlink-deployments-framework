package provider

import (
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
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
				RPCURL:            "http://localhost:8080",
				DeployerSignerGen: AccountGenCTFDefault(),
			},
			wantErr: "",
		},
		{
			name: "missing rpc url",
			config: RPCChainProviderConfig{
				RPCURL: "",
			},
			wantErr: "rpc url is required",
		},
		{
			name: "missing deployer signer generator",
			config: RPCChainProviderConfig{
				RPCURL:            "http://localhost:8080",
				DeployerSignerGen: nil,
			},
			wantErr: "deployer signer generator is required",
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
		wantErr      string
	}{
		{
			name:         "valid initialization",
			giveSelector: chain_selectors.APTOS_LOCALNET.Selector,
			giveConfig: RPCChainProviderConfig{
				RPCURL:            "http://localhost:8080",
				DeployerSignerGen: AccountGenPrivateKey(testPrivateKey),
			},
		},
		{
			name:         "fails config validation",
			giveSelector: chain_selectors.APTOS_LOCALNET.Selector,
			giveConfig: RPCChainProviderConfig{
				RPCURL:            "",
				DeployerSignerGen: AccountGenPrivateKey(testPrivateKey),
			},
			wantErr: "rpc url is required",
		},
		{
			name:         "fails to generate deployer account",
			giveSelector: chain_selectors.APTOS_LOCALNET.Selector,
			giveConfig: RPCChainProviderConfig{
				RPCURL:            "http://localhost:8080",
				DeployerSignerGen: AccountGenPrivateKey("invalid_private_key"),
			},
			wantErr: "failed to generate deployer account",
		},
		{
			name:         "chain id not found for selector",
			giveSelector: 999999, // Invalid selector
			giveConfig: RPCChainProviderConfig{
				RPCURL:            "http://localhost:8080",
				DeployerSignerGen: AccountGenPrivateKey(testPrivateKey),
			},
			wantErr: "failed to get chain ID from selector 999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewRPCChainProvider(tt.giveSelector, tt.giveConfig)

			got, err := p.Initialize()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, p.chain)

				gotChain, ok := got.(aptos.Chain)
				require.True(t, ok, "expected got to be of type aptos.Chain")
				assert.Equal(t, tt.giveSelector, gotChain.Selector)
				assert.NotEmpty(t, gotChain.Client)
				assert.NotEmpty(t, gotChain.DeployerSigner)
				assert.Equal(t, tt.giveConfig.RPCURL, gotChain.URL)
				assert.NotEmpty(t, gotChain.Confirm)
			}
		})
	}
}

func Test_RPCChainProvider_Name(t *testing.T) {
	t.Parallel()

	p := &RPCChainProvider{}
	assert.Equal(t, "Aptos RPC Chain Provider", p.Name())
}

func Test_RPCChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	p := &RPCChainProvider{selector: chain_selectors.APTOS_LOCALNET.Selector}
	assert.Equal(t, chain_selectors.APTOS_LOCALNET.Selector, p.ChainSelector())
}

func Test_RPCChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	chain := &aptos.Chain{}

	p := &RPCChainProvider{
		chain: chain,
	}

	assert.Equal(t, chain, p.BlockChain())
}
