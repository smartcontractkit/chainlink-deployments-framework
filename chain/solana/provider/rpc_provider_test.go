package provider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	sollib "github.com/gagliardetto/solana-go"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
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
			name:           "missing http url",
			giveConfigFunc: func(c *RPCChainProviderConfig) { c.HTTPURL = "" },
			wantErr:        "http url is required",
		},
		{
			name:           "missing deployer key generator",
			giveConfigFunc: func(c *RPCChainProviderConfig) { c.DeployerKeyGen = nil },
			wantErr:        "deployer key generator or deployer signer generator is required",
		},
		{
			name: "signer generator without deployer key generator",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.DeployerKeyGen = nil
				c.DeployerSignerGen = &staticSignerGen{}
			},
		},
		{
			name:           "missing programs path",
			giveConfigFunc: func(c *RPCChainProviderConfig) { c.ProgramsPath = "" },
			wantErr:        "programs path is required",
		},
		{
			name:           "missing keypair path",
			giveConfigFunc: func(c *RPCChainProviderConfig) { c.KeypairDirPath = "" },
			wantErr:        "keypair path is required",
		},
		{
			name: "programs path does not exist",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.ProgramsPath = "invalid/path/to/programs"
			},
			wantErr: "required file does not exist: invalid/path/to/programs",
		},
		{
			name: "programs path is not absolute",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.ProgramsPath = "."
			},
			wantErr: "required file is not absolute: .",
		},
		{
			name: "keypair path does not exist",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.KeypairDirPath = "invalid/path/to/keypair"
			},
			wantErr: "required file does not exist: invalid/path/to/keypair",
		},
		{
			name: "keypair path is not absolute",
			giveConfigFunc: func(c *RPCChainProviderConfig) {
				c.KeypairDirPath = "."
			},
			wantErr: "required file is not absolute: .",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// A valid configuration for the RPCChainProviderConfig
			config := RPCChainProviderConfig{
				HTTPURL:        "http://localhost:8080",
				WSURL:          "ws://localhost:8080",
				DeployerKeyGen: PrivateKeyRandom(),
				ProgramsPath:   t.TempDir(),
				KeypairDirPath: t.TempDir(),
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
		chainSelector = chainsel.TEST_22222222222222222222222222222222222222222222.Selector
		existingChain = &solana.Chain{}
	)

	tests := []struct {
		name              string
		giveSelector      uint64
		giveConfigFunc    func(t *testing.T, programsPath, keypairPath string) RPCChainProviderConfig
		giveExistingChain *solana.Chain // Use this to simulate an already initialized chain
		wantErr           string
	}{
		{
			name:         "valid initialization",
			giveSelector: chainSelector,
			giveConfigFunc: func(t *testing.T, programsPath, keypairDirPath string) RPCChainProviderConfig {
				t.Helper()

				return RPCChainProviderConfig{
					HTTPURL:        "http://localhost:8080",
					WSURL:          "ws://localhost:8080",
					DeployerKeyGen: PrivateKeyRandom(),
					ProgramsPath:   programsPath,
					KeypairDirPath: keypairDirPath,
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
			giveConfigFunc: func(t *testing.T, programsPath, keypairDirPath string) RPCChainProviderConfig {
				t.Helper()

				return RPCChainProviderConfig{}
			},
			wantErr: "http url is required",
		},
		{
			name:         "fails to generate deployer account",
			giveSelector: chainSelector,
			giveConfigFunc: func(t *testing.T, programsPath, keypairDirPath string) RPCChainProviderConfig {
				t.Helper()

				return RPCChainProviderConfig{
					HTTPURL:        "http://localhost:8080",
					WSURL:          "ws://localhost:8080",
					DeployerKeyGen: PrivateKeyFromRaw(""), // Invalid private key
					ProgramsPath:   programsPath,
					KeypairDirPath: keypairDirPath,
				}
			},
			wantErr: "failed to generate deployer keypair",
		},
		{
			name:         "failed to write keypair to file",
			giveSelector: 999999, // Invalid selector
			giveConfigFunc: func(t *testing.T, programsPath, keypairDirPath string) RPCChainProviderConfig {
				t.Helper()

				// Create a directory with read-only permissions to simulate a write failure
				readonlydir := filepath.Join(keypairDirPath, "readonlydir")
				err := os.Mkdir(readonlydir, 0400)
				require.NoError(t, err)

				return RPCChainProviderConfig{
					HTTPURL:        "http://localhost:8080",
					WSURL:          "ws://localhost:8080",
					DeployerKeyGen: PrivateKeyRandom(),
					ProgramsPath:   programsPath,
					KeypairDirPath: readonlydir,
				}
			},
			wantErr: "failed to write deployer keypair to file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				programsPath   = t.TempDir()
				keypairDirPath = t.TempDir()
			)

			var config RPCChainProviderConfig
			if tt.giveConfigFunc != nil {
				config = tt.giveConfigFunc(t, programsPath, keypairDirPath)
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

				gotChain, ok := got.(solana.Chain)
				require.True(t, ok, "expected got to be of type solana.Chain")

				// For the already initialized chain case, we can skip the rest of the checks
				if tt.giveExistingChain != nil {
					return
				}

				// Otherwise, check the fields of the chain

				assert.Equal(t, tt.giveSelector, gotChain.Selector)
				assert.NotNil(t, gotChain.Client)
				assert.Equal(t, config.HTTPURL, gotChain.URL)
				assert.Equal(t, config.WSURL, gotChain.WSURL)
				assert.NotNil(t, gotChain.DeployerKey)
				assert.Equal(t, programsPath, gotChain.ProgramsPath)
				assert.Equal(t,
					filepath.Join(keypairDirPath, "authority-keypair.json"),
					gotChain.KeypairPath,
				)
				assert.NotNil(t, gotChain.SendAndConfirm)
				assert.NotNil(t, gotChain.Confirm)
			}
		})
	}
}

