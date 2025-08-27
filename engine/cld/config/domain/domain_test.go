package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidNetworkAccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		access   string
		expected bool
	}{
		{
			name:     "mainnet is valid",
			access:   "mainnet",
			expected: true,
		},
		{
			name:     "testnet is valid",
			access:   "testnet",
			expected: true,
		},
		{
			name:     "invalid value",
			access:   "invalid",
			expected: false,
		},
		{
			name:     "empty value",
			access:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, isValidNetworkAccess(tt.access))
		})
	}
}

func TestEnvironment_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		environment Environment
		wantErr     bool
		errContains string
	}{
		{
			name: "valid environment with both access types",
			environment: Environment{
				NetworkAccess: []string{"testnet", "mainnet"},
			},
			wantErr: false,
		},
		{
			name: "valid environment with mainnet only",
			environment: Environment{
				NetworkAccess: []string{"mainnet"},
			},
			wantErr: false,
		},
		{
			name: "valid environment with testnet only",
			environment: Environment{
				NetworkAccess: []string{"testnet"},
			},
			wantErr: false,
		},
		{
			name: "empty network access",
			environment: Environment{
				NetworkAccess: []string{},
			},
			wantErr:     true,
			errContains: "networkAccess is required and cannot be empty",
		},
		{
			name: "invalid network access value",
			environment: Environment{
				NetworkAccess: []string{"invalid"},
			},
			wantErr:     true,
			errContains: "invalid networkAccess value: invalid",
		},
		{
			name: "duplicate network access values",
			environment: Environment{
				NetworkAccess: []string{"testnet", "testnet"},
			},
			wantErr:     true,
			errContains: "duplicate networkAccess value: testnet",
		},
		{
			name: "duplicate mainnet values",
			environment: Environment{
				NetworkAccess: []string{"mainnet", "mainnet"},
			},
			wantErr:     true,
			errContains: "duplicate networkAccess value: mainnet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.environment.validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filePath string
		wantErr  bool
		validate func(t *testing.T, config *DomainConfig)
	}{
		{
			name:     "load valid yaml fixture file",
			filePath: "testdata/valid.yaml",
			wantErr:  false,
			validate: func(t *testing.T, config *DomainConfig) {
				t.Helper()

				require.NotNil(t, config)

				// Check that all expected environments are loaded
				assert.Len(t, config.Environments, 6)

				// Check specific environments exist
				assert.Contains(t, config.Environments, "development")
				assert.Contains(t, config.Environments, "staging")
				assert.Contains(t, config.Environments, "production")
				assert.Contains(t, config.Environments, "local")
				assert.Contains(t, config.Environments, "dudeenv")
				assert.Contains(t, config.Environments, "testtest")

				// Test specific environment configurations
				dev := config.Environments["development"]
				assert.Equal(t, []string{"testnet"}, dev.NetworkAccess)

				staging := config.Environments["staging"]
				assert.ElementsMatch(t, []string{"testnet", "mainnet"}, staging.NetworkAccess)

				prod := config.Environments["production"]
				assert.Equal(t, []string{"mainnet"}, prod.NetworkAccess)

				local := config.Environments["local"]
				assert.Equal(t, []string{"testnet"}, local.NetworkAccess)

				dudeenv := config.Environments["dudeenv"]
				assert.Equal(t, []string{"testnet"}, dudeenv.NetworkAccess)

				testtest := config.Environments["testtest"]
				assert.Equal(t, []string{"mainnet"}, testtest.NetworkAccess)
			},
		},
		{
			name:     "invalid network access in environment",
			filePath: "testdata/mixed_access.yaml",
			wantErr:  true,
		},
		{
			name:     "error when file not found",
			filePath: "/tmp/nonexistent_file.yml",
			wantErr:  true,
		},
		{
			name:     "invalid yaml content",
			filePath: "testdata/invalid.yaml",
			wantErr:  true,
		},
		{
			name:     "empty environments",
			filePath: "testdata/empty_environments.yaml",
			wantErr:  false,
			validate: func(t *testing.T, config *DomainConfig) {
				t.Helper()

				require.NotNil(t, config)
				assert.Empty(t, config.Environments)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config, err := Load(tt.filePath)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, config)
			} else {
				require.NoError(t, err)
				require.NotNil(t, config)
				if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}
