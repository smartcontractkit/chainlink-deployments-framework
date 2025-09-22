package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			name:        "empty token should fail",
			baseURL:     "https://example.atlassian.net",
			username:    "user@example.com",
			token:       "",
			expectError: true,
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
			client, err := NewClient(tt.baseURL, tt.username, tt.token)

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
	client, err := NewClient(server.URL, "testuser", "testtoken")
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
	}{
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
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client, err := NewClient(server.URL, "testuser", "testtoken")
			require.NoError(t, err)

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
