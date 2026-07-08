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

func TestDatastoreType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		datastore DatastoreType
		expected  bool
	}{
		{
			name:      "file is valid",
			datastore: DatastoreTypeFile,
			expected:  true,
		},
		{
			name:      "catalog is valid",
			datastore: DatastoreTypeCatalog,
			expected:  true,
		},
		{
			name:      "all is valid",
			datastore: DatastoreTypeAll,
			expected:  true,
		},
		{
			name:      "invalid value",
			datastore: DatastoreType("invalid"),
			expected:  false,
		},
		{
			name:      "empty value",
			datastore: DatastoreType(""),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.datastore.IsValid())
		})
	}
}

func TestEnvironment_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		environment Environment
		wantErr     string
	}{
		{
			name: "valid environment with both access types",
			environment: Environment{
				NetworkTypes: []string{"testnet", "mainnet"},
			},
		},
		{
			name: "valid environment with mainnet only",
			environment: Environment{
				NetworkTypes: []string{"mainnet"},
			},
		},
		{
			name: "valid environment with testnet only",
			environment: Environment{
				NetworkTypes: []string{"testnet"},
			},
		},
		{
			name: "valid environment with file datastore",
			environment: Environment{
				NetworkTypes: []string{"testnet"},
				Datastore:    DatastoreTypeFile,
			},
		},
		{
			name: "valid environment with catalog datastore",
			environment: Environment{
				NetworkTypes: []string{"mainnet"},
				Datastore:    DatastoreTypeCatalog,
			},
		},
		{
			name: "valid environment with all datastore",
			environment: Environment{
				NetworkTypes: []string{"testnet"},
				Datastore:    DatastoreTypeAll,
			},
		},
		{
			name: "valid environment without datastore (optional field)",
			environment: Environment{
				NetworkTypes: []string{"testnet"},
				Datastore:    "",
			},
		},
		{
			name: "empty network access",
			environment: Environment{
				NetworkTypes: []string{},
			},
			wantErr: "network_types is required and cannot be empty",
		},
		{
			name: "invalid network access value",
			environment: Environment{
				NetworkTypes: []string{"invalid"},
			},
			wantErr: "invalid network_types value: invalid (must be 'mainnet' or 'testnet')",
		},
		{
			name: "duplicate network access values",
			environment: Environment{
				NetworkTypes: []string{"testnet", "testnet"},
			},
			wantErr: "duplicate network_types value: testnet",
		},
		{
			name: "duplicate mainnet values",
			environment: Environment{
				NetworkTypes: []string{"mainnet", "mainnet"},
			},
			wantErr: "duplicate network_types value: mainnet",
		},
		{
			name: "invalid datastore value",
			environment: Environment{
				NetworkTypes: []string{"testnet"},
				Datastore:    DatastoreType("invalid"),
			},
			wantErr: "invalid datastore value: invalid (must be 'file', 'catalog', or 'all')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.environment.validate()
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
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

				// Test specific environment configurations
				dev := config.Environments["development"]
				assert.Equal(t, []string{"testnet"}, dev.NetworkTypes)
				assert.Equal(t, DatastoreTypeFile, dev.Datastore)

				staging := config.Environments["staging"]
				assert.ElementsMatch(t, []string{"testnet", "mainnet"}, staging.NetworkTypes)
				assert.Equal(t, DatastoreTypeCatalog, staging.Datastore)

				prod := config.Environments["production"]
				assert.Equal(t, []string{"mainnet"}, prod.NetworkTypes)
				assert.Equal(t, DatastoreTypeCatalog, prod.Datastore)

				local := config.Environments["local"]
				assert.Equal(t, []string{"testnet"}, local.NetworkTypes)
				assert.Equal(t, DatastoreTypeFile, local.Datastore)

				dudeenv := config.Environments["dudeenv"]
				assert.Equal(t, []string{"testnet"}, dudeenv.NetworkTypes)
				assert.Equal(t, DatastoreTypeFile, dudeenv.Datastore) // Not set in YAML, should default to "file"

				testtest := config.Environments["testtest"]
				assert.Equal(t, []string{"mainnet"}, testtest.NetworkTypes)
				assert.Equal(t, DatastoreTypeCatalog, testtest.Datastore)
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
				assert.Equal(t, DatastoreTypeFile, dudeenv.Datastore)
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

func TestLoad_BinaryConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		configYAML   string
		wantProvider BinaryProvider
		wantVersion  string
		wantErr      string
	}{
		{
			name: "loads explicit s3 config",
			configYAML: `
environments:
  testnet:
    network_types:
      - testnet

binary:
  provider: s3
  version: latest
`,
			wantProvider: BinaryProviderS3,
			wantVersion:  "latest",
		},
		{
			name: "defaults version to latest",
			configYAML: `
environments:
  testnet:
    network_types:
      - testnet

binary:
  provider: s3
`,
			wantProvider: BinaryProviderS3,
			wantVersion:  DefaultBinaryVersion,
		},
		{
			name: "defaults provider to source when provider is omitted",
			configYAML: `
environments:
  testnet:
    network_types:
      - testnet

binary:
  version: v1.2.3
`,
			wantProvider: BinaryProviderSource,
			wantVersion:  "v1.2.3",
		},
		{
			name: "defaults to source when binary section is absent",
			configYAML: `
environments:
  testnet:
    network_types:
      - testnet
`,
			wantProvider: BinaryProviderSource,
			wantVersion:  DefaultBinaryVersion,
		},
		{
			name: "rejects unsupported provider",
			configYAML: `
environments:
  testnet:
    network_types:
      - testnet

binary:
  provider: artifact-registry
`,
			wantErr: "invalid binary configuration: invalid binary provider: artifact-registry (must be 'source' or 's3')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "domain.yaml")
			require.NoError(t, os.WriteFile(configPath, []byte(tt.configYAML), 0600))

			config, err := Load(configPath)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Nil(t, config)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, config.Binary)
			assert.Equal(t, tt.wantProvider, config.Binary.Provider)
			assert.Equal(t, tt.wantVersion, config.Binary.Version)
		})
	}
}

