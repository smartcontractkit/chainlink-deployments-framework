package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	cfgdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/domain"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// LoadNetworks retrieves the network configuration for the given domain and filters the networks
// according to the specified environment. This ensures that only networks relevant to the selected
// environment are accessible, minimizing the risk of accidental operations on unintended networks.
func LoadNetworks(
	env string, dom fdomain.Domain, lggr logger.Logger,
) (*cfgnet.Config, error) {
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

	return cfg.FilterWith(cfgnet.TypesFilter(networkTypes...)), nil
}

// loadNetworkConfig loads the network config from the .config/networks directory in the given fdomain.
func loadNetworkConfig(domain fdomain.Domain) (*cfgnet.Config, error) {
	// Check if the .config/networks directory exists in the domain
	configNetworkDir := domain.ConfigNetworksDirPath()
	if _, err := os.Stat(configNetworkDir); err != nil {
		return nil, fmt.Errorf("cannot find config directory: %w", err)
	}

	// Find all yaml config files in the .config/networks directory
	var configFiles []string

	yamlFiles, err := filepath.Glob(filepath.Join(configNetworkDir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find config files: %w", err)
	}
	configFiles = append(configFiles, yamlFiles...)

	ymlFiles, err := filepath.Glob(filepath.Join(configNetworkDir, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find config files: %w", err)
	}
	configFiles = append(configFiles, ymlFiles...)

	if len(configFiles) == 0 {
		return nil, fmt.Errorf("no config files found in %s", configNetworkDir)
	}

	cfg, err := cfgnet.Load(configFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to load config files: %w", err)
	}

	return cfg, nil
}

// loadDomainConfigNetworkTypes loads network types from domain config for the given environment.
func loadDomainConfigNetworkTypes(env string, dom fdomain.Domain) ([]cfgnet.NetworkType, error) {
	domainConfigPath := filepath.Join(dom.ConfigDomainFilePath())
	domainConfig, err := cfgdomain.Load(domainConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load domain config: %w", err)
	}

	envConfig, ok := domainConfig.Environments[env]
	if !ok {
		return nil, fmt.Errorf("environment %s not found in domain config", env)
	}

	networkTypes := make([]cfgnet.NetworkType, 0, len(envConfig.NetworkTypes))
	for _, networkType := range envConfig.NetworkTypes {
		networkTypes = append(networkTypes, cfgnet.NetworkType(networkType))
	}

	return networkTypes, nil
}
