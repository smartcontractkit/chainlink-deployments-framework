package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// fileCfg is the config that is loaded from the testdata/config.yml file.
	fileCfg = &Config{
		Onchain: OnchainConfig{
			KMS: KMSConfig{
				KeyID:     "f1a2b3c4",
				KeyRegion: "us-west-1",
			},
			EVM: EVMConfig{
				DeployerKey: "0xabc",
				Seth: &SethConfig{
					ConfigFilePath:  "/tmp/config",
					GethWrapperDirs: []string{"./dir1", "./dir2"},
				},
			},
			Solana: SolanaConfig{
				WalletKey:       "0xbcd",
				ProgramsDirPath: "/tmp/program",
			},
			Aptos: AptosConfig{
				DeployerKey: "0xcde",
			},
			Tron: TronConfig{
				DeployerKey: "0xdef",
			},
		},
		Offchain: OffchainConfig{
			JobDistributor: JobDistributorConfig{
				Auth: &JobDistributorAuth{
					CognitoAppClientID:     "af1a2b3c",
					CognitoAppClientSecret: "11111111",
					AWSRegion:              "us-west-1",
					Username:               "user1",
					Password:               "password1",
				},
				Endpoints: JobDistributorEndpoints{
					WSRPC: "ws://localhost:1234",
					GRPC:  "grpc://localhost:4567",
				},
			},
		},
		Catalog: CatalogConfig{
			GRPC: "http://localhost:1000",
		},
	}

	// envVars is the environment variables that used to set the config.
	envVars = map[string]string{
		"ONCHAIN_KMS_KEY_ID":                         "123",
		"ONCHAIN_KMS_KEY_REGION":                     "us-east-1",
		"ONCHAIN_EVM_DEPLOYER_KEY":                   "0x123",
		"ONCHAIN_EVM_SETH_CONFIG_FILE_PATH":          "config.json",
		"ONCHAIN_EVM_SETH_GETH_WRAPPER_DIRS":         "./a,./b",
		"ONCHAIN_SOLANA_WALLET_KEY":                  "0x123",
		"ONCHAIN_SOLANA_PROGRAMS_DIR_PATH":           "/tmp",
		"ONCHAIN_APTOS_DEPLOYER_KEY":                 "0x123",
		"ONCHAIN_TRON_DEPLOYER_KEY":                  "0x123",
		"ONCHAIN_SUI_DEPLOYER_KEY":                   "0x123",
		"OFFCHAIN_JD_AUTH_COGNITO_APP_CLIENT_ID":     "123",
		"OFFCHAIN_JD_AUTH_COGNITO_APP_CLIENT_SECRET": "123",
		"OFFCHAIN_JD_AUTH_AWS_REGION":                "us-east-1",
		"OFFCHAIN_JD_AUTH_USERNAME":                  "123",
		"OFFCHAIN_JD_AUTH_PASSWORD":                  "123",
		"OFFCHAIN_JD_ENDPOINTS_WSRPC":                "WSRPC2",
		"OFFCHAIN_JD_ENDPOINTS_GRPC":                 "GRPC2",
		"OFFCHAIN_OCR_X_SIGNERS":                     "awkward bat",
		"OFFCHAIN_OCR_X_PROPOSERS":                   "caring deer",
		"CATALOG_GRPC":                               "http://localhost:8080",
	}

	legacyEnvVars = map[string]string{
		"KMS_DEPLOYER_KEY_ID":               "123",
		"KMS_DEPLOYER_KEY_REGION":           "us-east-1",
		"TEST_WALLET_KEY":                   "0x123",
		"SETH_CONFIG_FILE":                  "config.json",
		"GETH_WRAPPERS_DIRS":                "./a,./b",
		"SOLANA_WALLET_KEY":                 "0x123",
		"SOLANA_PROGRAM_PATH":               "/tmp",
		"APTOS_DEPLOYER_KEY":                "0x123",
		"TRON_DEPLOYER_KEY":                 "0x123",
		"JD_AUTH_COGNITO_APP_CLIENT_ID":     "123",
		"JD_AUTH_COGNITO_APP_CLIENT_SECRET": "123",
		"JD_AUTH_AWS_REGION":                "us-east-1",
		"JD_AUTH_USERNAME":                  "123",
		"JD_AUTH_PASSWORD":                  "123",
		"JD_WS_RPC":                         "WSRPC2",
		"JD_GRPC":                           "GRPC2",
		"OCR_X_SIGNERS":                     "awkward bat",
		"OCR_X_PROPOSERS":                   "caring deer",
		"CATALOG_SERVICE_GRPC":              "http://localhost:8080",
		// These values do not have a legacy equivalent
		"ONCHAIN_SUI_DEPLOYER_KEY": "0x123",
	}

	// envCfg is the config that is loaded from the environment variables.
	envCfg = &Config{
		Onchain: OnchainConfig{
			KMS: KMSConfig{
				KeyID:     "123",
				KeyRegion: "us-east-1",
			},
			EVM: EVMConfig{
				DeployerKey: "0x123",
				Seth: &SethConfig{
					ConfigFilePath:  "config.json",
					GethWrapperDirs: []string{"./a", "./b"},
				},
			},
			Solana: SolanaConfig{
				WalletKey:       "0x123",
				ProgramsDirPath: "/tmp",
			},
			Aptos: AptosConfig{
				DeployerKey: "0x123",
			},
			Tron: TronConfig{
				DeployerKey: "0x123",
			},
			Sui: SuiConfig{
				DeployerKey: "0x123",
			},
		},
		Offchain: OffchainConfig{
			JobDistributor: JobDistributorConfig{
				Auth: &JobDistributorAuth{
					CognitoAppClientID:     "123",
					CognitoAppClientSecret: "123",
					AWSRegion:              "us-east-1",
					Username:               "123",
					Password:               "123",
				},
				Endpoints: JobDistributorEndpoints{
					WSRPC: "WSRPC2",
					GRPC:  "GRPC2",
				},
			},
			OCR: OCRConfig{
				XSigners:   "awkward bat",
				XProposers: "caring deer",
			},
		},
		Catalog: CatalogConfig{
			GRPC: "http://localhost:8080",
		},
	}
)

