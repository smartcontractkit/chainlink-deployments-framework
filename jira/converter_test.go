package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJiraToStruct(t *testing.T) { //nolint:paralleltest // Cannot use t.Parallel() due to os.Chdir() usage
	// Save original working directory and environment
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
      jira_field: "status.name"
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

	// Set up environment variable
	originalToken := os.Getenv("JIRA_TOKEN_EXEMPLAR")
	os.Setenv("JIRA_TOKEN_EXEMPLAR", "test-token-123")
	defer func() {
		if originalToken == "" {
			os.Unsetenv("JIRA_TOKEN_EXEMPLAR")
		} else {
			os.Setenv("JIRA_TOKEN_EXEMPLAR", originalToken)
		}
	}()

	// Define test struct
	type TestStruct struct {
		Summary     string `json:"summary"`
		Status      string `json:"status"`
		CustomField string `json:"custom_field"`
	}

	// Create a test server that mocks JIRA API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		// Check authentication
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Errorf("Expected basic auth")
		}
		if username != "testuser" {
			t.Errorf("Expected username 'testuser', got %s", username)
		}
		if password != "test-token-123" {
			t.Errorf("Expected password 'test-token-123', got %s", password)
		}

		// Check URL path
		expectedPath := "/rest/api/2/issue/TEST-123"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Check fields parameter (order may vary due to map iteration)
		fields := r.URL.Query().Get("fields")
		expectedFields := []string{"summary", "status.name", "customfield_10001"}
		actualFields := strings.Split(fields, ",")

		if len(actualFields) != len(expectedFields) {
			t.Errorf("Expected %d fields, got %d: %s", len(expectedFields), len(actualFields), fields)
		}

		// Check that all expected fields are present
		for _, expected := range expectedFields {
			found := false
			for _, actual := range actualFields {
				if actual == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected field '%s' not found in: %s", expected, fields)
			}
		}

		// Return mock JIRA response
		response := JiraIssue{
			Key: "TEST-123",
			Fields: map[string]any{
				"summary": "Test Issue Summary",
				"status": map[string]any{
					"name": "In Progress",
					"id":   "3",
				},
				"customfield_10001": "Custom Field Value",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Override the base URL in the config by creating a new config file
	configContentWithServer := strings.Replace(configContent, "https://example.atlassian.net", server.URL, 1)
	if err := os.WriteFile(configPath, []byte(configContentWithServer), 0600); err != nil {
		t.Fatalf("Failed to write updated config file: %v", err)
	}

	tests := []struct {
		name          string
		issueKey      string
		expectError   bool
		errorContains string
		validate      func(TestStruct) error
	}{
		{
			name:        "successful conversion",
			issueKey:    "TEST-123",
			expectError: false,
			validate: func(result TestStruct) error {
				if result.Summary != "Test Issue Summary" {
					return fmt.Errorf("Expected Summary 'Test Issue Summary', got '%s'", result.Summary)
				}
				if result.Status != "In Progress" {
					return fmt.Errorf("Expected Status 'In Progress', got '%s'", result.Status)
				}
				if result.CustomField != "Custom Field Value" {
					return fmt.Errorf("Expected CustomField 'Custom Field Value', got '%s'", result.CustomField)
				}

				return nil
			},
		},
		{
			name:          "empty issue key",
			issueKey:      "",
			expectError:   true,
			errorContains: "issue_key is required",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // Cannot use t.Parallel() due to shared test server
		t.Run(tt.name, func(t *testing.T) {
			result, err := JiraToStruct[TestStruct](tt.issueKey)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}

				return
			}

			require.NoError(t, err)

			if tt.validate != nil {
				require.NoError(t, tt.validate(result))
			}
		})
	}

	// Restore original working directory
	if err := os.Chdir(originalCwd); err != nil {
		t.Errorf("Failed to restore original working directory: %v", err)
	}
}

