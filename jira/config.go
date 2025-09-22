package jira

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// JiraConfig represents the JIRA configuration for a domain
type JiraConfig struct {
	Domain     string                  `yaml:"-"` // Domain path (populated at runtime, not from YAML)
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

// GetJiraFields extracts all JIRA field names from the field mappings for more efficient API calls
func (c *JiraConfig) GetJiraFields() []string {
	fields := make([]string, 0, len(c.FieldMaps))
	for _, fieldMapping := range c.FieldMaps {
		fields = append(fields, fieldMapping.JiraField)
	}

	return fields
}

// DomainConfig represents the full domain configuration file
type DomainConfig struct {
	Environments map[string]any `yaml:"environments"`
	Jira         *JiraConfig    `yaml:"jira"`
}

// loadDomainJiraConfig loads JIRA configuration for the specified domain
func loadDomainJiraConfig(domainName string) (*JiraConfig, error) {
	// Find the domains root by walking up from current directory
	domainsRoot, err := findDomainsRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find domains root: %w", err)
	}

	configPath := filepath.Join(domainsRoot, domainName, ".config", "domain.yaml")

	// Check if file exists
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		return nil, fmt.Errorf("domain config not found at %s", configPath)
	}

	data, readErr := os.ReadFile(configPath)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read domain config: %w", readErr)
	}

	var domainConfig DomainConfig
	if unmarshalErr := yaml.Unmarshal(data, &domainConfig); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to parse domain config: %w", unmarshalErr)
	}

	if domainConfig.Jira == nil {
		return nil, errors.New("no JIRA configuration found in domain config")
	}

	// Populate the domain field with the domain name
	domainConfig.Jira.Domain = domainName

	return domainConfig.Jira, nil
}

// findDomainsRoot walks up from the current working directory to find the domains directory
func findDomainsRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	current := cwd
	for {
		domainsPath := filepath.Join(current, "domains")
		if info, err := os.Stat(domainsPath); err == nil && info.IsDir() {
			return domainsPath, nil
		}

		parent := filepath.Dir(current)
		if parent == current { // reached root
			break
		}
		current = parent
	}

	return "", fmt.Errorf("could not find domains directory from current working directory: %s", cwd)
}
