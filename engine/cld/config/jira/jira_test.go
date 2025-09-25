package jira

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJiraConfig_GetJiraFields(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		config   Config
		expected []string
	}{
		{
			name: "single field mapping",
			config: Config{
				FieldMaps: map[string]FieldMapping{
					"summary": {JiraField: "summary"},
				},
			},
			expected: []string{"summary"},
		},
		{
			name: "multiple field mappings",
			config: Config{
				FieldMaps: map[string]FieldMapping{
					"summary":      {JiraField: "summary"},
					"status":       {JiraField: "status"},
					"custom_field": {JiraField: "customfield_10001"},
				},
			},
			expected: []string{"summary", "status", "customfield_10001"},
		},
		{
			name: "empty field mappings",
			config: Config{
				FieldMaps: map[string]FieldMapping{},
			},
			expected: []string{},
		},
		{
			name: "nil field mappings",
			config: Config{
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

			// Convert result to map for easier comparison
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

func TestJiraConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				Connection: JiraConnectionConfig{
					BaseURL:  "https://example.atlassian.net",
					Project:  "TEST",
					Username: "testuser",
				},
			},
			expectError: false,
		},
		{
			name: "missing base_url",
			config: Config{
				Connection: JiraConnectionConfig{
					Project:  "TEST",
					Username: "testuser",
				},
			},
			expectError: true,
			errorMsg:    "connection.base_url is required",
		},
		{
			name: "missing project",
			config: Config{
				Connection: JiraConnectionConfig{
					BaseURL:  "https://example.atlassian.net",
					Username: "testuser",
				},
			},
			expectError: true,
			errorMsg:    "connection.project is required",
		},
		{
			name: "missing username",
			config: Config{
				Connection: JiraConnectionConfig{
					BaseURL: "https://example.atlassian.net",
					Project: "TEST",
				},
			},
			expectError: true,
			errorMsg:    "connection.username is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