func TestJiraToStruct_ErrorCases(t *testing.T) { //nolint:paralleltest // Cannot use t.Parallel() due to os.Chdir() usage
	// Save original working directory and environment
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

	// Change to a directory within the exemplar domain
	testDir := filepath.Join(exemplarDir, "test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	// Set up environment variable
	originalToken := os.Getenv("JIRA_TOKEN_EXEMPLAR")
	os.Setenv("JIRA_TOKEN_EXEMPLAR", "test-token-123")
	defer func() {
		if originalToken == "" {
			os.Unsetenv("JIRA_TOKEN_EXEMPLAR")
		} else {
			os.Setenv("JIRA_TOKEN_EXEMPLAR", originalToken)
		}
	}()

	// Define test struct
	type TestStruct struct {
		Summary string `json:"summary"`
	}

	tests := []struct {
		name          string
		issueKey      string
		setup         func() error
		expectError   bool
		errorContains string
	}{
		{
			name:     "missing JIRA token",
			issueKey: "TEST-123",
			setup: func() error {
				// Create valid config file first
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
`
				configPath := filepath.Join(configDir, "domain.yaml")
				if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
					return err
				}
				// Remove the token
				os.Unsetenv("JIRA_TOKEN_EXEMPLAR")

				return nil
			},
			expectError:   true,
			errorContains: "JIRA_TOKEN environment variable is required",
		},
		{
			name:     "missing config file",
			issueKey: "TEST-123",
			setup: func() error {
				// Ensure token is set so we get the config error, not token error
				os.Setenv("JIRA_TOKEN_EXEMPLAR", "test-token-123")
				// Remove any existing config file
				configPath := filepath.Join(configDir, "domain.yaml")
				os.Remove(configPath)

				return nil
			},
			expectError:   true,
			errorContains: "failed to load domain JIRA config",
		},
		{
			name:     "invalid config file",
			issueKey: "TEST-123",
			setup: func() error {
				// Create invalid config file
				configPath := filepath.Join(configDir, "domain.yaml")
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
invalid: [unclosed
`

				return os.WriteFile(configPath, []byte(invalidContent), 0600)
			},
			expectError:   true,
			errorContains: "failed to load domain JIRA config",
		},
		{
			name:     "JIRA API error",
			issueKey: "TEST-123",
			setup: func() error {
				// Create valid config file
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
`
				configPath := filepath.Join(configDir, "domain.yaml")
				if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
					return err
				}
				// Ensure token is set
				os.Setenv("JIRA_TOKEN_EXEMPLAR", "test-token-123")

				return nil
			},
			expectError:   true,
			errorContains: "failed to fetch JIRA issue",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // Cannot use t.Parallel() due to shared test server
		t.Run(tt.name, func(t *testing.T) {
			// Save original working directory
			testCwd, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current working directory: %v", err)
			}

			// Setup test
			if setupErr := tt.setup(); setupErr != nil {
				t.Fatalf("Test setup failed: %v", setupErr)
			}

			result, err := JiraToStruct[TestStruct](tt.issueKey)

			// Restore original working directory
			if restoreErr := os.Chdir(testCwd); restoreErr != nil {
				t.Errorf("Failed to restore original working directory: %v", restoreErr)
			}

			if !tt.expectError {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)

			if tt.errorContains != "" {
				assert.Contains(t, err.Error(), tt.errorContains)
			}

			// Check that result is zero value
			var zero TestStruct
			assert.Equal(t, zero, result)
		})
	}

	// Restore original working directory
	if err := os.Chdir(originalCwd); err != nil {
		t.Errorf("Failed to restore original working directory: %v", err)
	}
}

func TestJiraToStruct_NoDomain(t *testing.T) { //nolint:paralleltest // Cannot use t.Parallel() due to os.Chdir() usage
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

	// Define test struct
	type TestStruct struct {
		Summary string `json:"summary"`
	}

	result, err := JiraToStruct[TestStruct]("TEST-123")

	require.Error(t, err)

	// Check that result is zero value
	var zero TestStruct
	assert.Equal(t, zero, result)
	assert.Contains(t, err.Error(), "failed to load domain JIRA config")

	// Restore original working directory
	if err := os.Chdir(originalCwd); err != nil {
		t.Errorf("Failed to restore original working directory: %v", err)
	}
}
