package network

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	cldf_config_domain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/domain"
	cldf_config_network "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	cldf_domain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
)

// stringToNetworkType converts a string network type to NetworkType enum.
func stringToNetworkType(s string) cldf_config_network.NetworkType {
	switch s {
	case "mainnet":
		return cldf_config_network.NetworkTypeMainnet
	case "testnet":
		return cldf_config_network.NetworkTypeTestnet
	default:
		return ""
	}
}

// determineNetworkTypes determines the network types for a given environment and domain.
// It first tries to load from domain config, and if that fails or doesn't exist,
// it falls back to the legacy switch logic.
func determineNetworkTypes(env string, domain cldf_domain.Domain, lggr logger.Logger) ([]cldf_config_network.NetworkType, error) {
	var (
		typesAll     = []cldf_config_network.NetworkType{cldf_config_network.NetworkTypeTestnet, cldf_config_network.NetworkTypeMainnet}
		typesTestnet = []cldf_config_network.NetworkType{cldf_config_network.NetworkTypeTestnet}
		typesMainnet = []cldf_config_network.NetworkType{cldf_config_network.NetworkTypeMainnet}
	)

	// Try to load domain config first
	domainConfigPath := filepath.Join(domain.DirPath(), ".config", "domain.yaml")
	if _, err := os.Stat(domainConfigPath); err == nil {
		domainConfig, err := cldf_config_domain.Load(domainConfigPath)
		if err != nil {
			lggr.Warnf("Failed to load domain config, falling back to legacy logic: %v", err)
		} else {
			envConfig, ok := domainConfig.Environments[env]
			if !ok {
				return nil, fmt.Errorf("environment %s not found in domain config", env)
			}

			var networkTypes []cldf_config_network.NetworkType
			for _, accessType := range envConfig.NetworkAccess {
				if nt := stringToNetworkType(accessType); nt != "" {
					networkTypes = append(networkTypes, nt)
				}
			}
			return networkTypes, nil
		}
	}

	// Fallback to legacy switch logic
	var networkTypes []cldf_config_network.NetworkType
	switch env {
	case environment.Local, environment.StagingTestnet, environment.ProdTestnet:
		networkTypes = typesTestnet
	case environment.StagingMainnet, environment.ProdMainnet:
		networkTypes = typesMainnet
	case environment.Prod:
		networkTypes = typesAll
	// The following environments are legacy environments that are used to support domains which
	// have not transitioned to the new environment structure.
	case environment.Testnet, environment.SolStaging:
		networkTypes = typesTestnet
	case environment.Staging:
		if domain.Key() == "data-streams" {
			networkTypes = typesAll
		} else {
			networkTypes = typesTestnet
		}
	case environment.Mainnet:
		networkTypes = typesMainnet
	default:
		lggr.Errorf("Unknown environment: %s", env)
		return nil, fmt.Errorf("unknown env: %s", env)
	}

	return networkTypes, nil
}

// LoadNetworks retrieves the network configuration for the given domain and filters the networks
// according to the specified environment. This ensures that only networks relevant to the selected
// environment are accessible, minimizing the risk of accidental operations on unintended networks.
func LoadNetworks(
	env string, domain cldf_domain.Domain, lggr logger.Logger,
) (*cldf_config_network.Config, error) {
	cfg, err := loadNetworkConfig(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to load network config: %w", err)
	}

	networkTypes, err := determineNetworkTypes(env, domain, lggr)
	if err != nil {
		return nil, fmt.Errorf("failed to determine network types: %w", err)
	}

	lggr.Infof("Loaded %s Networks for %s/%s", networkTypes, domain.Key(), env)

	return cfg.FilterWith(cldf_config_network.TypesFilter(networkTypes...)), nil
}

// loadNetworkConfig loads the network config from the .config directory in the given domain.
func loadNetworkConfig(domain cldf_domain.Domain) (*cldf_config_network.Config, error) {
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

	cfg, err := cldf_config_network.Load(configFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to load config files: %w", err)
	}

	return cfg, nil
}
