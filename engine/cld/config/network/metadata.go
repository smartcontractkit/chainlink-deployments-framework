package network

import (
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/helper"
	"gopkg.in/yaml.v3"
)

// EVMMetadata is a struct that holds metadata specific to EVM networks.
type EVMMetadata struct {
	AnvilConfig *AnvilConfig `yaml:"anvil_config,omitempty"`
}

// StellarMetadata holds metadata specific to Stellar networks.
// The main RPC URL comes from network.RPCs (like other chains); only passphrase and Friendbot (faucet) URL live here.
type StellarMetadata struct {
	NetworkPassphrase string `yaml:"network_passphrase"`
	FriendbotURL      string `yaml:"friendbot_url"`
}

// AnvilConfig holds the configuration for starting an Anvil node.
type AnvilConfig struct {
	Image          string `yaml:"image"`
	Port           uint64 `yaml:"port"`
	ArchiveHTTPURL string `yaml:"archive_http_url"`
}

// Validate checks if the AnvilConfig has all required fields set.
func (a AnvilConfig) Validate() error {
	if a.Image == "" {
		return errors.New("image is not defined")
	}
	if a.Port == 0 {
		return errors.New("port is not defined")
	}

	return nil
}

// DecodeMetadata converts the metadata field from an any interface to a user-specified type using yaml marshaling.
// Use your own custom types or one of the predefined common types.
// Example usage:
//
//	type CustomMetadata struct {
//		CustomField  string `yaml:"custom_field"`
//		AnotherField int    `yaml:"another_field"`
//	}
//
//	customMetadata, err := DecodeMetadata[CustomMetadata](metadata)
//	if err != nil {
//	  // handle error
//	}
func DecodeMetadata[T any](metadata any) (T, error) {
	var target T
	if metadata == nil {
		return target, errors.New("metadata is nil")
	}

	// Marshal the metadata back to YAML bytes
	yamlBytes, err := yaml.Marshal(metadata)
	if err != nil {
		return target, fmt.Errorf("failed to marshal metadata to YAML: %w", err)
	}

	// Unmarshal into the target type
	if err := yaml.Unmarshal(yamlBytes, &target); err != nil {
		return target, fmt.Errorf("failed to unmarshal metadata to target type: %w", err)
	}

	// Coerce big int strings as YAML parsing may interpret large numbers as strings
	matchFunc := helper.DefaultMatchKeysToFix
	target = helper.CoerceBigIntStringsForKeys(target, matchFunc).(T)

	return target, nil
}
