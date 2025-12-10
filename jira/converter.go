package jira

import (
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// JiraToStruct is the main function that domains use to convert JIRA issues to config structs
// It loads the specified domain's JIRA schema and maps the issue fields
func JiraToStruct[T any](client *Client, dom fdomain.Domain, issueKey string) (T, error) {
	var zero T

	if client == nil {
		return zero, errors.New("JIRA client is required")
	}
	if issueKey == "" {
		return zero, errors.New("issue_key is required")
	}

	jiraConfig, err := config.LoadJiraConfig(dom)
	if err != nil {
		return zero, fmt.Errorf("failed to load domain JIRA config: %w", err)
	}

	// Extract all JIRA field names from the field maps
	fieldsToFetch := jiraConfig.GetJiraFields()

	// Fetch issue from JIRA
	issue, err := client.GetIssue(issueKey, fieldsToFetch)
	if err != nil {
		return zero, fmt.Errorf("failed to fetch JIRA issue %s: %w", issueKey, err)
	}

	// Map JIRA fields to target struct
	result, err := mapFieldsToStruct[T](issue, jiraConfig)
	if err != nil {
		return zero, fmt.Errorf("failed to map JIRA fields to struct: %w", err)
	}

	return result, nil
}
