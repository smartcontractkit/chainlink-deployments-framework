package domain

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidNetworkAccess(t *testing.T) {
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
			assert.Equal(t, tt.expected, isValidNetworkAccess(tt.access))
		})
	}
}

func TestEnvironment_Validate(t *testing.T) {
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
			err := tt.environment.Validate()
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
	tests := []struct {
		name           string
		useFixtureFile bool
		filePath       string
		yamlContent    string
		wantErr        bool
		validate       func(t *testing.T, config *DomainConfig)
	}{
		{
			name:           "load valid yaml fixture file",
			useFixtureFile: true,
			filePath:       "testdata/valid.yaml",
			wantErr:        false,
			validate: func(t *testing.T, config *DomainConfig) {
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
			name:           "error when file not found",
			useFixtureFile: true,
			filePath:       "/tmp/nonexistent_file.yml",
			wantErr:        true,
		},
		{
			name:           "invalid yaml content",
			useFixtureFile: true,
			filePath:       "testdata/invalid.yaml",
			wantErr:        true,
		},
		{
			name:           "empty environments",
			useFixtureFile: true,
			filePath:       "testdata/empty_environments.yaml",
			wantErr:        false,
			validate: func(t *testing.T, config *DomainConfig) {
				require.NotNil(t, config)
				assert.Len(t, config.Environments, 0)
			},
		},
		{
			name:           "multiple environments with mixed access",
			useFixtureFile: true,
			filePath:       "testdata/mixed_access.yaml",
			wantErr:        false,
			validate: func(t *testing.T, config *DomainConfig) {
				require.NotNil(t, config)
				assert.Len(t, config.Environments, 3)

				dev := config.Environments["dev"]
				assert.Equal(t, []string{"testnet"}, dev.NetworkAccess)

				staging := config.Environments["staging"]
				assert.ElementsMatch(t, []string{"testnet", "mainnet"}, staging.NetworkAccess)

				prod := config.Environments["prod"]
				assert.Equal(t, []string{"mainnet"}, prod.NetworkAccess)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filePath string

			if tt.useFixtureFile {
				filePath = tt.filePath
			} else {
				// Create temporary file with YAML content
				tmpFile, err := os.CreateTemp("", "domain_test_*.yml")
				require.NoError(t, err)
				defer os.Remove(tmpFile.Name())

				_, err = tmpFile.WriteString(tt.yamlContent)
				require.NoError(t, err)
				require.NoError(t, tmpFile.Close())

				filePath = tmpFile.Name()
			}

			config, err := Load(filePath)

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

func TestDomainConfig_Validation(t *testing.T) {
	tests := []struct {
		name           string
		yamlContent    string
		loadShouldFail bool
		envToValidate  string
		validationErr  bool
		errContains    string
		validate       func(t *testing.T, config *DomainConfig)
	}{
		{
			name: "environments with invalid network access should be loadable but fail validation",
			yamlContent: `
environments:
  development:
    networkAccess:
      - invalid_access
`,
			loadShouldFail: false,
			envToValidate:  "development",
			validationErr:  true,
			errContains:    "invalid networkAccess value: invalid_access",
		},
		{
			name: "environments with duplicate network access should fail validation",
			yamlContent: `
environments:
  development:
    networkAccess:
      - testnet
      - testnet
`,
			loadShouldFail: false,
			envToValidate:  "development",
			validationErr:  true,
			errContains:    "duplicate networkAccess value: testnet",
		},
		{
			name: "environments with duplicate mainnet should fail validation",
			yamlContent: `
environments:
  production:
    networkAccess:
      - mainnet
      - mainnet
`,
			loadShouldFail: false,
			envToValidate:  "production",
			validationErr:  true,
			errContains:    "duplicate networkAccess value: mainnet",
		},
		{
			name: "environments with empty network access should fail validation",
			yamlContent: `
environments:
  development:
    networkAccess: []
`,
			loadShouldFail: false,
			envToValidate:  "development",
			validationErr:  true,
			errContains:    "networkAccess is required and cannot be empty",
		},
		{
			name: "environments with mixed invalid and valid access should fail validation",
			yamlContent: `
environments:
  staging:
    networkAccess:
      - testnet
      - invalid_network
      - mainnet
`,
			loadShouldFail: false,
			envToValidate:  "staging",
			validationErr:  true,
			errContains:    "invalid networkAccess value: invalid_network",
		},
		{
			name: "empty environments map should load successfully",
			yamlContent: `
environments: {}
`,
			loadShouldFail: false,
			validate: func(t *testing.T, config *DomainConfig) {
				require.NotNil(t, config)
				assert.Len(t, config.Environments, 0)
			},
		},
		{
			name: "valid environments should pass validation",
			yamlContent: `
environments:
  development:
    networkAccess:
      - testnet
  staging:
    networkAccess:
      - testnet
      - mainnet
  production:
    networkAccess:
      - mainnet
`,
			loadShouldFail: false,
			validate: func(t *testing.T, config *DomainConfig) {
				require.NotNil(t, config)
				assert.Len(t, config.Environments, 3)

				// Validate each environment
				for envName, env := range config.Environments {
					err := env.Validate()
					assert.NoError(t, err, "Environment %s should be valid", envName)
				}
			},
		},
		{
			name: "multiple environments with some invalid should allow selective validation",
			yamlContent: `
environments:
  valid_env:
    networkAccess:
      - testnet
  invalid_env:
    networkAccess:
      - invalid_access
`,
			loadShouldFail: false,
			validate: func(t *testing.T, config *DomainConfig) {
				require.NotNil(t, config)
				assert.Len(t, config.Environments, 2)

				// Valid environment should pass validation
				validEnv := config.Environments["valid_env"]
				err := validEnv.Validate()
				assert.NoError(t, err)

				// Invalid environment should fail validation
				invalidEnv := config.Environments["invalid_env"]
				err = invalidEnv.Validate()
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid networkAccess value: invalid_access")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file with YAML content
			tmpFile, err := os.CreateTemp("", "domain_test_*.yml")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tt.yamlContent)
			require.NoError(t, err)
			require.NoError(t, tmpFile.Close())

			config, err := Load(tmpFile.Name())

			if tt.loadShouldFail {
				require.Error(t, err)
				assert.Nil(t, config)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, config)

			// If we need to validate a specific environment
			if tt.envToValidate != "" {
				env, exists := config.Environments[tt.envToValidate]
				require.True(t, exists, "Environment %s should exist", tt.envToValidate)

				err = env.Validate()
				if tt.validationErr {
					require.Error(t, err)
					if tt.errContains != "" {
						assert.Contains(t, err.Error(), tt.errContains)
					}
				} else {
					require.NoError(t, err)
				}
			}

			// Run custom validation if provided
			if tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}
