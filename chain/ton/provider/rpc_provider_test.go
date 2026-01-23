package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/ton/wallet"

	tonchain "github.com/smartcontractkit/chainlink-deployments-framework/chain/ton"
)

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
				c.HTTPURL = "liteserver://publickey@localhost:8080"
				c.WSURL = "ws://localhost:8080"
				c.DeployerSignerGen = PrivateKeyRandom()
				c.WalletVersion = ""
			},
		},
		{
			name: "valid config with V3R2 wallet version",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.HTTPURL = "liteserver://publickey@localhost:8080"
				c.DeployerSignerGen = PrivateKeyRandom()
				c.WalletVersion = WalletVersionV3R2
			},
		},
		{
			name: "valid config with V4R2 wallet version",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.HTTPURL = "liteserver://publickey@localhost:8080"
				c.DeployerSignerGen = PrivateKeyRandom()
				c.WalletVersion = WalletVersionV4R2
			},
		},
		{
			name: "valid config with V5R1 wallet version",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.HTTPURL = "liteserver://publickey@localhost:8080"
				c.DeployerSignerGen = PrivateKeyRandom()
				c.WalletVersion = WalletVersionV5R1
			},
		},
		{
			name: "missing liteserver url",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.HTTPURL = ""
				c.DeployerSignerGen = PrivateKeyRandom()
			},
			wantErr: "liteserver url is required",
		},
		{
			name: "invalid liteserver url prefix",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.HTTPURL = "http://example.com"
				c.DeployerSignerGen = PrivateKeyRandom()
			},
			wantErr: "invalid liteserver URL format: expected liteserver:// prefix",
		},
		{
			name: "invalid liteserver url missing @ separator",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.HTTPURL = "liteserver://nohostport"
				c.DeployerSignerGen = PrivateKeyRandom()
			},
			wantErr: "invalid liteserver URL format: expected publickey@host:port",
		},
		{
			name: "invalid liteserver url empty public key",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.HTTPURL = "liteserver://@localhost:8080"
				c.DeployerSignerGen = PrivateKeyRandom()
			},
			wantErr: "invalid liteserver URL format: public key cannot be empty",
		},
		{
			name: "invalid liteserver url empty host:port",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.HTTPURL = "liteserver://publickey@"
				c.DeployerSignerGen = PrivateKeyRandom()
			},
			wantErr: "invalid liteserver URL format: host:port cannot be empty",
		},
		{
			name: "missing deployer signer generator",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.HTTPURL = "liteserver://publickey@localhost:8080"
				c.DeployerSignerGen = nil
			},
			wantErr: "deployer signer generator is required",
		},
		{
			name: "unsupported wallet version",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.HTTPURL = "liteserver://publickey@localhost:8080"
				c.DeployerSignerGen = PrivateKeyRandom()
				c.WalletVersion = "V9R9"
			},
			wantErr: "unsupported wallet version",
		},
	}

	for _, tt := range tests {
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

func Test_RPCChainProvider_Initialize(t *testing.T) {
	t.Parallel()

	existingChain := &tonchain.Chain{ChainMetadata: tonchain.ChainMetadata{Selector: 123}}
	p := &RPCChainProvider{
		selector: 123,
		chain:    existingChain,
	}

	got, err := p.Initialize(t.Context())
	require.NoError(t, err)

	gotChain, ok := got.(tonchain.Chain)
	require.True(t, ok)
	assert.Equal(t, existingChain.Selector, gotChain.Selector)
}

func Test_RPCChainProvider_Initialize_InvalidConfig(t *testing.T) {
	t.Parallel()

	p := &RPCChainProvider{
		selector: 123,
		config: RPCChainProviderConfig{
			HTTPURL:           "", // invalid - missing URL
			DeployerSignerGen: PrivateKeyRandom(),
		},
	}

	_, err := p.Initialize(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate provider config")
}

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

// Unit tests for extracted functions

func Test_getWalletVersionConfig(t *testing.T) {
	t.Parallel()

	t.Run("V3R2 returns wallet.V3R2", func(t *testing.T) {
		t.Parallel()
		cfg, err := getWalletVersionConfig(WalletVersionV3R2)
		require.NoError(t, err)
		assert.Equal(t, wallet.V3R2, cfg)
	})

	t.Run("V4R2 returns wallet.V4R2", func(t *testing.T) {
		t.Parallel()
		cfg, err := getWalletVersionConfig(WalletVersionV4R2)
		require.NoError(t, err)
		assert.Equal(t, wallet.V4R2, cfg)
	})

	t.Run("V5R1 returns ConfigV5R1Final", func(t *testing.T) {
		t.Parallel()
		cfg, err := getWalletVersionConfig(WalletVersionV5R1)
		require.NoError(t, err)
		v5Config, ok := cfg.(wallet.ConfigV5R1Final)
		require.True(t, ok, "expected ConfigV5R1Final type")
		assert.Equal(t, int32(wallet.MainnetGlobalID), v5Config.NetworkGlobalID)
		assert.Equal(t, int8(0), v5Config.Workchain)
	})

	t.Run("Default returns ConfigV5R1Final", func(t *testing.T) {
		t.Parallel()
		cfg, err := getWalletVersionConfig(WalletVersionDefault)
		require.NoError(t, err)
		v5Config, ok := cfg.(wallet.ConfigV5R1Final)
		require.True(t, ok, "expected ConfigV5R1Final type")
		assert.Equal(t, int32(wallet.MainnetGlobalID), v5Config.NetworkGlobalID)
		assert.Equal(t, int8(0), v5Config.Workchain)
	})

	t.Run("Unsupported version returns error", func(t *testing.T) {
		t.Parallel()
		cfg, err := getWalletVersionConfig("V9R9")
		require.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "unsupported wallet version")
	})
}

func Test_NewRPCChainProvider(t *testing.T) {
	t.Parallel()

	selector := uint64(12345)
	config := RPCChainProviderConfig{
		HTTPURL:           "liteserver://publickey@localhost:8080",
		DeployerSignerGen: PrivateKeyRandom(),
		WalletVersion:     WalletVersionV5R1,
	}

	p := NewRPCChainProvider(selector, config)

	require.NotNil(t, p)
	assert.Equal(t, selector, p.selector)
	assert.Equal(t, config.HTTPURL, p.config.HTTPURL)
	assert.Equal(t, config.WalletVersion, p.config.WalletVersion)
	assert.Nil(t, p.chain)
}

func Test_buildChain(t *testing.T) {
	t.Parallel()

	// Create a test wallet using a fixed private key
	privateKey := make([]byte, 32)
	for i := range privateKey {
		privateKey[i] = byte(i)
	}
	testWallet, err := wallet.FromPrivateKeyWithOptions(nil, privateKey, wallet.V4R2, wallet.WithWorkchain(0))
	require.NoError(t, err)

	selector := uint64(789)
	httpURL := "liteserver://publickey@localhost:8080"

	chain := buildChain(selector, nil, testWallet, httpURL)

	require.NotNil(t, chain)
	assert.Equal(t, selector, chain.Selector)
	assert.Equal(t, httpURL, chain.URL)
	assert.Nil(t, chain.Client)
	assert.Equal(t, testWallet, chain.Wallet)
	assert.Equal(t, testWallet.WalletAddress(), chain.WalletAddress)

	assert.Equal(t, defaultAmountTonString, chain.Amount.String())
}

func Test_WalletVersionConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, WalletVersionV3R2, WalletVersion("V3R2"))
	assert.Equal(t, WalletVersionV4R2, WalletVersion("V4R2"))
	assert.Equal(t, WalletVersionV5R1, WalletVersion("V5R1"))
	assert.Equal(t, WalletVersionDefault, WalletVersion(""))
}

