package config

import (
	"errors"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/domain"
	cfgjira "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/jira"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// ErrJiraConfigNotFound is returned when no JIRA configuration is found in the domain config.
var ErrJiraConfigNotFound = errors.New("no Jira configuration found in domain config")

// LoadJiraConfig retrieves the JIRA configuration for a given domain.
func LoadJiraConfig(dom fdomain.Domain) (*cfgjira.Config, error) {
	domainConfigPath := dom.ConfigDomainFilePath()

	// Load the full domain config (this handles validation including JIRA)
	domainConfig, err := domain.Load(domainConfigPath)
	if err != nil {
		return nil, err
	}

	// Extract the JIRA config (validation already done by domain.Load)
	if domainConfig.Jira == nil {
		return nil, ErrJiraConfigNotFound
	}

	return domainConfig.Jira, nil
}
