package jira

import (
	"errors"

	"github.com/spf13/viper"
)

// JiraConfig represents the JIRA configuration for a domain
type JiraConfig struct {
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
func (c *JiraConfig) GetJiraFields() []string {
	fields := make([]string, 0, len(c.FieldMaps))
	for _, fieldMapping := range c.FieldMaps {
		fields = append(fields, fieldMapping.JiraField)
	}

	return fields
}

// validate validates the JIRA configuration.
func (c *JiraConfig) validate() error {
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

// DomainConfig represents the domain configuration file with JIRA section
type DomainConfig struct {
	Jira *JiraConfig `mapstructure:"jira" yaml:"jira"`
}

// Load loads JIRA configuration from a domain YAML file.
func Load(filePath string) (*JiraConfig, error) {
	v := viper.New()
	v.SetConfigFile(filePath)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := &DomainConfig{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	if cfg.Jira == nil {
		return nil, errors.New("no JIRA configuration found in domain config")
	}

	if err := cfg.Jira.validate(); err != nil {
		return nil, err
	}

	return cfg.Jira, nil
}
