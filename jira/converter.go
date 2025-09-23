package jira

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// JiraToStruct is the main function that domains use to convert JIRA issues to config structs
// It loads the specified domain's JIRA schema and maps the issue fields
func JiraToStruct[T any](dom fdomain.Domain, issueKey string) (T, error) {
	var zero T

	if issueKey == "" {
		return zero, errors.New("issue_key is required")
	}

	jiraConfig, err := config.LoadJiraConfig(dom)
	if err != nil {
		return zero, fmt.Errorf("failed to load domain JIRA config: %w", err)
	}

	// Extract all JIRA field names from the field maps
	var fieldsToFetch = jiraConfig.GetJiraFields()

	var domainNameUpper = strings.ToUpper(dom.Key())

	// 2. Get JIRA token from environment variable
	token := os.Getenv("JIRA_TOKEN_" + domainNameUpper)
	if token == "" {
		return zero, fmt.Errorf("%s_JIRA_TOKEN environment variable is required", domainNameUpper)
	}

	// 3. Create JIRA client
	client, err := NewClient(jiraConfig.Connection.BaseURL, jiraConfig.Connection.Username, token)
	if err != nil {
		return zero, fmt.Errorf("failed to create JIRA client: %w", err)
	}

	// 4. Fetch issue from JIRA
	issue, err := client.GetIssue(issueKey, fieldsToFetch)
	if err != nil {
		return zero, fmt.Errorf("failed to fetch JIRA issue %s: %w", issueKey, err)
	}

	// 5. Map JIRA fields to target struct
	result, err := mapFieldsToStruct[T](issue, jiraConfig)
	if err != nil {
		return zero, fmt.Errorf("failed to map JIRA fields to struct: %w", err)
	}

	return result, nil
}
