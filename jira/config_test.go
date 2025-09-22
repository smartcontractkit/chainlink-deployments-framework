package jira

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d fields, got %d", len(tt.expected), len(result))
				return
			}

			// Convert to map for easier comparison
			resultMap := make(map[string]bool)
			for _, field := range result {
				resultMap[field] = true
			}

			for _, expectedField := range tt.expected {
				if !resultMap[expectedField] {
					t.Errorf("Expected field %s not found in result", expectedField)
				}
			}
		})
	}
}

func TestDetectCurrentDomain(t *testing.T) {
	t.Parallel()
	// Save original working directory
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	// Create a temporary directory structure
	tempDir := t.TempDir()

	// Create domains structure
	domainsDir := filepath.Join(tempDir, "domains")
	exemplarDir := filepath.Join(domainsDir, "exemplar")
	nestedDir := filepath.Join(exemplarDir, "some", "nested", "path")

	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory structure: %v", err)
	}

	// Create a directory structure that doesn't contain domains
	otherDir := filepath.Join(tempDir, "some", "other", "path")
	if err := os.MkdirAll(otherDir, 0755); err != nil {
		t.Fatalf("Failed to create other directory structure: %v", err)
	}

	tests := []struct {
		name           string
		workingDir     string
		expectError    bool
		expectedDomain string
	}{
		{
			name:           "detect domain from nested path",
			workingDir:     nestedDir,
			expectError:    false,
			expectedDomain: "exemplar",
		},
		{
			name:           "detect domain from direct domain path",
			workingDir:     exemplarDir,
			expectError:    false,
			expectedDomain: "exemplar",
		},
	}

	// Test case for no domains directory found - use a separate temp directory
	t.Run("no domains directory found", func(t *testing.T) {
		// Create a completely separate temporary directory without domains
		separateTempDir := t.TempDir()
		otherPath := filepath.Join(separateTempDir, "some", "other", "path")
		if err := os.MkdirAll(otherPath, 0755); err != nil {
			t.Fatalf("Failed to create separate directory structure: %v", err)
		}

		// Change to the separate directory
		if err := os.Chdir(otherPath); err != nil {
			t.Fatalf("Failed to change to separate test directory: %v", err)
		}

		domain, err := detectCurrentDomain()

		if err == nil {
			t.Errorf("Expected error but got none, domain: %s", domain)
		}

		if !strings.Contains(err.Error(), "could not detect domain") {
			t.Errorf("Expected domain detection error, got: %s", err.Error())
		}
	})

	// Run the main tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Change to test working directory
			if err := os.Chdir(tt.workingDir); err != nil {
				t.Fatalf("Failed to change to test directory: %v", err)
			}

			domain, err := detectCurrentDomain()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none, domain: %s", domain)
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check that the returned domain contains the expected domain name
			if !strings.Contains(domain, tt.expectedDomain) {
				t.Errorf("Expected domain to contain '%s', got '%s'", tt.expectedDomain, domain)
			}

			// Check that the returned path ends with the domain name
			if !strings.HasSuffix(domain, tt.expectedDomain) {
				t.Errorf("Expected domain path to end with '%s', got '%s'", tt.expectedDomain, domain)
			}
		})
	}

	// Restore original working directory
	if err := os.Chdir(originalCwd); err != nil {
		t.Errorf("Failed to restore original working directory: %v", err)
	}
}

func TestLoadDomainJiraConfig(t *testing.T) {
	t.Parallel()
	// Save original working directory
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	// Create a temporary directory structure
	tempDir := t.TempDir()

	// Create domains structure
	domainsDir := filepath.Join(tempDir, "domains")
	exemplarDir := filepath.Join(domainsDir, "exemplar")
	configDir := filepath.Join(exemplarDir, ".config")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory structure: %v", err)
	}

	// Create a valid domain config file with JIRA configuration
	configContent := `
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
`

	configPath := filepath.Join(configDir, "domain.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Change to a directory within the exemplar domain
	testDir := filepath.Join(exemplarDir, "test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	tests := []struct {
		name        string
		setup       func() error
		expectError bool
		checkConfig func(*JiraConfig) error
	}{
		{
			name: "load valid config",
			setup: func() error {
				// Config file already created above
				return nil
			},
			expectError: false,
			checkConfig: func(config *JiraConfig) error {
				if config.Domain != "exemplar" {
					return fmt.Errorf("Expected domain 'exemplar', got '%s'", config.Domain)
				}
				if config.Connection.BaseURL != "https://example.atlassian.net" {
					return fmt.Errorf("Expected base_url 'https://example.atlassian.net', got '%s'", config.Connection.BaseURL)
				}
				if config.Connection.Project != "TEST" {
					return fmt.Errorf("Expected project 'TEST', got '%s'", config.Connection.Project)
				}
				if config.Connection.Username != "testuser" {
					return fmt.Errorf("Expected username 'testuser', got '%s'", config.Connection.Username)
				}
				if len(config.FieldMaps) != 3 {
					return fmt.Errorf("Expected 3 field maps, got %d", len(config.FieldMaps))
				}

				return nil
			},
		},
		{
			name: "config file not found",
			setup: func() error {
				// Remove the config file
				return os.Remove(configPath)
			},
			expectError: true,
		},
		{
			name: "invalid YAML config",
			setup: func() error {
				// Write invalid YAML
				invalidContent := `
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
invalid: [unclosed
`

				return os.WriteFile(configPath, []byte(invalidContent), 0600)
			},
			expectError: true,
		},
		{
			name: "missing JIRA configuration",
			setup: func() error {
				// Write valid YAML but without JIRA section
				invalidContent := `
environments:
  testnet:
    network_types:
      - testnet
`

				return os.WriteFile(configPath, []byte(invalidContent), 0600)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Setup test
			if err := tt.setup(); err != nil {
				t.Fatalf("Test setup failed: %v", err)
			}

			config, err := loadDomainJiraConfig()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if config != nil {
					t.Errorf("Expected nil config but got %v", config)
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if config == nil {
				t.Errorf("Expected config but got nil")
				return
			}

			if tt.checkConfig != nil {
				if err := tt.checkConfig(config); err != nil {
					t.Errorf("Config validation failed: %v", err)
				}
			}
		})
	}

	// Restore original working directory
	if err := os.Chdir(originalCwd); err != nil {
		t.Errorf("Failed to restore original working directory: %v", err)
	}
}

func TestLoadDomainJiraConfig_NoDomain(t *testing.T) {
	t.Parallel()
	// Save original working directory
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	// Create a temporary directory without domains structure
	tempDir := t.TempDir()
	if err = os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	config, err := loadDomainJiraConfig()

	if err == nil {
		t.Errorf("Expected error but got none")
	}

	if config != nil {
		t.Errorf("Expected nil config but got %v", config)
	}

	if !strings.Contains(err.Error(), "failed to detect domain") {
		t.Errorf("Expected domain detection error, got: %s", err.Error())
	}

	// Restore original working directory
	if err := os.Chdir(originalCwd); err != nil {
		t.Errorf("Failed to restore original working directory: %v", err)
	}
}
