package jira

import (
	"fmt"
)

// JiraToStruct is the main function that domains use to convert JIRA issues to config structs
// It automatically loads the domain's JIRA schema and maps the issue fields
func JiraToStruct[T any](issueKey string) (T, error) {
	var zero T

	if issueKey == "" {
		return zero, fmt.Errorf("issue_key is required")
	}

	// 1. Load domain's JIRA configuration
	config, err := loadDomainJiraConfig()
	if err != nil {
		return zero, fmt.Errorf("failed to load domain JIRA config: %w", err)
	}

	// 2. Create JIRA client
	client, err := NewClient(config.Connection.BaseURL, config.Connection.Username)
	if err != nil {
		return zero, fmt.Errorf("failed to create JIRA client: %w", err)
	}

	// 3. Fetch issue from JIRA
	issue, err := client.GetIssue(issueKey)
	if err != nil {
		return zero, fmt.Errorf("failed to fetch JIRA issue %s: %w", issueKey, err)
	}

	// 4. Map JIRA fields to target struct
	result, err := mapFieldsToStruct[T](issue, config)
	if err != nil {
		return zero, fmt.Errorf("failed to map JIRA fields to struct: %w", err)
	}

	return result, nil
}