func TestLoad_CREDefaultRegistries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		checkConfig func(*DomainConfig) error
	}{
		{
			name: "valid cre.default_registries on env",
			configYAML: `
environments:
  testnet:
    network_types:
      - testnet
    cre:
      enabled: true
      default_registries:
        - id: private
          label: "Private (Chainlink-hosted)"
          type: off-chain
          secrets_auth_flows:
            - browser
            - owner-key-signing
`,
			expectError: false,
			checkConfig: func(config *DomainConfig) error {
				env := config.Environments["testnet"]
				if env.CRE == nil || !env.CRE.Enabled {
					return assert.AnError
				}
				if len(env.CRE.DefaultRegistries) != 1 {
					return assert.AnError
				}
				r := env.CRE.DefaultRegistries[0]
				if r.ID != "private" || r.Type != "off-chain" {
					return assert.AnError
				}

				return nil
			},
		},
		{
			name: "missing registry id",
			configYAML: `
environments:
  testnet:
    network_types:
      - testnet
    cre:
      enabled: true
      default_registries:
        - label: "x"
          type: off-chain
`,
			expectError: true,
		},
		{
			name: "cre disabled - registries ignored during validation",
			configYAML: `
environments:
  testnet:
    network_types:
      - testnet
    cre:
      enabled: false
`,
			expectError: false,
			checkConfig: func(config *DomainConfig) error {
				env := config.Environments["testnet"]
				if env.CRE == nil || env.CRE.Enabled {
					return assert.AnError
				}

				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "domain.yaml")
			err := os.WriteFile(configPath, []byte(tt.configYAML), 0600)
			require.NoError(t, err)

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
