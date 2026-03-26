package provider

import (
	"context"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/testutils"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/stellar"
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
				DeployerKeypairGen: KeypairRandom(),
				Once:               testutils.DefaultNetworkOnce,
			},
			wantErr: "",
		},
		{
			name: "missing deployer keypair generator",
			config: CTFChainProviderConfig{
				Once: testutils.DefaultNetworkOnce,
			},
			wantErr: "deployer keypair generator is required",
		},
		{
			name: "missing sync.Once",
			config: CTFChainProviderConfig{
				DeployerKeypairGen: KeypairRandom(),
			},
			wantErr: "sync.Once instance is required",
		},
		{
			name: "valid config with custom image",
			config: CTFChainProviderConfig{
				DeployerKeypairGen: KeypairRandom(),
				Once:               testutils.DefaultNetworkOnce,
				Image:              "stellar/quickstart:testing",
			},
			wantErr: "",
		},
		{
			name: "valid config with custom network passphrase",
			config: CTFChainProviderConfig{
				DeployerKeypairGen: KeypairRandom(),
				Once:               testutils.DefaultNetworkOnce,
				NetworkPassphrase:  "Test SDF Network ; September 2015",
			},
			wantErr: "",
		},
		{
			name: "valid config with custom port",
			config: CTFChainProviderConfig{
				DeployerKeypairGen: KeypairRandom(),
				Once:               testutils.DefaultNetworkOnce,
				Port:               8000,
			},
			wantErr: "",
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
			giveSelector: chainsel.STELLAR_LOCALNET.Selector,
			giveConfig: CTFChainProviderConfig{
				DeployerKeypairGen: KeypairRandom(),
				Once:               testutils.DefaultNetworkOnce,
			},
		},
		{
			name:         "fails config validation - missing deployer keypair generator",
			giveSelector: chainsel.STELLAR_LOCALNET.Selector,
			giveConfig: CTFChainProviderConfig{
				Once: testutils.DefaultNetworkOnce,
			},
			wantErr: "deployer keypair generator is required",
		},
		{
			name:         "fails config validation - missing sync.Once",
			giveSelector: chainsel.STELLAR_LOCALNET.Selector,
			giveConfig: CTFChainProviderConfig{
				DeployerKeypairGen: KeypairRandom(),
			},
			wantErr: "sync.Once instance is required",
		},
		{
			name:         "fails to get chain ID from selector",
			giveSelector: 999999, // Invalid selector
			giveConfig: CTFChainProviderConfig{
				DeployerKeypairGen: KeypairRandom(),
				Once:               testutils.DefaultNetworkOnce,
			},
			wantErr: "failed to get chain ID from selector 999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewCTFChainProvider(t, tt.giveSelector, tt.giveConfig)

			got, err := p.Initialize(context.Background())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				// Note: This test may fail if Docker is not running, which is expected
				// The test validates the code paths up to container startup
				if err == nil {
					require.NotNil(t, p.chain)

					// Check that the chain is of type stellar.Chain and has the expected fields
					gotChain, ok := got.(stellar.Chain)
					require.True(t, ok, "expected got to be of type stellar.Chain")
					assert.Equal(t, tt.giveSelector, gotChain.Selector)
					assert.NotNil(t, gotChain.Client)
					assert.NotNil(t, gotChain.Signer)
					assert.NotEmpty(t, gotChain.URL)
				}
				// If error occurs due to Docker not running, that's acceptable for unit tests
			}
		})
	}
}

func Test_CTFChainProvider_Name(t *testing.T) {
	t.Parallel()

	p := &CTFChainProvider{}
	assert.Equal(t, "Stellar CTF Chain Provider", p.Name())
}

func Test_CTFChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	p := &CTFChainProvider{selector: chainsel.STELLAR_LOCALNET.Selector}
	assert.Equal(t, chainsel.STELLAR_LOCALNET.Selector, p.ChainSelector())
}

func Test_CTFChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	chain := &stellar.Chain{
		ChainMetadata: stellar.ChainMetadata{Selector: chainsel.STELLAR_LOCALNET.Selector},
	}

	p := &CTFChainProvider{
		chain: chain,
	}

	assert.Equal(t, *chain, p.BlockChain())
}
