package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func TestNewClient(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		baseURL     string
		username    string
		token       string
		expectError bool
	}{
		{
			name:        "valid client creation",
			baseURL:     "https://example.atlassian.net",
			username:    "user@example.com",
			token:       "valid-token",
			expectError: false,
		},
		{
			name:        "empty token allowed for lazy initialization",
			baseURL:     "https://example.atlassian.net",
			username:    "user@example.com",
			token:       "",
			expectError: false,
		},
		{
			name:        "baseURL with trailing slash should be trimmed",
			baseURL:     "https://example.atlassian.net/",
			username:    "user@example.com",
			token:       "valid-token",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, err := NewClient(tt.baseURL, tt.username, tt.token, "")

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, client)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, client)

			expectedBaseURL := "https://example.atlassian.net"
			assert.Equal(t, expectedBaseURL, client.baseURL)
			assert.Equal(t, tt.username, client.username)
			assert.Equal(t, tt.token, client.token)
			assert.NotNil(t, client.httpClient)
			assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
		})
	}
}

func TestClient_GetIssue(t *testing.T) {
	t.Parallel()
	// Create a test server
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
		if password != "testtoken" {
			t.Errorf("Expected password 'testtoken', got %s", password)
		}

		// Check Accept header
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected Accept: application/json, got %s", r.Header.Get("Accept"))
		}

		// Check URL path
		expectedPath := "/rest/api/2/issue/TEST-123"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Check fields parameter if provided
		fields := r.URL.Query().Get("fields")
		switch fields {
		case "":
			// No fields specified, return all default fields
			response := JiraIssue{
				Key: "TEST-123",
				Fields: map[string]any{
					"summary":     "Test Issue",
					"description": "This is a test issue description",
					"status": map[string]any{
						"name": "To Do",
						"id":   "1",
					},
					"assignee": map[string]any{
						"displayName":  "John Doe",
						"emailAddress": "john@example.com",
					},
					"reporter": map[string]any{
						"displayName":  "Jane Smith",
						"emailAddress": "jane@example.com",
					},
					"created": "2023-01-01T10:00:00.000+0000",
					"updated": "2023-01-01T10:00:00.000+0000",
					"labels": []any{
						"bug",
						"urgent",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case "summary,status":
			// Specific fields requested - return only those fields
			response := JiraIssue{
				Key: "TEST-123",
				Fields: map[string]any{
					"summary": "Test Issue",
					"status": map[string]any{
						"name": "To Do",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		default:
			t.Errorf("Unexpected fields parameter: %s", fields)
		}
	}))
	defer server.Close()

	// Create client
	client, err := NewClient(server.URL, "testuser", "testtoken", "")
	require.NoError(t, err)

	tests := []struct {
		name        string
		issueKey    string
		fields      []string
		expectError bool
	}{
		{
			name:        "get issue without fields",
			issueKey:    "TEST-123",
			fields:      nil,
			expectError: false,
		},
		{
			name:        "get issue with specific fields",
			issueKey:    "TEST-123",
			fields:      []string{"summary", "status"},
			expectError: false,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // Cannot use t.Parallel() due to shared test server
		t.Run(tt.name, func(t *testing.T) {
			issue, err := client.GetIssue(tt.issueKey, tt.fields)

			if tt.expectError {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, issue)
			assert.Equal(t, "TEST-123", issue.Key)
			assert.NotNil(t, issue.Fields)

			summary, ok := issue.Fields["summary"]
			assert.True(t, ok, "Expected summary field")
			assert.Equal(t, "Test Issue", summary)
		})
	}
}

func TestClient_GetIssue_ErrorCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		issueKey       string
		fields         []string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
		errorContains  string
		emptyToken     bool
		domainName     string
	}{
		{
			name:          "empty token without domain name",
			issueKey:      "TEST-123",
			fields:        nil,
			expectError:   true,
			errorContains: "JIRA token is required but not set",
			emptyToken:    true,
			domainName:    "",
		},
		{
			name:          "empty token with domain name",
			issueKey:      "TEST-123",
			fields:        nil,
			expectError:   true,
			errorContains: "Please set JIRA_TOKEN_TESTDOMAIN environment variable",
			emptyToken:    true,
			domainName:    "TESTDOMAIN",
		},
		{
			name:          "empty issue key",
			issueKey:      "",
			fields:        nil,
			expectError:   true,
			errorContains: "issue key cannot be empty",
		},
		{
			name:          "invalid issue key format - no dash",
			issueKey:      "ABC123",
			fields:        nil,
			expectError:   true,
			errorContains: "invalid JIRA issue key format",
		},
		{
			name:          "invalid issue key format - lowercase project",
			issueKey:      "abc-123",
			fields:        nil,
			expectError:   true,
			errorContains: "invalid JIRA issue key format",
		},
		{
			name:          "invalid issue key format - zero number",
			issueKey:      "ABC-0",
			fields:        nil,
			expectError:   true,
			errorContains: "invalid JIRA issue key format",
		},
		{
			name:          "invalid issue key format - leading zero",
			issueKey:      "ABC-01",
			fields:        nil,
			expectError:   true,
			errorContains: "invalid JIRA issue key format",
		},
		{
			name:     "server returns 404",
			issueKey: "NOTFOUND-123",
			fields:   nil,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("Issue not found"))
			},
			expectError:   true,
			errorContains: "status 404",
		},
		{
			name:     "server returns 500",
			issueKey: "ERROR-123",
			fields:   nil,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("Internal server error"))
			},
			expectError:   true,
			errorContains: "status 500",
		},
		{
			name:     "invalid JSON response",
			issueKey: "INVALID-123",
			fields:   nil,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte("invalid json"))
			},
			expectError:   true,
			errorContains: "failed to parse JIRA response",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // Cannot use t.Parallel() due to shared test server
		t.Run(tt.name, func(t *testing.T) {
			var client *Client
			var err error

			// For validation tests, we don't need a server
			if tt.serverResponse == nil {
				token := "testtoken"
				if tt.emptyToken {
					token = ""
				}
				client, err = NewClient("https://example.com", "testuser", token, tt.domainName)
				require.NoError(t, err)
			} else {
				server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
				defer server.Close()

				token := "testtoken"
				if tt.emptyToken {
					token = ""
				}
				client, err = NewClient(server.URL, "testuser", token, tt.domainName)
				require.NoError(t, err)
			}

			issue, err := client.GetIssue(tt.issueKey, tt.fields)

			if !tt.expectError {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)

			if tt.errorContains != "" {
				assert.Contains(t, err.Error(), tt.errorContains)
			}
			assert.Nil(t, issue)
		})
	}
}