func Test_RPCChainProvider_Initialize_FailedPrivateKeyGeneration(t *testing.T) {
	t.Parallel()

	p := &RPCChainProvider{
		selector: 123,
		config: RPCChainProviderConfig{
			HTTPURL:           "liteserver://publickey@localhost:8080",
			DeployerSignerGen: PrivateKeyFromRaw("invalid-hex"), // invalid hex will cause generation to fail
			WalletVersion:     WalletVersionV5R1,
		},
	}

	_, err := p.Initialize(t.Context())
	require.Error(t, err)
}

func Test_createWallet(t *testing.T) {
	t.Parallel()

	t.Run("success with valid private key and version", func(t *testing.T) {
		t.Parallel()

		privateKey := make([]byte, 32)
		for i := range privateKey {
			privateKey[i] = byte(i)
		}

		w, err := createWallet(nil, privateKey, WalletVersionV4R2)
		require.NoError(t, err)
		require.NotNil(t, w)
	})

	t.Run("error with unsupported wallet version", func(t *testing.T) {
		t.Parallel()

		privateKey := make([]byte, 32)

		w, err := createWallet(nil, privateKey, "UNSUPPORTED")
		require.Error(t, err)
		assert.Nil(t, w)
		assert.Contains(t, err.Error(), "unsupported wallet version")
	})

	t.Run("success with default wallet version", func(t *testing.T) {
		t.Parallel()

		privateKey := make([]byte, 32)
		for i := range privateKey {
			privateKey[i] = byte(i + 10)
		}

		w, err := createWallet(nil, privateKey, WalletVersionDefault)
		require.NoError(t, err)
		require.NotNil(t, w)
	})
}

func Test_setupConnection_invalidURL(t *testing.T) {
	t.Parallel()

	// Test that setupConnection fails with invalid URL
	_, err := setupConnection(t.Context(), "invalid-url")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to liteserver")
}
