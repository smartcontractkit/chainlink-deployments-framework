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

	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func TestJiraToStruct(t *testing.T) { //nolint:paralleltest // Cannot use t.Parallel() because we have a shared test server
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

	for _, tt := range tests { //nolint:paralleltest // Cannot use t.Parallel() because of shared test server
		t.Run(tt.name, func(t *testing.T) {
			// Set up test domain and config with the server URL
			dom := setupTestDomain(t)
			writeJiraDomainConfig(t, dom, server.URL)

			// Create client and call JiraToStruct
			client, clientErr := NewClientFromDomain(dom)
			var result TestStruct
			var err error
			if clientErr != nil {
				result, err = *new(TestStruct), clientErr
			} else {
				result, err = JiraToStruct[TestStruct](client, dom, tt.issueKey)
			}

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
}

func TestJiraToStruct_ErrorCases(t *testing.T) {
	t.Parallel()

	// Define test struct
	type TestStruct struct {
		Summary string `json:"summary"`
	}

	tests := []struct {
		name          string
		issueKey      string
		setupDomain   func(t *testing.T) fdomain.Domain
		setupEnv      func(t *testing.T)
		expectError   bool
		errorContains string
	}{
		{
			name:     "missing JIRA token",
			issueKey: "TEST-123",
			setupDomain: func(t *testing.T) fdomain.Domain {
				t.Helper()
				dom := setupTestDomain(t)
				writeJiraDomainConfig(t, dom, "https://example.atlassian.net")

				return dom
			},
			setupEnv: func(t *testing.T) {
				t.Helper()

				// Ensure the token is not set
				os.Unsetenv("JIRA_TOKEN_EXEMPLAR")
			},
			expectError:   true,
			errorContains: "Please set JIRA_TOKEN_EXEMPLAR environment variable",
		},
		{
			name:     "missing config file",
			issueKey: "TEST-123",
			setupDomain: func(t *testing.T) fdomain.Domain {
				// Don't write config file
				t.Helper()

				return setupTestDomain(t)
			},
			setupEnv: func(t *testing.T) {
				t.Helper()

				os.Setenv("JIRA_TOKEN_EXEMPLAR", "test-token-123")
			},
			expectError:   true,
			errorContains: "failed to load domain JIRA config",
		},
		{
			name:     "JIRA API error",
			issueKey: "TEST-123",
			setupDomain: func(t *testing.T) fdomain.Domain {
				t.Helper()

				dom := setupTestDomain(t)
				// Use invalid URL that will cause connection error
				writeJiraDomainConfig(t, dom, "https://invalid-jira-url.example.com")

				return dom
			},
			setupEnv: func(t *testing.T) {
				t.Helper()

				os.Setenv("JIRA_TOKEN_EXEMPLAR", "test-token-123")
			},
			expectError:   true,
			errorContains: "failed to fetch JIRA issue",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // Cannot use t.Parallel() due to shared environment variables
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			originalToken := os.Getenv("JIRA_TOKEN_EXEMPLAR")
			defer func() {
				if originalToken == "" {
					os.Unsetenv("JIRA_TOKEN_EXEMPLAR")
				} else {
					os.Setenv("JIRA_TOKEN_EXEMPLAR", originalToken)
				}
			}()

			if tt.setupEnv != nil {
				tt.setupEnv(t)
			}

			// Setup domain
			dom := tt.setupDomain(t)

			// Create client and call JiraToStruct
			client, clientErr := NewClientFromDomain(dom)
			var result TestStruct
			var err error
			if clientErr != nil {
				result, err = *new(TestStruct), clientErr
			} else {
				result, err = JiraToStruct[TestStruct](client, dom, tt.issueKey)
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
}

func TestJiraToStruct_EmptyIssueKey(t *testing.T) { //nolint:paralleltest // Cannot use t.Parallel() because we manipulate environment variables
	// Define test struct
	type TestStruct struct {
		Summary string `json:"summary"`
	}

	// Set up test domain and config
	dom := setupTestDomain(t)
	writeJiraDomainConfig(t, dom, "https://example.atlassian.net")

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

	// Create client and call JiraToStruct
	client, clientErr := NewClientFromDomain(dom)
	var result TestStruct
	var err error
	if clientErr != nil {
		result, err = *new(TestStruct), clientErr
	} else {
		result, err = JiraToStruct[TestStruct](client, dom, "")
	}

	require.Error(t, err)

	// Check that result is zero value
	var zero TestStruct
	assert.Equal(t, zero, result)
	assert.Contains(t, err.Error(), "issue_key is required")
}

// setupTestDomain sets up a minimal domain structure with a .config directory and returns the domain
func setupTestDomain(t *testing.T) fdomain.Domain {
	t.Helper()

	// Create a temporary directory structure for testing
	rootDir := t.TempDir()
	domainKey := "exemplar"

	// Set up minimal domain structure
	domainDir := filepath.Join(rootDir, domainKey)
	require.NoError(t, os.MkdirAll(domainDir, 0755))

	// Create .config directory structure
	configDir := filepath.Join(domainDir, ".config")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	return fdomain.NewDomain(rootDir, domainKey)
}

// writeJiraDomainConfig writes a JIRA domain config file to the domain's .config directory
func writeJiraDomainConfig(t *testing.T, dom fdomain.Domain, baseURL string) {
	t.Helper()

	configContent := fmt.Sprintf(`
environments:
  testnet:
    network_types:
      - testnet

jira:
  connection:
    base_url: "%s"
    project: "TEST"
    username: "testuser"
  field_maps:
    summary:
      jira_field: "summary"
    status:
      jira_field: "status.name"
    custom_field:
      jira_field: "customfield_10001"
`, baseURL)

	configPath := dom.ConfigDomainFilePath()
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)
}
