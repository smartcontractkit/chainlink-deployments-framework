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
			name: "missing liteserver url",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.HTTPURL = ""
				c.DeployerSignerGen = PrivateKeyRandom()
			},
			wantErr: "liteserver url is required",
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

func Test_validateLiteserverURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		wantErr string
	}{
		{
			name: "valid URL",
			url:  "liteserver://validkey@localhost:8080",
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: "liteserver url is required",
		},
		{
			name:    "invalid prefix",
			url:     "http://example.com",
			wantErr: "invalid liteserver URL format: expected liteserver:// prefix",
		},
		{
			name:    "missing @",
			url:     "liteserver://invalidurl",
			wantErr: "invalid liteserver URL format: expected publickey@host:port",
		},
		{
			name:    "multiple @",
			url:     "liteserver://key1@key2@host:port",
			wantErr: "invalid liteserver URL format: expected publickey@host:port",
		},
		{
			name:    "empty public key",
			url:     "liteserver://@localhost:8080",
			wantErr: "invalid liteserver URL format: public key cannot be empty",
		},
		{
			name:    "empty host:port",
			url:     "liteserver://validkey@",
			wantErr: "invalid liteserver URL format: host:port cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateLiteserverURL(tt.url)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_getWalletVersionConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version WalletVersion
		wantErr bool
	}{
		{
			name:    "V3R2",
			version: WalletVersionV3R2,
			wantErr: false,
		},
		{
			name:    "V4R2",
			version: WalletVersionV4R2,
			wantErr: false,
		},
		{
			name:    "V5R1",
			version: WalletVersionV5R1,
			wantErr: false,
		},
		{
			name:    "Default (empty)",
			version: WalletVersionDefault,
			wantErr: false,
		},
		{
			name:    "Unsupported version",
			version: "V9R9",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := getWalletVersionConfig(tt.version)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
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

func Test_createWallet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		version       WalletVersion
		expectError   bool
		errorContains string
	}{
		{
			name:          "unsupported wallet version returns error",
			version:       "V9R9",
			expectError:   true,
			errorContains: "unsupported wallet version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			privateKey := make([]byte, 32)
			wallet, err := createWallet(nil, privateKey, tt.version)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, wallet)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, wallet)
			}
		})
	}
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

	assert.Equal(t, defaultAmountTonString, chain.Amount)
	assert.Equal(t, testWallet, chain.Wallet)
}

func Test_createLiteclientConnectionPool_InvalidURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		url           string
		errorContains string
	}{
		{
			name:          "empty URL returns error",
			url:           "",
			errorContains: "liteserver url is required",
		},
		{
			name:          "invalid prefix returns error",
			url:           "http://example.com",
			errorContains: "invalid liteserver URL format",
		},
		{
			name:          "missing @ returns error",
			url:           "liteserver://invalidurl",
			errorContains: "invalid liteserver URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pool, err := createLiteclientConnectionPool(t.Context(), tt.url)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorContains)
			assert.Nil(t, pool)
		})
	}
}
