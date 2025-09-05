package provider

import (
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/testutils"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
)

func Test_CTFChainProviderConfig_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		giveConfigFunc func(*CTFChainProviderConfig)
		wantErr        string
	}{
		{
			name: "valid config",
		},
		{
			name:           "missing deployer key generator",
			giveConfigFunc: func(c *CTFChainProviderConfig) { c.DeployerKeyGen = nil },
			wantErr:        "deployer key generator is required",
		},
		{
			name:           "missing programs path",
			giveConfigFunc: func(c *CTFChainProviderConfig) { c.ProgramsPath = "" },
			wantErr:        "programs path is required",
		},
		{
			name:           "missing program IDs",
			giveConfigFunc: func(c *CTFChainProviderConfig) { c.ProgramIDs = nil },
			wantErr:        "program ids is required",
		},
		{
			name: "programs path does not exist",
			giveConfigFunc: func(c *CTFChainProviderConfig) {
				c.ProgramsPath = "invalid/path/to/programs"
			},
			wantErr: "required file does not exist: invalid/path/to/programs",
		},
		{
			name: "programs path is not absolute",
			giveConfigFunc: func(c *CTFChainProviderConfig) {
				c.ProgramsPath = "."
			},
			wantErr: "required file is not absolute: .",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// A valid configuration for the CTFChainProviderConfig
			config := CTFChainProviderConfig{
				DeployerKeyGen: PrivateKeyRandom(),
				ProgramsPath:   t.TempDir(),
				ProgramIDs:     map[string]string{},
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

func Test_CTFChainProvider_Initialize(t *testing.T) {
	t.Parallel()

	var (
		chainSelector = chainsel.TEST_22222222222222222222222222222222222222222222.Selector
		existingChain = &solana.Chain{}
	)

	tests := []struct {
		name              string
		giveSelector      uint64
		giveConfig        CTFChainProviderConfig
		giveExistingChain *solana.Chain // Use this to simulate an already initialized chain
		wantErr           string
	}{
		{
			name:         "valid initialization",
			giveSelector: chainSelector,
			giveConfig: CTFChainProviderConfig{
				DeployerKeyGen: PrivateKeyRandom(),
				ProgramsPath:   t.TempDir(),
				ProgramIDs:     map[string]string{},
				Once:           testutils.DefaultNetworkOnce,
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
			giveConfig:   CTFChainProviderConfig{},
			wantErr:      "deployer key generator is required",
		},
		{
			name:         "chain id not found for selector",
			giveSelector: 999999, // Invalid selector
			giveConfig: CTFChainProviderConfig{
				DeployerKeyGen: PrivateKeyRandom(),
				ProgramsPath:   t.TempDir(),
				ProgramIDs:     map[string]string{},
				Once:           testutils.DefaultNetworkOnce,
			},
			wantErr: "failed to get chain ID from selector 999999",
		},
		{
			name:         "fails to generate deployer account",
			giveSelector: chainSelector,
			giveConfig: CTFChainProviderConfig{
				DeployerKeyGen: PrivateKeyFromRaw("invalid_private_key"),
				ProgramsPath:   t.TempDir(),
				ProgramIDs:     map[string]string{},
				Once:           testutils.DefaultNetworkOnce,
			},
			wantErr: "failed to generate deployer keypair",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewCTFChainProvider(t, tt.giveSelector, tt.giveConfig)

			if tt.giveExistingChain != nil {
				p.chain = tt.giveExistingChain // Simulate an already initialized chain
			}

			got, err := p.Initialize(t.Context())

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, p.chain)

				// Check that the chain is of type *solana.Chain and has the expected fields
				gotChain, ok := got.(solana.Chain)
				require.True(t, ok, "expected got to be of type solana.Chain")

				// For the already initialized chain case, we can skip the rest of the checks
				if tt.giveExistingChain != nil {
					return
				}

				// Otherwise, check the fields of the chain
				assert.Equal(t, tt.giveSelector, gotChain.Selector)
				assert.NotNil(t, gotChain.Client)
				assert.NotEmpty(t, gotChain.URL)
				assert.NotEmpty(t, gotChain.WSURL)
				assert.NotNil(t, gotChain.DeployerKey)
				assert.NotEmpty(t, gotChain.ProgramsPath)
				assert.NotEmpty(t, gotChain.KeypairPath)
				assert.NotNil(t, gotChain.SendAndConfirm)
				assert.NotNil(t, gotChain.Confirm)
			}
		})
	}
}

func Test_CTFChainProvider_Name(t *testing.T) {
	t.Parallel()

	p := &CTFChainProvider{}
	assert.Equal(t, "Solana CTF Chain Provider", p.Name())
}

func Test_CTFChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	p := &CTFChainProvider{selector: chainsel.SOLANA_DEVNET.Selector}
	assert.Equal(t, chainsel.SOLANA_DEVNET.Selector, p.ChainSelector())
}

func Test_CTFChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	chain := &solana.Chain{}

	p := &CTFChainProvider{
		chain: chain,
	}

	assert.Equal(t, *chain, p.BlockChain())
}
