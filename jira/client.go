package jira

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// Client represents a JIRA API client
type Client struct {
	baseURL    string
	username   string
	token      string
	httpClient *http.Client
}

// JiraIssue represents a JIRA issue response
type JiraIssue struct {
	Key    string         `json:"key"`
	Fields map[string]any `json:"fields"`
}

// NewClient creates a new JIRA client with the provided authentication token
func NewClient(baseURL, username, token string) (*Client, error) {
	if token == "" {
		return nil, errors.New("JIRA token is required")
	}

	return &Client{
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		username: username,
		token:    token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// NewClientFromDomain creates a JIRA client using domain configuration and environment variables
func NewClientFromDomain(dom fdomain.Domain) (*Client, error) {
	jiraConfig, err := config.LoadJiraConfig(dom)
	if err != nil {
		return nil, fmt.Errorf("failed to load domain JIRA config: %w", err)
	}

	domainNameUpper := strings.ToUpper(dom.Key())
	token := os.Getenv("JIRA_TOKEN_" + domainNameUpper)
	if token == "" {
		return nil, fmt.Errorf("JIRA_TOKEN_%s environment variable is required", domainNameUpper)
	}

	client, err := NewClient(jiraConfig.Connection.BaseURL, jiraConfig.Connection.Username, token)
	if err != nil {
		return nil, fmt.Errorf("failed to create JIRA client: %w", err)
	}

	return client, nil
}

// jiraIssueKeyPattern matches standard JIRA issue keys (PROJECT-NUMBER format)
var jiraIssueKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]*-[1-9][0-9]*$`)

// GetIssue fetches a JIRA issue by key. If fields is empty, Jira returns all default fields.
func (c *Client) GetIssue(issueKey string, fields []string) (*JiraIssue, error) {
	// Validate JIRA issue key format
	if issueKey == "" {
		return nil, errors.New("issue key cannot be empty")
	}
	if !jiraIssueKeyPattern.MatchString(issueKey) {
		return nil, fmt.Errorf("invalid JIRA issue key format: %q (expected format: PROJECT-NUMBER, e.g., ABC-123)", issueKey)
	}

	base, err := url.JoinPath(c.baseURL, "rest", "api", "2", "issue", url.PathEscape(issueKey))
	if err != nil {
		return nil, fmt.Errorf("failed to build request URL: %w", err)
	}

	reqURL := base
	if len(fields) > 0 {
		q := url.Values{}
		q.Set("fields", strings.Join(fields, ","))
		reqURL = base + "?" + q.Encode()
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set basic auth header
	req.SetBasicAuth(c.username, c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		const maxErrBody = 4096 // limit error body to 4KB
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBody))

		return nil, fmt.Errorf("JIRA API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var issue JiraIssue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse JIRA response: %w", err)
	}

	return &issue, nil
}
