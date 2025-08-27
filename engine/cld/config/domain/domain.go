package domain

import (
	"errors"

	"github.com/spf13/viper"
)

// Environment represents a single environment configuration.
type Environment struct {
	NetworkAccess []string `mapstructure:"networkAccess" yaml:"networkAccess"`
}

// validate validates the environment configuration.
func (e *Environment) validate() error {
	if len(e.NetworkAccess) == 0 {
		return errors.New("networkAccess is required and cannot be empty")
	}

	// Check for valid values
	for _, access := range e.NetworkAccess {
		if !isValidNetworkAccess(access) {
			return errors.New("invalid networkAccess value: " + access + " (must be 'mainnet' or 'testnet')")
		}
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, access := range e.NetworkAccess {
		if seen[access] {
			return errors.New("duplicate networkAccess value: " + access)
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

// validate validates all environments in the domain configuration.
func (cfg *DomainConfig) validate() error {
	// Validate each environment in the domain configuration.
	for name, env := range cfg.Environments {
		if err := env.validate(); err != nil {
			return errors.Join(errors.New("invalid config for environment "+name), err)
		}
	}

	return nil
}

// Load loads domain configuration from a YAML file.
func Load(filePath string) (*DomainConfig, error) {
	v := viper.New()
	v.SetConfigFile(filePath)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := &DomainConfig{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