func Test_Load(t *testing.T) { //nolint:paralleltest // see comment in setupTestEnvVars
	tests := []struct {
		name       string
		beforeFunc func(t *testing.T)
		givePath   string
		want       *Config
		wantErr    string
	}{
		{
			name:     "load from file",
			givePath: "./testdata/config.yml",
			want:     fileCfg,
		},
		{
			name:     "load from empty file and env vars",
			givePath: "./testdata/empty.yml",
			want: &Config{
				Onchain: OnchainConfig{
					KMS: KMSConfig{},
					EVM: EVMConfig{
						Seth: nil, // Testing optional pointer fields
					},
					Solana: SolanaConfig{},
					Aptos:  AptosConfig{},
					Tron:   TronConfig{},
				},
				Offchain: OffchainConfig{
					JobDistributor: JobDistributorConfig{
						Auth:      nil, // Testing optional pointer fields
						Endpoints: JobDistributorEndpoints{},
					},
					OCR: OCRConfig{},
				},
				Catalog: CatalogConfig{},
			},
		},
		{
			name: "override with env",
			beforeFunc: func(t *testing.T) {
				t.Helper()

				setupEnvVars(t, envVars)
			},
			givePath: "./testdata/config.yml",
			want:     envCfg,
		},
		{
			name: "fallback to env when file not found",
			beforeFunc: func(t *testing.T) {
				t.Helper()

				setupEnvVars(t, envVars)
			},
			givePath: "./testdata/invalid.yml",
			want:     envCfg,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // see comment in setupTestEnvVars
		t.Run(tt.name, func(t *testing.T) {
			if tt.beforeFunc != nil {
				tt.beforeFunc(t)
			}

			got, err := Load(tt.givePath)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_LoadFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		givePath string
		want     *Config
		wantErr  string
	}{
		{
			name:     "load from file",
			givePath: "./testdata/config.yml",
			want:     fileCfg,
		},
		{
			name:     "load from file with invalid path",
			givePath: "./testdata/invalid.yml",
			wantErr:  "no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := LoadFile(tt.givePath)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_LoadEnv(t *testing.T) { //nolint:paralleltest // see comment in setupEnvVars
	setupEnvVars(t, envVars)

	got, err := LoadEnv()
	require.NoError(t, err)

	assert.Equal(t, envCfg, got)
}

func Test_LoadEnv_Legacy(t *testing.T) { //nolint:paralleltest // see comment in setupEnvVars
	setupEnvVars(t, legacyEnvVars)

	got, err := LoadEnv()
	require.NoError(t, err)

	assert.Equal(t, envCfg, got)
}

// setupTestEnvVars sets up the environment variables for the test.
//
// CAUTION: Because this function uses t.Setenv which affects the entire process, tests which call
// this function cannot be run in parallel.
func setupEnvVars(t *testing.T, envVars map[string]string) {
	t.Helper()

	for key, value := range envVars {
		t.Setenv(key, value)
	}
}
