package config

import (
	"fmt"
	"path/filepath"

	cfgdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/domain"
	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// LoadDatastoreType retrieves the datastore type configuration for a given domain and environment.
// It loads the domain configuration file and extracts the datastore setting for the specified environment.
func LoadDatastoreType(dom fdomain.Domain, env string) (cfgdomain.DatastoreType, error) {
	domainCfgPath := dom.ConfigDomainFilePath()
	domainCfg, err := cfgdomain.Load(domainCfgPath)
	if err != nil {
		return "", fmt.Errorf("failed to load domain config: %w", err)
	}

	envConfig, exists := domainCfg.Environments[env]
	if !exists {
		return "", fmt.Errorf("environment %s not found in domain config", env)
	}

	return envConfig.Datastore, nil
}

// LoadEnvConfig retrieves the environment configuration for a given domain and environment.
//
// Loading strategy:
//   - In CI environments: Loads configuration exclusively from environment variables set by the CI pipeline.
//   - In local development: Loads configuration from a local config file if it exists, otherwise falls back
//     to environment variables. Environment variables can override file values when both are present.
func LoadEnvConfig(dom fdomain.Domain, env string) (*cfgenv.Config, error) {
	if isCI() {
		cfg, err := cfgenv.LoadEnv()
		if err != nil {
			return nil, fmt.Errorf("failed to load env config: %w", err)
		}

		return cfg, nil
	}

	fp := filepath.Join(dom.ConfigLocalFilePath(env))

	return cfgenv.Load(fp)
}
