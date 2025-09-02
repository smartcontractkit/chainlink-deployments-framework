package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	config_env "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
)

func Test_LoadEnvConfig(t *testing.T) { //nolint:paralleltest // These tests are not parallel safe due to setting of env vars
	tests := []struct {
		name     string
		skipCI   bool                                       // Option to skip this test in CI because loading from file requires the 'CI' env var to not be set, but it is always set when running the test in CI.
		envvars  map[string]string                          // Environment variables to set
		wantFunc func(t *testing.T, cfg *config_env.Config) // Function to validate the config
		wantErr  string
	}{
		{
			name:   "Load networks and config with new config enabled (loads from file)",
			skipCI: true,
			wantFunc: func(t *testing.T, cfg *config_env.Config) {
				t.Helper()

				require.NotNil(t, cfg)

				// Validate environment configuration
				assert.Equal(t, "0xabc", cfg.Onchain.EVM.DeployerKey)
				assert.Equal(t, "af1a2b3c", cfg.Offchain.JobDistributor.Auth.CognitoAppClientID)
				assert.Equal(t, "11111111", cfg.Offchain.JobDistributor.Auth.CognitoAppClientSecret)
				assert.Equal(t, "us-west-1", cfg.Offchain.JobDistributor.Auth.AWSRegion)
				assert.Equal(t, "testuser", cfg.Offchain.JobDistributor.Auth.Username)
				assert.Equal(t, "testpassword", cfg.Offchain.JobDistributor.Auth.Password)
				assert.Equal(t, "test signers phrase", cfg.Offchain.OCR.XSigners)
				assert.Equal(t, "test proposers phrase", cfg.Offchain.OCR.XProposers)
				assert.Equal(t, "http://localhost:1000", cfg.Catalog.GRPC)
				assert.Equal(t, "0xbcd", cfg.Onchain.Solana.WalletKey)
				assert.Equal(t, "0xcde", cfg.Onchain.Aptos.DeployerKey)
				assert.Equal(t, "0xdef", cfg.Onchain.Tron.DeployerKey)
				assert.Equal(t, "f1a2b3c4", cfg.Onchain.KMS.KeyID)
				assert.Equal(t, "us-west-1", cfg.Onchain.KMS.KeyRegion)
			},
		},
		{
			name: "Load config with new config enabled (loads from legacy env vars)",
			envvars: map[string]string{
				"CI":                                "true",
				"NEW_CONFIG_ENABLED":                "true",
				"JD_WS_RPC":                         "ws://localhost:1234",
				"JD_GRPC":                           "grpc://localhost:4567",
				"JD_AUTH_COGNITO_APP_CLIENT_ID":     "2b3caf1a",
				"JD_AUTH_COGNITO_APP_CLIENT_SECRET": "22222222",
				"JD_AUTH_AWS_REGION":                "us-east-1",
				"JD_AUTH_USERNAME":                  "testuser2",
				"JD_AUTH_PASSWORD":                  "testpassword2",
				"OCR_X_SIGNERS":                     "testing load signers from env",
				"OCR_X_PROPOSERS":                   "testing load proposers from env",
				"KMS_DEPLOYER_KEY_ID":               "c4f1a2b3",
				"KMS_DEPLOYER_KEY_REGION":           "us-east-1",
				"TEST_WALLET_KEY":                   "0x123",
				"SETH_CONFIG_FILE":                  "/tmp/config",
				"GETH_WRAPPERS_DIRS":                "dir1,dir2",
				"SOLANA_WALLET_KEY":                 "0x234",
				"SOLANA_PROGRAM_PATH":               "0xcde",
				"APTOS_DEPLOYER_KEY":                "0x345",
				"TRON_DEPLOYER_KEY":                 "0x456",
				"CATALOG_SERVICE_GRPC":              "http://localhost:2000",
			},
			wantFunc: func(t *testing.T, cfg *config_env.Config) {
				t.Helper()

				require.NotNil(t, cfg)

				// Validate environment configuration
				assert.Equal(t, "ws://localhost:1234", cfg.Offchain.JobDistributor.Endpoints.WSRPC)
				assert.Equal(t, "grpc://localhost:4567", cfg.Offchain.JobDistributor.Endpoints.GRPC)
				assert.Equal(t, "2b3caf1a", cfg.Offchain.JobDistributor.Auth.CognitoAppClientID)
				assert.Equal(t, "22222222", cfg.Offchain.JobDistributor.Auth.CognitoAppClientSecret)
				assert.Equal(t, "us-east-1", cfg.Offchain.JobDistributor.Auth.AWSRegion)
				assert.Equal(t, "testuser2", cfg.Offchain.JobDistributor.Auth.Username)
				assert.Equal(t, "testpassword2", cfg.Offchain.JobDistributor.Auth.Password)
				assert.Equal(t, "testing load signers from env", cfg.Offchain.OCR.XSigners)
				assert.Equal(t, "testing load proposers from env", cfg.Offchain.OCR.XProposers)
				assert.Equal(t, "c4f1a2b3", cfg.Onchain.KMS.KeyID)
				assert.Equal(t, "us-east-1", cfg.Onchain.KMS.KeyRegion)
				assert.Equal(t, "0x123", cfg.Onchain.EVM.DeployerKey)
				assert.Equal(t, "/tmp/config", cfg.Onchain.EVM.Seth.ConfigFilePath)
				assert.Equal(t, []string{"dir1", "dir2"}, cfg.Onchain.EVM.Seth.GethWrapperDirs)
				assert.Equal(t, "0x234", cfg.Onchain.Solana.WalletKey)
				assert.Equal(t, "0x345", cfg.Onchain.Aptos.DeployerKey)
				assert.Equal(t, "0x456", cfg.Onchain.Tron.DeployerKey)
				assert.Equal(t, "http://localhost:2000", cfg.Catalog.GRPC)
			},
		},
		{
			name: "Load config with new config enabled (loads from env vars)",
			envvars: map[string]string{
				"CI":                                         "true",
				"OFFCHAIN_JD_ENDPOINTS_WSRPC":                "ws://localhost:1234",
				"OFFCHAIN_JD_ENDPOINTS_GRPC":                 "grpc://localhost:4567",
				"OFFCHAIN_JD_AUTH_COGNITO_APP_CLIENT_ID":     "2b3caf1a",
				"OFFCHAIN_JD_AUTH_COGNITO_APP_CLIENT_SECRET": "22222222",
				"OFFCHAIN_JD_AUTH_AWS_REGION":                "us-east-1",
				"OFFCHAIN_JD_AUTH_USERNAME":                  "testuser2",
				"OFFCHAIN_JD_AUTH_PASSWORD":                  "testpassword2",
				"OFFCHAIN_OCR_X_SIGNERS":                     "testing load signers from env",
				"OFFCHAIN_OCR_X_PROPOSERS":                   "testing load proposers from env",
				"ONCHAIN_KMS_KEY_ID":                         "c4f1a2b3",
				"ONCHAIN_KMS_KEY_REGION":                     "us-east-1",
				"ONCHAIN_EVM_DEPLOYER_KEY":                   "0x123",
				"ONCHAIN_EVM_SETH_CONFIG_FILE_PATH":          "/tmp/config",
				"ONCHAIN_EVM_SETH_GETH_WRAPPER_DIRS":         "dir1,dir2",
				"CATALOG_GRPC":                               "http://localhost:2000",
				"ONCHAIN_SOLANA_WALLET_KEY":                  "0x234",
				"ONCHAIN_SOLANA_PROGRAM_PATH":                "0xcde",
				"ONCHAIN_APTOS_DEPLOYER_KEY":                 "0x345",
				"ONCHAIN_TRON_DEPLOYER_KEY":                  "0x456",
				"ONCHAIN_SUI_DEPLOYER_KEY":                   "0x567",
				"ONCHAIN_GETH_WRAPPERS_DIRS":                 "dir1,dir2",
				"ONCHAIN_SETH_CONFIG_FILE":                   "/tmp/config",
			},
			wantFunc: func(t *testing.T, cfg *config_env.Config) {
				t.Helper()

				require.NotNil(t, cfg)

				// Validate environment configuration
				assert.Equal(t, "ws://localhost:1234", cfg.Offchain.JobDistributor.Endpoints.WSRPC)
				assert.Equal(t, "grpc://localhost:4567", cfg.Offchain.JobDistributor.Endpoints.GRPC)
				assert.Equal(t, "2b3caf1a", cfg.Offchain.JobDistributor.Auth.CognitoAppClientID)
				assert.Equal(t, "22222222", cfg.Offchain.JobDistributor.Auth.CognitoAppClientSecret)
				assert.Equal(t, "us-east-1", cfg.Offchain.JobDistributor.Auth.AWSRegion)
				assert.Equal(t, "testuser2", cfg.Offchain.JobDistributor.Auth.Username)
				assert.Equal(t, "testpassword2", cfg.Offchain.JobDistributor.Auth.Password)
				assert.Equal(t, "testing load signers from env", cfg.Offchain.OCR.XSigners)
				assert.Equal(t, "testing load proposers from env", cfg.Offchain.OCR.XProposers)
				assert.Equal(t, "c4f1a2b3", cfg.Onchain.KMS.KeyID)
				assert.Equal(t, "us-east-1", cfg.Onchain.KMS.KeyRegion)
				assert.Equal(t, "0x123", cfg.Onchain.EVM.DeployerKey)
				assert.Equal(t, []string{"dir1", "dir2"}, cfg.Onchain.EVM.Seth.GethWrapperDirs)
				assert.Equal(t, "/tmp/config", cfg.Onchain.EVM.Seth.ConfigFilePath)
				assert.Equal(t, "0x234", cfg.Onchain.Solana.WalletKey)
				assert.Equal(t, "0x345", cfg.Onchain.Aptos.DeployerKey)
				assert.Equal(t, "0x456", cfg.Onchain.Tron.DeployerKey)
				assert.Equal(t, "0x567", cfg.Onchain.Sui.DeployerKey)
				assert.Equal(t, "http://localhost:2000", cfg.Catalog.GRPC)
			},
		},
	}

	for _, tt := range tests { //nolint:paralleltest // These tests are not parallel safe due to setting of env vars
		t.Run(tt.name, func(t *testing.T) {
			dom, envKey := setupConfigDirs(t)
			writeConfigLocalFile(t, dom, envKey, "config.testnet.yaml")

			if tt.skipCI {
				t.Skip("Skipping test in CI")
			}

			// Setup environment variables for this test
			if tt.envvars != nil {
				for k, v := range tt.envvars {
					os.Setenv(k, v)
				}

				t.Cleanup(func() {
					for k := range tt.envvars {
						os.Unsetenv(k)
					}
				})
			}

			// Execute the test
			got, err := LoadEnvConfig(dom, envKey)

			// Check for expected errors
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				// Validate successful case
				require.NoError(t, err)

				if tt.wantFunc == nil {
					require.Fail(t, "you must provide a wantFunc check for this test")
				}

				tt.wantFunc(t, got)
			}
		})
	}
}
