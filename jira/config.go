package jira

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// JiraConfig represents the JIRA configuration for a domain
type JiraConfig struct {
	Connection JiraConnectionConfig    `yaml:"connection"`
	FieldMaps  map[string]FieldMapping `yaml:"field_maps"`
}

// JiraConnectionConfig contains JIRA connection details
type JiraConnectionConfig struct {
	BaseURL  string `yaml:"base_url"`
	Project  string `yaml:"project"`
	Username string `yaml:"username,omitempty"`
}

// FieldMapping defines how a JIRA field maps to a config field
type FieldMapping struct {
	JiraField string `yaml:"jira_field"` // e.g., "customfield_10001"
}

// loadDomainJiraConfig loads JIRA configuration for the detected domain
func loadDomainJiraConfig() (*JiraConfig, error) {
	domain, err := detectCurrentDomain()
	if err != nil {
		return nil, fmt.Errorf("failed to detect domain: %w", err)
	}

	configPath := fmt.Sprintf("%s/.config/jira-schema.yaml", domain)

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("JIRA config not found for domain at %s", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JIRA config: %w", err)
	}

	var config JiraConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JIRA config: %w", err)
	}

	return &config, nil
}

// detectCurrentDomain attempts to detect which domain we're operating in
// by analyzing the current working directory
func detectCurrentDomain() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Look for domains directory in current path or parent paths
	current := cwd
	for i := 0; i < 10; i++ { // limit search depth
		if strings.Contains(current, "domains") {
			// Extract domain name from path like /path/to/domains/exemplar/...
			parts := strings.Split(current, string(filepath.Separator))
			for j, part := range parts {
				if part == "domains" && j+1 < len(parts) {
					domainName := parts[j+1]
					// Return the full domain path
					return filepath.Join(filepath.Dir(current)[:strings.LastIndex(filepath.Dir(current), "domains")+7], domainName), nil
				}
			}
		}

		parent := filepath.Dir(current)
		if parent == current { // reached root
			break
		}
		current = parent
	}

	return "", fmt.Errorf("could not detect domain from current working directory: %s", cwd)
}