func TestNewClientFromDomain(t *testing.T) { //nolint:paralleltest // Cannot use t.Parallel() because we manipulate environment variables
	tests := []struct {
		name          string
		setupDomain   func(t *testing.T) fdomain.Domain
		setupEnv      func(t *testing.T)
		expectError   bool
		errorContains string
		validate      func(*Client) error
	}{
		{
			name: "successful client creation",
			setupDomain: func(t *testing.T) fdomain.Domain {
				t.Helper()
				dom := setupTestDomain(t)
				writeJiraDomainConfig(t, dom, "https://example.atlassian.net")

				return dom
			},
			setupEnv: func(t *testing.T) {
				t.Helper()
				os.Setenv("JIRA_TOKEN_EXEMPLAR", "test-token-123")
			},
			expectError: false,
			validate: func(client *Client) error {
				if client.baseURL != "https://example.atlassian.net" {
					return fmt.Errorf("expected baseURL 'https://example.atlassian.net', got '%s'", client.baseURL)
				}
				if client.username != "testuser" {
					return fmt.Errorf("expected username 'testuser', got '%s'", client.username)
				}
				if client.token != "test-token-123" {
					return fmt.Errorf("expected token 'test-token-123', got '%s'", client.token)
				}

				return nil
			},
		},
		{
			name: "missing JIRA token allowed for lazy initialization",
			setupDomain: func(t *testing.T) fdomain.Domain {
				t.Helper()
				dom := setupTestDomain(t)
				writeJiraDomainConfig(t, dom, "https://example.atlassian.net")

				return dom
			},
			setupEnv: func(t *testing.T) {
				t.Helper()
				// Don't set the token
			},
			expectError: false,
			validate: func(client *Client) error {
				if client.baseURL != "https://example.atlassian.net" {
					return fmt.Errorf("expected baseURL 'https://example.atlassian.net', got '%s'", client.baseURL)
				}
				if client.username != "testuser" {
					return fmt.Errorf("expected username 'testuser', got '%s'", client.username)
				}
				if client.token != "" {
					return fmt.Errorf("expected empty token, got '%s'", client.token)
				}
				if client.domainName != "EXEMPLAR" {
					return fmt.Errorf("expected domainName 'EXEMPLAR', got '%s'", client.domainName)
				}

				return nil
			},
		},
		{
			name: "missing config file",
			setupDomain: func(t *testing.T) fdomain.Domain {
				t.Helper()
				// Don't write config file

				return setupTestDomain(t)
			},
			setupEnv: func(t *testing.T) {
				t.Helper()
				os.Setenv("JIRA_TOKEN_EXEMPLAR", "test-token-123")
			},
			expectError:   true,
			errorContains: "failed to load domain JIRA config",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // Cannot use t.Parallel() because we manipulate environment variables
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

			client, err := NewClientFromDomain(dom)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, client)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, client)

			if tt.validate != nil {
				require.NoError(t, tt.validate(client))
			}
		})
	}
}
