package environment

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	config_domain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/domain"
	config_env "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	config_network "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// Config consolidates all the config that is required to be loaded for a domain environment.
//
// Specifically it contains the network config and secrets which is loaded from files or env vars.
type Config struct {
	Networks *config_network.Config // The network config loaded from the network manifest file
	Env      *config_env.Config     // The cld engine's environment config
}

// LoadConfig loads and consolidates all configuration required for a domain environment, including
// network configuration and environment-specific settings.n.
func LoadConfig(dom domain.Domain, env string, lggr logger.Logger) (*Config, error) {
	// Load the network manifest for the domain environment.
	networks, err := LoadNetworks(env, dom, lggr)
	if err != nil {
		return nil, fmt.Errorf("failed to load networks: %w", err)
	}

	envCfg, err := LoadEnvConfig(dom, env)
	if err != nil {
		return nil, fmt.Errorf("failed to load env config: %w", err)
	}

	return &Config{
		Networks: networks,
		Env:      envCfg,
	}, nil
}

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

// LoadNetworks retrieves the network configuration for the given domain and filters the networks
// according to the specified environment. This ensures that only networks relevant to the selected
// environment are accessible, minimizing the risk of accidental operations on unintended networks.
func LoadNetworks(
	env string, dom domain.Domain, lggr logger.Logger,
) (*config_network.Config, error) {
	cfg, err := loadNetworkConfig(dom)
	if err != nil {
		return nil, fmt.Errorf("failed to load network config: %w", err)
	}

	// Load network types from domain config
	domainConfigPath := filepath.Join(dom.ConfigDomainFilePath())
	if _, statErr := os.Stat(domainConfigPath); statErr != nil {
		return nil, fmt.Errorf("domain config not found at %s: %w", domainConfigPath, statErr)
	}

	networkTypes, err := loadDomainConfigNetworkTypes(env, dom)
	if err != nil {
		return nil, fmt.Errorf("failed to load domain config network types: %w", err)
	}

	lggr.Infof("Loaded %s Networks for %s/%s", networkTypes, dom.Key(), env)

	return cfg.FilterWith(config_network.TypesFilter(networkTypes...)), nil
}

// loadNetworkConfig loads the network config from the .config directory in the given domain.
func loadNetworkConfig(domain domain.Domain) (*config_network.Config, error) {
	// Check if the .config directory exists in the domain
	configDir := filepath.Join(domain.DirPath(), ".config")
	if _, err := os.Stat(configDir); err != nil {
		return nil, fmt.Errorf("cannot find config directory: %w", err)
	}

	// Find all yaml config files in the .config directory and any subdirectories
	var configFiles []string

	yamlFiles, err := filepath.Glob(filepath.Join(configDir, "**", "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find config files: %w", err)
	}
	configFiles = append(configFiles, yamlFiles...)

	ymlFiles, err := filepath.Glob(filepath.Join(configDir, "**", "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find config files: %w", err)
	}
	configFiles = append(configFiles, ymlFiles...)

	if len(configFiles) == 0 {
		return nil, fmt.Errorf("no config files found in %s", configDir)
	}

	cfg, err := config_network.Load(configFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to load config files: %w", err)
	}

	return cfg, nil
}

// loadDomainConfigNetworkTypes loads network types from domain config for the given environment.
func loadDomainConfigNetworkTypes(env string, dom domain.Domain) ([]config_network.NetworkType, error) {
	domainConfigPath := filepath.Join(dom.ConfigDomainFilePath())
	domainConfig, err := config_domain.Load(domainConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load domain config: %w", err)
	}

	envConfig, ok := domainConfig.Environments[env]
	if !ok {
		return nil, fmt.Errorf("environment %s not found in domain config", env)
	}

	networkTypes := make([]config_network.NetworkType, 0, len(envConfig.NetworkTypes))
	for _, networkType := range envConfig.NetworkTypes {
		networkTypes = append(networkTypes, config_network.NetworkType(networkType))
	}

	return networkTypes, nil
}
