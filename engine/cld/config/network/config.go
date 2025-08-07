package network

import (
	"fmt"
	"maps"
	"os"
	"slices"

	"gopkg.in/yaml.v3"
)

// Manifest is the YAML representation of network configuration.
type Manifest struct {
	// A YAML array of networks.
	Networks []Network `yaml:"networks"`
}

// Config represents the configuration of a collection of networks. This is loaded from the YAML
// manifest file/s.
type Config struct {
	// networks is a map of networks by their chain selector. This differs from the manifest
	// representation of the networks so that we can ensure uniqueness and quickly lookup a network
	// by its chain selector.
	networks map[uint64]Network
}

// NewConfig creates a new config from a slice of networks. Any duplicate chain selectors will
// be overwritten.
func NewConfig(networks []Network) *Config {
	nmap := make(map[uint64]Network)

	for _, network := range networks {
		nmap[network.ChainSelector] = network
	}

	return &Config{
		networks: nmap,
	}
}

// Networks returns a slice of all networks in the config.
func (c *Config) Networks() []Network {
	return slices.Collect(maps.Values(c.networks))
}

// NetworkBySelector retrieves a network by its chain selector. If the network is not found, an
// error is returned.
func (c *Config) NetworkBySelector(selector uint64) (Network, error) {
	network, ok := c.networks[selector]
	if !ok {
		return Network{}, fmt.Errorf("network with selector %d not found in configuration", selector)
	}

	return network, nil
}

// ChainSelectors returns a slice of all chain selectors from the Config.
func (c *Config) ChainSelectors() []uint64 {
	return slices.Collect(maps.Keys(c.networks))
}

// Merge merges another config into the current config.
// It overwrites any networks with the same chain selector.
func (c *Config) Merge(other *Config) {
	maps.Copy(c.networks, other.networks)
}

// MarshalYAML implements the yaml.Marshaler interface for the Config struct.
// It converts the internal map structure to a YAML format with a top-level "networks" key.
func (c *Config) MarshalYAML() (any, error) {
	node := Manifest{
		Networks: c.Networks(),
	}

	return node, nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for the Config struct.
func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	node := Manifest{}

	if err := value.Decode(&node); err != nil {
		return err
	}

	*c = *NewConfig(node.Networks)

	return nil
}

// NetworkFilter defines a function type that filters networks based on certain criteria.
type NetworkFilter func(Network) bool

// FilterWith returns a new Config containing only Networks that pass all provided filter functions.
// Filters are applied in sequence (AND logic) - a network must pass all filters to be included.
func (c *Config) FilterWith(filters ...NetworkFilter) *Config {
	// Start with all networks from the current config
	networks := c.Networks()

	// Apply each filter sequentially, removing networks that don't pass
	for _, filter := range filters {
		networks = slices.DeleteFunc(networks, func(network Network) bool {
			return !filter(network) // Delete networks that don't pass the filter
		})
	}

	return NewConfig(networks)
}

// TypesFilter returns a filter function that matches chains with the specified network types.
func TypesFilter(networkTypes ...NetworkType) NetworkFilter {
	return func(network Network) bool {
		return slices.Contains(networkTypes, network.Type)
	}
}

// ChainSelectorFilter returns a filter function that matches chains with the specified chain
// selector
func ChainSelectorFilter(selector uint64) NetworkFilter {
	return func(network Network) bool {
		return network.ChainSelector == selector
	}
}

// ChainFamilyFilter returns a filter function that matches chains with the specified chain family.
func ChainFamilyFilter(chainFamily string) NetworkFilter {
	return func(network Network) bool {
		family, err := network.ChainFamily()
		if err != nil {
			return false
		}

		return family == chainFamily
	}
}

// Load loads configuration from the specified file paths, and merges them into a single Config.
//
// It accepts load options to customize the loading behavior.
func Load(filePaths []string, opts ...LoadOption) (*Config, error) {
	cfg := NewConfig([]Network{})

	// Apply load options to populate the loading configuration.
	loadCfg := &loadConfig{}
	for _, opt := range opts {
		opt(loadCfg)
	}

	// Load each file path into the config.
	for _, fp := range filePaths {
		data, err := os.ReadFile(fp)
		if err != nil {
			return nil, fmt.Errorf("failed to read networks file: %w", err)
		}

		var fileCfg Config
		if err := yaml.Unmarshal(data, &fileCfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal networks YAML: %w", err)
		}

		cfg.Merge(&fileCfg)
	}

	// Apply the URL transformer if one is provided.
	if loadCfg.URLTransformer != nil {
		loadCfg.URLTransformer(cfg)
	}

	return cfg, nil
}

// LoadOption defines a function that modifies the load configuration.
type LoadOption func(*loadConfig)

// URLTransformer is a function that transforms the URLs in the Config.
type URLTransformer func(*Config)

// loadConfig holds the configuration for loading the config.
type loadConfig struct {
	URLTransformer URLTransformer
}

// WithURLTransform allows setting a custom URL transformer for the config.
func WithURLTransform(t URLTransformer) func(opts *loadConfig) {
	return func(opts *loadConfig) {
		opts.URLTransformer = t
	}
}
