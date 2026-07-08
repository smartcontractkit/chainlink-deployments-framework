package domain

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"

	"github.com/smartcontractkit/chainlink-deployments-framework/cre"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/jira"
)

// DatastoreType represents the type of datastore to use for persisting deployment data.
type DatastoreType string

const (
	// DatastoreTypeFile indicates data should be persisted to local JSON files (default behavior).
	DatastoreTypeFile DatastoreType = "file"
	// DatastoreTypeCatalog indicates data should be persisted to the remote catalog service.
	DatastoreTypeCatalog DatastoreType = "catalog"
	// DatastoreTypeAll indicates data should be persisted to both local JSON files and the remote catalog service.
	// This is useful to keep backward compatibility during the transition period from file-based to remote catalog.
	DatastoreTypeAll DatastoreType = "all"
)

// String returns the string representation of the DatastoreType.
func (d DatastoreType) String() string {
	return string(d)
}

// IsValid checks if the DatastoreType is a valid value.
func (d DatastoreType) IsValid() bool {
	return d == DatastoreTypeFile || d == DatastoreTypeCatalog || d == DatastoreTypeAll
}

// CREConfig represents the CRE (Chainlink Runtime Environment) configuration for a domain.
type CREConfig struct {
	Enabled           bool                       `mapstructure:"enabled" yaml:"enabled"`
	DefaultRegistries []cre.ContextRegistryEntry `mapstructure:"default_registries" yaml:"default_registries,omitempty"`
}

// BinaryProvider identifies how a domain binary should be resolved.
type BinaryProvider string

const (
	// BinaryProviderSource builds the domain binary from source.
	BinaryProviderSource BinaryProvider = "source"
	// BinaryProviderS3 downloads the domain binary from the CLD-managed S3 bucket.
	BinaryProviderS3 BinaryProvider = "s3"

	// DefaultBinaryVersion is used when binary.version is not specified.
	DefaultBinaryVersion = "latest"
)

// IsValid reports whether the binary provider is supported.
func (p BinaryProvider) IsValid() bool {
	return p == "" || p == BinaryProviderSource || p == BinaryProviderS3
}

// BinaryConfig represents the optional domain binary configuration.
// When provider is "s3", bucket, prefix, auth, and object paths are managed
// internally by CLD and are not configurable per domain. The binary name is
// derived from the domain key. When omitted from domain.yaml, defaults are
// applied during Load.
type BinaryConfig struct {
	Provider BinaryProvider `mapstructure:"provider" yaml:"provider,omitempty"`
	Version  string         `mapstructure:"version" yaml:"version,omitempty"`
}

// DefaultBinaryConfig returns the default binary configuration used when the
// binary section is omitted from domain.yaml.
func DefaultBinaryConfig() *BinaryConfig {
	return &BinaryConfig{
		Provider: BinaryProviderSource,
		Version:  DefaultBinaryVersion,
	}
}

func (cfg *BinaryConfig) applyDefaults() {
	if cfg.Provider == "" {
		cfg.Provider = BinaryProviderSource
	}

	if cfg.Version == "" {
		cfg.Version = DefaultBinaryVersion
	}
}

func (cfg *BinaryConfig) validate() error {
	if cfg == nil || cfg.Provider.IsValid() {
		return nil
	}

	return fmt.Errorf("invalid binary provider: %s (must be 'source' or 's3')", cfg.Provider)
}

// Environment represents a single environment configuration.
type Environment struct {
	NetworkTypes []string      `mapstructure:"network_types" yaml:"network_types"`
	Datastore    DatastoreType `mapstructure:"datastore" yaml:"datastore"`
	CRE          *CREConfig    `mapstructure:"cre" yaml:"cre,omitempty"`
}

// creDefaultRegistries returns the CRE default registries when CRE is enabled,
// or nil otherwise.
func (e *Environment) creDefaultRegistries() []cre.ContextRegistryEntry {
	if e.CRE == nil || !e.CRE.Enabled {
		return nil
	}

	return e.CRE.DefaultRegistries
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
		return fmt.Errorf("invalid datastore value: %s (must be 'file', 'catalog', or 'all')", e.Datastore)
	}

	for i, r := range e.creDefaultRegistries() {
		if err := r.Validate(); err != nil {
			return fmt.Errorf("cre.default_registries[%d]: %w", i, err)
		}
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
	Binary       *BinaryConfig          `mapstructure:"binary" yaml:"binary,omitempty"`
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

	if err := cfg.Binary.validate(); err != nil {
		return fmt.Errorf("invalid binary configuration: %w", err)
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
			// todo: remove this default when catalog is fully enabled for domains
			env.Datastore = DatastoreTypeFile // Default to file if not specified
			cfg.Environments[name] = env
		}
	}

	if cfg.Binary == nil {
		cfg.Binary = DefaultBinaryConfig()
	} else {
		cfg.Binary.applyDefaults()
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
