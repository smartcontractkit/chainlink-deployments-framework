package jira

import (
	"errors"
)

// Config represents the JIRA configuration for a domain
type Config struct {
	Connection JiraConnectionConfig    `mapstructure:"connection" yaml:"connection"`
	FieldMaps  map[string]FieldMapping `mapstructure:"field_maps" yaml:"field_maps"`
}

// JiraConnectionConfig contains JIRA connection details
type JiraConnectionConfig struct {
	BaseURL  string `mapstructure:"base_url" yaml:"base_url"`
	Project  string `mapstructure:"project" yaml:"project"`
	Username string `mapstructure:"username" yaml:"username,omitempty"`
}

// FieldMapping defines how a JIRA field maps to a config field
type FieldMapping struct {
	JiraField string `mapstructure:"jira_field" yaml:"jira_field"` // e.g., "customfield_10001"
}

// GetJiraFields extracts all JIRA field names from the field mappings for more efficient API calls
func (c *Config) GetJiraFields() []string {
	fields := make([]string, 0, len(c.FieldMaps))
	for _, fieldMapping := range c.FieldMaps {
		fields = append(fields, fieldMapping.JiraField)
	}

	return fields
}

// Validate validates the JIRA configuration.
func (c *Config) Validate() error {
	if c.Connection.BaseURL == "" {
		return errors.New("connection.base_url is required")
	}
	if c.Connection.Project == "" {
		return errors.New("connection.project is required")
	}
	if c.Connection.Username == "" {
		return errors.New("connection.username is required")
	}

	return nil
}
