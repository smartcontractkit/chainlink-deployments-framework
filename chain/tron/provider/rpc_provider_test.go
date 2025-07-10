package provider

import (
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RPCChainProviderConfig_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		giveConfigFunc func(*RPCChainProviderConfig)
		wantErr        string
	}{
		{
			name: "valid config",
		},
		{
			name:           "missing rpc url",
			giveConfigFunc: func(c *RPCChainProviderConfig) { c.RPCURL = "" },
			wantErr:        "rpc url is required",
		},
		{
			name:           "missing deployer signer generator",
			giveConfigFunc: func(c *RPCChainProviderConfig) { c.DeployerSignerGen = nil },
			wantErr:        "deployer signer generator is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// A valid configuration for the RPCChainProviderConfig
			config := RPCChainProviderConfig{
				RPCURL:            "http://localhost:8080",
				DeployerSignerGen: AccountRandom(),
			}

			if tt.giveConfigFunc != nil {
				tt.giveConfigFunc(&config)
			}

			err := config.validate()
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

	var (
		chainSelector = chain_selectors.TEST_22222222222222222222222222222222222222222222.Selector
		existingChain = &tron.Chain{}
	)

	tests := []struct {
		name              string
		giveSelector      uint64
		giveConfigFunc    func(t *testing.T) RPCChainProviderConfig
		giveExistingChain *tron.Chain // Use this to simulate an already initialized chain
		wantErr           string
	}{
		{
			name:         "valid initialization",
			giveSelector: chainSelector,
			giveConfigFunc: func(t *testing.T) RPCChainProviderConfig {
				t.Helper()

				return RPCChainProviderConfig{
					RPCURL:            "http://localhost:8080",
					DeployerSignerGen: AccountRandom(),
				}
			},
		},
		{
			name:              "returns an already initialized chain",
			giveSelector:      chainSelector,
			giveExistingChain: existingChain,
		},
		{
			name:         "fails config validation",
			giveSelector: chainSelector,
			giveConfigFunc: func(t *testing.T) RPCChainProviderConfig {
				t.Helper()

				return RPCChainProviderConfig{}
			},
			wantErr: "invalid Tron RPC config",
		},
		{
			name:         "fails to generate deployer account",
			giveSelector: chainSelector,
			giveConfigFunc: func(t *testing.T) RPCChainProviderConfig {
				t.Helper()

				return RPCChainProviderConfig{
					RPCURL:            "http://localhost:8080",
					DeployerSignerGen: AccountFromRaw(""), // Invalid private key
				}
			},
			wantErr: "failed to generate signer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var config RPCChainProviderConfig
			if tt.giveConfigFunc != nil {
				config = tt.giveConfigFunc(t)
			}

			p := NewRPCChainProvider(tt.giveSelector, config)

			if tt.giveExistingChain != nil {
				p.chain = tt.giveExistingChain
			}

			got, err := p.Initialize(t.Context())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, p.chain)

				gotChain, ok := got.(tron.Chain)
				require.True(t, ok, "expected got to be of type tron.Chain")

				// For the already initialized chain case, we can skip the rest of the checks
				if tt.giveExistingChain != nil {
					return
				}

				// Otherwise, check the fields of the chain

				assert.Equal(t, tt.giveSelector, gotChain.Selector)
				assert.NotNil(t, gotChain.Client)
				assert.Equal(t, config.RPCURL, gotChain.URL)
				assert.NotNil(t, gotChain.Keystore)
				assert.NotNil(t, gotChain.Account)
				assert.NotNil(t, gotChain.SendAndConfirm)
				assert.NotNil(t, gotChain.DeployContractAndConfirm)
				assert.NotNil(t, gotChain.TriggerContractAndConfirm)
			}
		})
	}
}

func Test_RPCChainProvider_Name(t *testing.T) {
	t.Parallel()

	p := &RPCChainProvider{}
	assert.Equal(t, "Tron RPC Chain Provider", p.Name())
}

func Test_RPCChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	p := &RPCChainProvider{selector: chain_selectors.TRON_MAINNET.Selector}
	assert.Equal(t, chain_selectors.TRON_MAINNET.Selector, p.ChainSelector())
}

func Test_RPCChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	chain := &tron.Chain{}

	p := &RPCChainProvider{
		chain: chain,
	}

	assert.Equal(t, *chain, p.BlockChain())
}
