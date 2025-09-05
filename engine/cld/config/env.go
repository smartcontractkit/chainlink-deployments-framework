package config

import (
	"fmt"
	"path/filepath"

	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// LoadEnvConfig retrieves the environment configuration for a given domain and environment.
//
// Loading strategy:
//   - In CI environments: Loads configuration exclusively from environment variables set by the CI pipeline.
//   - In local development: Loads configuration from a local config file specific to the domain and environment.
func LoadEnvConfig(dom fdomain.Domain, env string) (*cfgenv.Config, error) {
	if isCI() {
		cfg, err := cfgenv.LoadEnv()
		if err != nil {
			return nil, fmt.Errorf("failed to load env config: %w", err)
		}

		return cfg, nil
	}

	fp := filepath.Join(dom.ConfigLocalFilePath(env))

	return cfgenv.LoadFile(fp)
}
