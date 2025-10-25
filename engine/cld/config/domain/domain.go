package domain

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/jira"
)

// DatastoreType represents the type of datastore to use for persisting deployment data.
type DatastoreType string

const (
	// DatastoreTypeFile indicates data should be persisted to local JSON files (default behavior).
	DatastoreTypeFile DatastoreType = "file"
	// DatastoreTypeCatalog indicates data should be persisted to the remote catalog service.
	DatastoreTypeCatalog DatastoreType = "catalog"
)

// String returns the string representation of the DatastoreType.
func (d DatastoreType) String() string {
	return string(d)
}

// IsValid checks if the DatastoreType is a valid value.
func (d DatastoreType) IsValid() bool {
	return d == DatastoreTypeFile || d == DatastoreTypeCatalog
}

// Environment represents a single environment configuration.
type Environment struct {
	NetworkTypes []string      `mapstructure:"network_types" yaml:"network_types"`
	Datastore    DatastoreType `mapstructure:"datastore" yaml:"datastore"`
}

// validate validates the environment configuration.
func (e *Environment) validate() error {
	if len(e.NetworkTypes) == 0 {
		return errors.New("network_types is required and cannot be empty")
	}

	// Check for valid values
	for _, networkType := range e.NetworkTypes {
		if !isValidNetworkType(networkType) {
			return errors.New("invalid network_types value: " + networkType + " (must be 'mainnet' or 'testnet')")
		}
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, networkType := range e.NetworkTypes {
		if seen[networkType] {
			return errors.New("duplicate network_types value: " + networkType)
		}
		seen[networkType] = true
	}

	// Validate datastore field if provided
	if e.Datastore != "" && !e.Datastore.IsValid() {
		return fmt.Errorf("invalid datastore value: %s (must be 'file' or 'catalog')", e.Datastore)
	}

	return nil
}

// isValidNetworkType checks if the network type value is valid.
func isValidNetworkType(networkType string) bool {
	return networkType == "mainnet" || networkType == "testnet"
}

// DomainConfig represents the parsed and validated domain configuration.
type DomainConfig struct {
	Environments map[string]Environment `mapstructure:"environments" yaml:"environments"`
	Jira         *jira.Config           `mapstructure:"jira" yaml:"jira"`
}

// validate validates all environments in the domain configuration.
func (cfg *DomainConfig) validate() error {
	if len(cfg.Environments) == 0 {
		return errors.New("environments is required and cannot be empty")
	}

	// Validate each environment in the domain configuration.
	for name, env := range cfg.Environments {
		if err := env.validate(); err != nil {
			return fmt.Errorf("invalid config for environment %s: %w", name, err)
		}
	}

	// Validate JIRA config if present (it's optional)
	if cfg.Jira != nil {
		if err := cfg.Jira.Validate(); err != nil {
			return fmt.Errorf("invalid JIRA configuration: %w", err)
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

	// Apply defaults to environments
	for name, env := range cfg.Environments {
		if env.Datastore == "" {
			env.Datastore = DatastoreTypeFile // Default to file if not specified
			cfg.Environments[name] = env
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
