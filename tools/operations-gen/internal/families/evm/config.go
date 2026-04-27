package evm

// EvmContractConfig is the EVM-specific contract configuration decoded from YAML.
type EvmContractConfig struct {
	Name              string              `yaml:"contract_name"`
	Version           string              `yaml:"version"`
	VersionPath       string              `yaml:"version_path,omitempty"` // Optional: override folder path derived from version
	PackageName       string              `yaml:"package_name,omitempty"` // Optional: override package name
	OmitDeploy        bool                `yaml:"omit_deploy,omitempty"`  // Optional: skip Deploy operation
	GobindingsPackage string              `yaml:"gobindings_package"`     // Optional: override the derived Go import path of the gobindings package for this contract.
	Functions         []EvmFunctionConfig `yaml:"functions"`
	ConfigDir         string              `yaml:"-"`
}

type EvmInputConfig struct {
	// GobindingsPackage is the parent Go import path containing versioned abigen packages.
	// Contract packages default to:
	//   <gobindings_package>/<version_path>/<package_name>
	GobindingsPackage string `yaml:"gobindings_package"`
}

// EvmFunctionConfig selects a contract function and assigns its access control.
type EvmFunctionConfig struct {
	Name   string `yaml:"name"`
	Access string `yaml:"access,omitempty"` // "owner", "role", or "public"
	// Role is the OpenZeppelin-style role name used when Access is "role".
	// Accepted formats:
	//   - DEFAULT_ADMIN_ROLE                 → all-zero bytes32
	//   - SOME_ROLE                          → keccak256("SOME_ROLE"), matching Solidity's bytes32 constant
	Role string `yaml:"role,omitempty"`
}

type EvmOutputConfig struct {
	BasePath string `yaml:"base_path"`
}
