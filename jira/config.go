package jira

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// loadDomainJiraConfig loads JIRA configuration for the detected domain
func loadDomainJiraConfig() (*JiraConfig, error) {
	domain, err := detectCurrentDomain()
	if err != nil {
		return nil, fmt.Errorf("failed to detect domain: %w", err)
	}

	configPath := filepath.Join(domain, ".config", "domain.yaml")

	// Check if file exists
	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("domain config not found at %s", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read domain config: %w", err)
	}

	var domainConfig DomainConfig
	if err := yaml.Unmarshal(data, &domainConfig); err != nil {
		return nil, fmt.Errorf("failed to parse domain config: %w", err)
	}

	if domainConfig.Jira == nil {
		return nil, errors.New("no JIRA configuration found in domain config")
	}

	// Populate the domain field with just the domain name (last part of the path)
	domainConfig.Jira.Domain = filepath.Base(domain)

	return domainConfig.Jira, nil
}

// detectCurrentDomain attempts to detect which domain we're operating in
// by analyzing the current working directory
func detectCurrentDomain() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Walk up the directory tree looking for domains directory
	current := cwd
	for {
		domainsPath := filepath.Join(current, "domains")
		if info, err := os.Stat(domainsPath); err == nil && info.IsDir() {
			// Found domains directory, now find which domain we're in
			return findDomainInPath(cwd, domainsPath)
		}

		parent := filepath.Dir(current)
		if parent == current { // reached root
			break
		}
		current = parent
	}

	return "", fmt.Errorf("could not detect domain from current working directory: %s", cwd)
}

// findDomainInPath determines which domain directory the current path is within
func findDomainInPath(cwd, domainsPath string) (string, error) {
	relPath, err := filepath.Rel(domainsPath, cwd)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) == 0 || parts[0] == "" {
		return "", errors.New("not inside a domain directory")
	}

	domainName := parts[0]
	domainPath := filepath.Join(domainsPath, domainName)

	// Verify the domain directory exists
	if _, err := os.Stat(domainPath); err != nil {
		return "", fmt.Errorf("domain directory does not exist: %s", domainPath)
	}

	return domainPath, nil
}
