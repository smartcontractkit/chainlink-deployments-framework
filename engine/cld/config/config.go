package config

import (
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	cfgjira "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/jira"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// Config aggregates all configuration required by the Chainlink Deployments (CLD) engine.
// It combines network-specific settings and environment-specific configuration
// to provide a complete runtime configuration for deployment operations.
type Config struct {
	// Networks contains blockchain network configurations loaded from YAML manifest files.
	// This includes chain selectors, RPC endpoints, and network-specific parameters
	// for all supported blockchain networks.
	Networks *cfgnet.Config

	// Env contains environment-specific configuration including credentials, API keys,
	// and deployment settings. This configuration varies between environments
	// (development, staging, production) and contains sensitive data.
	Env *cfgenv.Config

	// Jira contains JIRA integration configuration including connection details
	// and field mappings for using JIRA in resolvers.
	Jira *cfgjira.Config
}

// Load loads and consolidates all configuration required for a domain environment, including
// network configuration and environment-specific settings.n.
func Load(dom fdomain.Domain, env string, lggr logger.Logger) (*Config, error) {
	networks, err := LoadNetworks(env, dom, lggr)
	if err != nil {
		return nil, fmt.Errorf("failed to load networks: %w", err)
	}

	envCfg, err := LoadEnvConfig(dom, env)
	if err != nil {
		return nil, fmt.Errorf("failed to load env config: %w", err)
	}

	jiraCfg, err := LoadJiraConfig(dom)
	if err != nil {
		if errors.Is(err, ErrJiraConfigNotFound) {
			lggr.Infof("JIRA config not available: %v", err)
			jiraCfg = nil
		} else {
			return nil, fmt.Errorf("failed to load JIRA config: %w", err)
		}
	}

	return &Config{
		Networks: networks,
		Env:      envCfg,
		Jira:     jiraCfg,
	}, nil
}