// staticSignerGen is a SignerGenerator that returns a fixed signer or error.
type staticSignerGen struct {
	signer solana.Signer
	err    error
}

func (g *staticSignerGen) Generate(context.Context) (solana.Signer, error) {
	return g.signer, g.err
}

func Test_RPCChainProvider_Initialize_SignerGen(t *testing.T) {
	t.Parallel()

	chainSelector := chainsel.TEST_22222222222222222222222222222222222222222222.Selector
	signer := solana.NewPrivateKeySigner(sollib.NewWallet().PrivateKey)

	tests := []struct {
		name            string
		giveKeyGen      PrivateKeyGenerator
		giveSignerGen   *staticSignerGen
		wantKeypairFile bool
		wantErr         string
	}{
		{
			name:            "signer with CLI keypair",
			giveKeyGen:      PrivateKeyRandom(),
			giveSignerGen:   &staticSignerGen{signer: signer},
			wantKeypairFile: true,
		},
		{
			name:          "signer without CLI keypair",
			giveSignerGen: &staticSignerGen{signer: signer},
		},
		{
			name:          "fails to generate signer",
			giveSignerGen: &staticSignerGen{err: errors.New("kms unavailable")},
			wantErr:       "failed to generate deployer signer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			keypairDirPath := t.TempDir()

			got, err := NewRPCChainProvider(chainSelector, RPCChainProviderConfig{
				HTTPURL:           "http://localhost:8080",
				DeployerKeyGen:    tt.giveKeyGen,
				DeployerSignerGen: tt.giveSignerGen,
				ProgramsPath:      t.TempDir(),
				KeypairDirPath:    keypairDirPath,
			}).Initialize(t.Context())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)

			gotChain, ok := got.(solana.Chain)
			require.True(t, ok, "expected got to be of type solana.Chain")

			// The deployer signs only through the signer; the CLI keypair is never exposed as
			// the deployer key.
			assert.Nil(t, gotChain.DeployerKey)
			assert.Equal(t, signer, gotChain.DeployerSigner)
			assert.Equal(t, signer, gotChain.Deployer())

			if tt.wantKeypairFile {
				assert.Equal(t, filepath.Join(keypairDirPath, "authority-keypair.json"), gotChain.KeypairPath)
				assert.FileExists(t, gotChain.KeypairPath)
			} else {
				assert.Empty(t, gotChain.KeypairPath)
			}
		})
	}
}

func Test_RPCChainProvider_Name(t *testing.T) {
	t.Parallel()

	p := &RPCChainProvider{}
	assert.Equal(t, "Solana RPC Chain Provider", p.Name())
}

func Test_RPCChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	p := &RPCChainProvider{selector: chainsel.SOLANA_DEVNET.Selector}
	assert.Equal(t, chainsel.SOLANA_DEVNET.Selector, p.ChainSelector())
}

func Test_RPCChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	chain := &solana.Chain{}

	p := &RPCChainProvider{
		chain: chain,
	}

	assert.Equal(t, *chain, p.BlockChain())
}
