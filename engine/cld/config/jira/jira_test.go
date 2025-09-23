package jira

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJiraConfig_GetJiraFields(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		config   JiraConfig
		expected []string
	}{
		{
			name: "single field mapping",
			config: JiraConfig{
				FieldMaps: map[string]FieldMapping{
					"summary": {JiraField: "summary"},
				},
			},
			expected: []string{"summary"},
		},
		{
			name: "multiple field mappings",
			config: JiraConfig{
				FieldMaps: map[string]FieldMapping{
					"summary":     {JiraField: "summary"},
					"status":      {JiraField: "status"},
					"customField": {JiraField: "customfield_10001"},
				},
			},
			expected: []string{"summary", "status", "customfield_10001"},
		},
		{
			name: "custom field mappings",
			config: JiraConfig{
				FieldMaps: map[string]FieldMapping{
					"summary":     {JiraField: "summary"},
					"storyPoints": {JiraField: "customfield_10028"}, // Story Points
					"priority":    {JiraField: "customfield_10016"}, // Priority
					"epicLink":    {JiraField: "customfield_10014"}, // Epic Link
					"description": {JiraField: "description"},
				},
			},
			expected: []string{"summary", "customfield_10028", "customfield_10016", "customfield_10014", "description"},
		},
		{
			name: "empty field mappings",
			config: JiraConfig{
				FieldMaps: map[string]FieldMapping{},
			},
			expected: []string{},
		},
		{
			name: "nil field mappings",
			config: JiraConfig{
				FieldMaps: nil,
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.config.GetJiraFields()

			assert.Len(t, result, len(tt.expected))

			// Convert to map for easier comparison
			resultMap := make(map[string]bool)
			for _, field := range result {
				resultMap[field] = true
			}

			for _, expectedField := range tt.expected {
				assert.True(t, resultMap[expectedField], "Expected field %s not found in result", expectedField)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		checkConfig func(*JiraConfig) error
	}{
		{
			name: "valid config",
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
			checkConfig: func(config *JiraConfig) error {
				if config.Connection.BaseURL != "https://example.atlassian.net" {
					return assert.AnError
				}
				if config.Connection.Project != "TEST" {
					return assert.AnError
				}
				if config.Connection.Username != "testuser" {
					return assert.AnError
				}
				if len(config.FieldMaps) != 3 {
					return assert.AnError
				}

				return nil
			},
		},
		{
			name: "missing JIRA section",
			configYAML: `
environments:
  testnet:
    network_types:
      - testnet
`,
			expectError: true,
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

			// Create temporary file
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
