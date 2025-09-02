package config

import (
	"fmt"
	"path/filepath"

	config_env "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// LoadEnvConfig retrieves the environment configuration for a given domain and environment.
//
// Loading strategy:
//   - In CI environments: Loads configuration exclusively from environment variables set by the CI pipeline.
//   - In local development: Loads configuration from a local config file specific to the domain and environment.
func LoadEnvConfig(dom domain.Domain, env string) (*config_env.Config, error) {
	if isCI() {
		cfg, err := config_env.LoadEnv()
		if err != nil {
			return nil, fmt.Errorf("failed to load env config: %w", err)
		}

		return cfg, nil
	}

	fp := filepath.Join(dom.ConfigLocalFilePath(env))

	return config_env.LoadFile(fp)
}
