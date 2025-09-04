package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tonchain "github.com/smartcontractkit/chainlink-deployments-framework/chain/ton"
)

// --- RPCChainProviderConfig.validate ---

func Test_RPCChainProviderConfig_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		giveConfigFunc func(*RPCChainProviderConfig)
		wantErr        string
	}{
		{
			name: "valid config (empty wallet version uses default)",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.HTTPURL = "http://localhost:8080/config.json"
				c.WSURL = "ws://localhost:8080"
				c.DeployerSignerGen = PrivateKeyRandom()
				c.WalletVersion = ""
			},
		},
		{
			name: "missing http url",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.HTTPURL = ""
				c.DeployerSignerGen = PrivateKeyRandom()
			},
			wantErr: "rpc url is required",
		},
		{
			name: "missing deployer signer generator",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.HTTPURL = "http://localhost:8080/config.json"
				c.DeployerSignerGen = nil
			},
			wantErr: "deployer signer generator is required",
		},
		{
			name: "unsupported wallet version",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.HTTPURL = "http://localhost:8080/config.json"
				c.DeployerSignerGen = PrivateKeyRandom()
				c.WalletVersion = "V9R9"
			},
			wantErr: "unsupported wallet version",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := RPCChainProviderConfig{}
			if tt.giveConfigFunc != nil {
				tt.giveConfigFunc(&cfg)
			}

			err := cfg.validate()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// --- RPCChainProvider.Initialize ---

func Test_RPCChainProvider_Initialize(t *testing.T) {
	t.Parallel()

	var (
		selector      uint64 = 16423721717087811551
		existingChain        = &tonchain.Chain{}
	)

	tests := []struct {
		name              string
		giveSelector      uint64
		giveConfigFunc    func(t *testing.T) RPCChainProviderConfig
		giveExistingChain *tonchain.Chain
		wantErr           string
	}{
		{
			name:              "returns an already initialized chain",
			giveSelector:      selector,
			giveExistingChain: existingChain,
		},
		{
			name:         "fails config validation (missing url & keygen)",
			giveSelector: selector,
			giveConfigFunc: func(t *testing.T) RPCChainProviderConfig {
				t.Helper()
				return RPCChainProviderConfig{} // invalid
			},
			wantErr: "rpc url is required",
		},
		{
			name:         "fails to retrieve ton network config (bad URL)",
			giveSelector: selector,
			giveConfigFunc: func(t *testing.T) RPCChainProviderConfig {
				t.Helper()
				return RPCChainProviderConfig{
					HTTPURL:           "http://127.0.0.1:0/not-a-config.json",
					WSURL:             "",
					DeployerSignerGen: PrivateKeyRandom(),
					WalletVersion:     "",
				}
			},
			wantErr: "failed to retrieve ton network config",
		},
		{
			name:         "fails to generate private key",
			giveSelector: selector,
			giveConfigFunc: func(t *testing.T) RPCChainProviderConfig {
				t.Helper()
				return RPCChainProviderConfig{
					HTTPURL:           "https://ton-blockchain.github.io/testnet-global.config.json", // will fail before keygen if URL is good; keep bad URL to avoid network
					DeployerSignerGen: PrivateKeyFromRaw("invalid-key"),
				}
			},
			wantErr: "failed to parse private key",
		},
		{
			name:         "everything ok",
			giveSelector: selector,
			giveConfigFunc: func(t *testing.T) RPCChainProviderConfig {
				t.Helper()
				return RPCChainProviderConfig{
					HTTPURL:           "https://ton-blockchain.github.io/testnet-global.config.json",
					DeployerSignerGen: PrivateKeyFromRaw("0b1f7dbb19112fdac53344cf49731e41bfc420ac6a71d38c89fb38d04a6563d99aa3d1fa430550e8de5171ec55453b4e048c1701cadfa56726d489c56d67bab3"),
				}
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var cfg RPCChainProviderConfig
			if tt.giveConfigFunc != nil {
				cfg = tt.giveConfigFunc(t)
			}

			p := NewRPCChainProvider(tt.giveSelector, cfg)

			if tt.giveExistingChain != nil {
				p.chain = tt.giveExistingChain
			}

			got, err := p.Initialize(t.Context())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, p.chain)

			gotChain, ok := got.(tonchain.Chain)
			require.True(t, ok, "expected got to be of type ton.Chain")

			// For the already initialized chain case, we can skip additional checks.
			if tt.giveExistingChain != nil {
				return
			}

			assert.Equal(t, tt.giveSelector, gotChain.Selector)
			assert.Equal(t, cfg.HTTPURL, gotChain.URL)
			assert.NotNil(t, gotChain.Client)
			assert.NotNil(t, gotChain.Wallet)
			assert.NotNil(t, gotChain.WalletAddress)
		})
	}
}

// --- Accessors ---

func Test_RPCChainProvider_Name(t *testing.T) {
	t.Parallel()

	p := &RPCChainProvider{}
	assert.Equal(t, "TON RPC Chain Provider", p.Name())
}

func Test_RPCChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	p := &RPCChainProvider{selector: 12345}
	assert.Equal(t, uint64(12345), p.ChainSelector())
}

func Test_RPCChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	c := &tonchain.Chain{}
	p := &RPCChainProvider{chain: c}

	assert.Equal(t, *c, p.BlockChain())
}
