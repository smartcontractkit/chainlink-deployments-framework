package domain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidNetworkType(t *testing.T) {
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

			assert.Equal(t, tt.expected, isValidNetworkType(tt.access))
		})
	}
}

func TestIsValidDatastore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		datastore string
		expected  bool
	}{
		{
			name:      "file is valid",
			datastore: "file",
			expected:  true,
		},
		{
			name:      "catalog is valid",
			datastore: "catalog",
			expected:  true,
		},
		{
			name:      "invalid value",
			datastore: "invalid",
			expected:  false,
		},
		{
			name:      "empty value",
			datastore: "",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, isValidDatastore(tt.datastore))
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
				NetworkTypes: []string{"testnet", "mainnet"},
			},
			wantErr: false,
		},
		{
			name: "valid environment with mainnet only",
			environment: Environment{
				NetworkTypes: []string{"mainnet"},
			},
			wantErr: false,
		},
		{
			name: "valid environment with testnet only",
			environment: Environment{
				NetworkTypes: []string{"testnet"},
			},
			wantErr: false,
		},
		{
			name: "valid environment with file datastore",
			environment: Environment{
				NetworkTypes: []string{"testnet"},
				Datastore:    "file",
			},
			wantErr: false,
		},
		{
			name: "valid environment with catalog datastore",
			environment: Environment{
				NetworkTypes: []string{"mainnet"},
				Datastore:    "catalog",
			},
			wantErr: false,
		},
		{
			name: "valid environment without datastore (optional field)",
			environment: Environment{
				NetworkTypes: []string{"testnet"},
				Datastore:    "",
			},
			wantErr: false,
		},
		{
			name: "empty network access",
			environment: Environment{
				NetworkTypes: []string{},
			},
			wantErr:     true,
			errContains: "network_types is required and cannot be empty",
		},
		{
			name: "invalid network access value",
			environment: Environment{
				NetworkTypes: []string{"invalid"},
			},
			wantErr:     true,
			errContains: "invalid network_types value: invalid",
		},
		{
			name: "duplicate network access values",
			environment: Environment{
				NetworkTypes: []string{"testnet", "testnet"},
			},
			wantErr:     true,
			errContains: "duplicate network_types value: testnet",
		},
		{
			name: "duplicate mainnet values",
			environment: Environment{
				NetworkTypes: []string{"mainnet", "mainnet"},
			},
			wantErr:     true,
			errContains: "duplicate network_types value: mainnet",
		},
		{
			name: "invalid datastore value",
			environment: Environment{
				NetworkTypes: []string{"testnet"},
				Datastore:    "invalid",
			},
			wantErr:     true,
			errContains: "invalid datastore value: invalid",
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
				assert.Equal(t, []string{"testnet"}, dev.NetworkTypes)
				assert.Equal(t, "file", dev.Datastore)

				staging := config.Environments["staging"]
				assert.ElementsMatch(t, []string{"testnet", "mainnet"}, staging.NetworkTypes)
				assert.Equal(t, "catalog", staging.Datastore)

				prod := config.Environments["production"]
				assert.Equal(t, []string{"mainnet"}, prod.NetworkTypes)
				assert.Equal(t, "catalog", prod.Datastore)

				local := config.Environments["local"]
				assert.Equal(t, []string{"testnet"}, local.NetworkTypes)
				assert.Equal(t, "file", local.Datastore)

				dudeenv := config.Environments["dudeenv"]
				assert.Equal(t, []string{"testnet"}, dudeenv.NetworkTypes)
				assert.Equal(t, "file", dudeenv.Datastore) // Not set in YAML, should default to "file"

				testtest := config.Environments["testtest"]
				assert.Equal(t, []string{"mainnet"}, testtest.NetworkTypes)
				assert.Equal(t, "catalog", testtest.Datastore)
			},
		},
		{
			name:     "invalid network access in environment",
			filePath: "testdata/mixed.yaml",
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
			filePath: "testdata/empty.yaml",
			wantErr:  true,
		},
		{
			name:     "datastore defaults to file when not specified",
			filePath: "testdata/valid.yaml",
			wantErr:  false,
			validate: func(t *testing.T, config *DomainConfig) {
				t.Helper()

				// dudeenv doesn't have datastore specified in YAML, should default to "file"
				dudeenv := config.Environments["dudeenv"]
				assert.Equal(t, "file", dudeenv.Datastore)
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

func TestLoad_WithJiraConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		checkConfig func(*DomainConfig) error
	}{
		{
			name: "valid config with jira",
			configYAML: `
environments:
  testnet:
    network_types:
      - testnet

jira:
  connection:
    base_url: "https://example.atlassian.net"
    project: "TEST"
    username: "testuser"
  field_maps:
    summary:
      jira_field: "summary"
    status:
      jira_field: "status"
    custom_field:
      jira_field: "customfield_10001"
`,
			expectError: false,
			checkConfig: func(config *DomainConfig) error {
				if config.Jira == nil {
					return assert.AnError
				}
				if config.Jira.Connection.BaseURL != "https://example.atlassian.net" {
					return assert.AnError
				}
				if config.Jira.Connection.Project != "TEST" {
					return assert.AnError
				}
				if config.Jira.Connection.Username != "testuser" {
					return assert.AnError
				}
				if len(config.Jira.FieldMaps) != 3 {
					return assert.AnError
				}

				return nil
			},
		},
		{
			name: "config without jira (optional)",
			configYAML: `
environments:
  testnet:
    network_types:
      - testnet
`,
			expectError: false,
			checkConfig: func(config *DomainConfig) error {
				if config.Jira != nil {
					return assert.AnError
				}

				return nil
			},
		},
		{
			name: "missing base_url",
			configYAML: `
environments:
  testnet:
    network_types:
      - testnet

jira:
  connection:
    project: "TEST"
    username: "testuser"
  field_maps:
    summary:
      jira_field: "summary"
`,
			expectError: true,
		},
		{
			name: "missing project",
			configYAML: `
environments:
  testnet:
    network_types:
      - testnet

jira:
  connection:
    base_url: "https://example.atlassian.net"
    username: "testuser"
  field_maps:
    summary:
      jira_field: "summary"
`,
			expectError: true,
		},
		{
			name: "missing username",
			configYAML: `
environments:
  testnet:
    network_types:
      - testnet

jira:
  connection:
    base_url: "https://example.atlassian.net"
    project: "TEST"
  field_maps:
    summary:
      jira_field: "summary"
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temporary file with test config
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "domain.yaml")
			err := os.WriteFile(configPath, []byte(tt.configYAML), 0600)
			require.NoError(t, err)

			// Load config
			config, err := Load(configPath)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, config)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, config)

			if tt.checkConfig != nil {
				require.NoError(t, tt.checkConfig(config))
			}
		})
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	t.Parallel()

	// Create temporary file with invalid YAML
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "domain.yaml")

	invalidYAML := `
environments:
  testnet:
    network_types:
      - testnet

jira:
  connection:
    base_url: "https://example.atlassian.net"
    project: "TEST"
    username: "testuser"
  field_maps:
    summary:
      jira_field: "summary"
invalid: [unclosed
`
	err := os.WriteFile(configPath, []byte(invalidYAML), 0600)
	require.NoError(t, err)

	// Load config should fail
	config, err := Load(configPath)
	require.Error(t, err)
	require.Nil(t, config)
}

func TestLoad_FileNotFound(t *testing.T) {
	t.Parallel()

	// Try to load non-existent file
	config, err := Load("/non/existent/path.yaml")
	require.Error(t, err)
	require.Nil(t, config)
}
