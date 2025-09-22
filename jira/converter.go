package jira

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// JiraToStruct is the main function that domains use to convert JIRA issues to config structs
// It automatically loads the domain's JIRA schema and maps the issue fields
func JiraToStruct[T any](issueKey string) (T, error) {
	var zero T

	if issueKey == "" {
		return zero, errors.New("issue_key is required")
	}

	// 1. Load domain's JIRA configuration
	config, err := loadDomainJiraConfig()
	if err != nil {
		return zero, fmt.Errorf("failed to load domain JIRA config: %w", err)
	}

	// Extract all JIRA field names from the field maps
	var fieldsToFetch = config.GetJiraFields()

	var domain = strings.ToUpper(config.Domain)

	// 2. Get JIRA token from environment variable
	token := os.Getenv(domain + "_JIRA_TOKEN")
	if token == "" {
		return zero, errors.New("JIRA_TOKEN environment variable is required")
	}

	// 3. Create JIRA client
	client, err := NewClient(config.Connection.BaseURL, config.Connection.Username, token)
	if err != nil {
		return zero, fmt.Errorf("failed to create JIRA client: %w", err)
	}

	// 4. Fetch issue from JIRA
	issue, err := client.GetIssue(issueKey, fieldsToFetch)
	if err != nil {
		return zero, fmt.Errorf("failed to fetch JIRA issue %s: %w", issueKey, err)
	}

	// 5. Map JIRA fields to target struct
	result, err := mapFieldsToStruct[T](issue, config)
	if err != nil {
		return zero, fmt.Errorf("failed to map JIRA fields to struct: %w", err)
	}

	return result, nil
}
