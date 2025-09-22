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
)

func TestJiraToStruct(t *testing.T) {
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
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
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
	originalToken := os.Getenv("EXEMPLAR_JIRA_TOKEN")
	os.Setenv("EXEMPLAR_JIRA_TOKEN", "test-token-123")
	defer func() {
		if originalToken == "" {
			os.Unsetenv("EXEMPLAR_JIRA_TOKEN")
		} else {
			os.Setenv("EXEMPLAR_JIRA_TOKEN", originalToken)
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
		if r.Method != "GET" {
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
			Fields: map[string]interface{}{
				"summary": "Test Issue Summary",
				"status": map[string]interface{}{
					"name": "In Progress",
					"id":   "3",
				},
				"customfield_10001": "Custom Field Value",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Override the base URL in the config by creating a new config file
	configContentWithServer := strings.Replace(configContent, "https://example.atlassian.net", server.URL, 1)
	if err := os.WriteFile(configPath, []byte(configContentWithServer), 0644); err != nil {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := JiraToStruct[TestStruct](tt.issueKey)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				if err := tt.validate(result); err != nil {
					t.Errorf("Validation failed: %v", err)
				}
			}
		})
	}

	// Restore original working directory
	if err := os.Chdir(originalCwd); err != nil {
		t.Errorf("Failed to restore original working directory: %v", err)
	}
}

func TestJiraToStruct_ErrorCases(t *testing.T) {
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
	originalToken := os.Getenv("EXEMPLAR_JIRA_TOKEN")
	os.Setenv("EXEMPLAR_JIRA_TOKEN", "test-token-123")
	defer func() {
		if originalToken == "" {
			os.Unsetenv("EXEMPLAR_JIRA_TOKEN")
		} else {
			os.Setenv("EXEMPLAR_JIRA_TOKEN", originalToken)
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
				if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
					return err
				}
				// Remove the token
				os.Unsetenv("EXEMPLAR_JIRA_TOKEN")
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
				os.Setenv("EXEMPLAR_JIRA_TOKEN", "test-token-123")
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
				return os.WriteFile(configPath, []byte(invalidContent), 0644)
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
				if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
					return err
				}
				// Ensure token is set
				os.Setenv("EXEMPLAR_JIRA_TOKEN", "test-token-123")
				return nil
			},
			expectError:   true,
			errorContains: "failed to fetch JIRA issue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test
			if err := tt.setup(); err != nil {
				t.Fatalf("Test setup failed: %v", err)
			}

			result, err := JiraToStruct[TestStruct](tt.issueKey)

			if !tt.expectError {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			if err == nil {
				t.Errorf("Expected error but got none")
				return
			}

			if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
			}

			// Check that result is zero value
			var zero TestStruct
			if result != zero {
				t.Errorf("Expected zero value result, got %v", result)
			}
		})
	}

	// Restore original working directory
	if err := os.Chdir(originalCwd); err != nil {
		t.Errorf("Failed to restore original working directory: %v", err)
	}
}

func TestJiraToStruct_NoDomain(t *testing.T) {
	// Save original working directory
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	// Create a temporary directory without domains structure
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	// Define test struct
	type TestStruct struct {
		Summary string `json:"summary"`
	}

	result, err := JiraToStruct[TestStruct]("TEST-123")

	if err == nil {
		t.Errorf("Expected error but got none")
	}

	// Check that result is zero value
	var zero TestStruct
	if result != zero {
		t.Errorf("Expected zero value result, got %v", result)
	}

	if !strings.Contains(err.Error(), "failed to load domain JIRA config") {
		t.Errorf("Expected domain config error, got: %s", err.Error())
	}

	// Restore original working directory
	if err := os.Chdir(originalCwd); err != nil {
		t.Errorf("Failed to restore original working directory: %v", err)
	}
}
