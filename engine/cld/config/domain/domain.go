package domain

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

// Environment represents a single environment configuration.
type Environment struct {
	NetworkAccess []string `mapstructure:"networkAccess" yaml:"networkAccess"`
}

// Validate validates the environment configuration.
func (e *Environment) Validate() error {
	if len(e.NetworkAccess) == 0 {
		return errors.New("networkAccess is required and cannot be empty")
	}

	// Check for valid values
	for _, access := range e.NetworkAccess {
		if !isValidNetworkAccess(access) {
			return fmt.Errorf("invalid networkAccess value: %s (must be 'mainnet' or 'testnet')", access)
		}
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, access := range e.NetworkAccess {
		if seen[access] {
			return fmt.Errorf("duplicate networkAccess value: %s", access)
		}
		seen[access] = true
	}

	return nil
}

// isValidNetworkAccess checks if the network access value is valid.
func isValidNetworkAccess(access string) bool {
	return access == "mainnet" || access == "testnet"
}

// DomainConfig represents the parsed and validated domain configuration.
type DomainConfig struct {
	Environments map[string]Environment `mapstructure:"environments" yaml:"environments"`
}

// Load loads domain configuration from a YAML file.
func Load(filePath string) (*DomainConfig, error) {
	v := viper.New()
	v.SetConfigFile(filePath)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := &DomainConfig{}
	err := v.Unmarshal(cfg)

	return cfg, err
}
