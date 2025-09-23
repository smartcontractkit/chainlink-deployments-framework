package config

import (
	cfgjira "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/jira"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// LoadJiraConfig retrieves the JIRA configuration for a given domain.
func LoadJiraConfig(dom fdomain.Domain) (*cfgjira.JiraConfig, error) {
	domainConfigPath := dom.ConfigDomainFilePath()

	return cfgjira.Load(domainConfigPath)
}
