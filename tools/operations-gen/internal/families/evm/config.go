package evm

// Config is the top-level structure of operations_gen_config.yaml.
type Config struct {
	Version   string           `yaml:"version"`
	Output    OutputConfig     `yaml:"output"`
	Contracts []ContractConfig `yaml:"contracts"`
}

// OutputConfig controls where generated files are written.
type OutputConfig struct {
	// BasePath is the root directory for all generated output, resolved relative
	// to the config file's location.
	BasePath string `yaml:"base_path"`
}

// ContractConfig describes a single contract to generate operations for.
type ContractConfig struct {
	Name    string `yaml:"contract_name"`
	Version string `yaml:"version"`
	// OutputVersionPath is the path segment under the output base for this
	// contract's operations (e.g. v1_0_0). When empty, derived from version.
	OutputVersionPath string `yaml:"output_version_path,omitempty"`
	// PackageName overrides the default snake_case derivation of the Go package name.
	PackageName string `yaml:"package_name,omitempty"`
	// NoDeployment skips generating the Deploy operation and omits the bytecode constant.
	NoDeployment bool `yaml:"no_deployment,omitempty"`
	// GobindingsPackage is the Go import path of the gobindings package for this contract.
	// Required — the generator reads ABI and bytecode directly from the gobindings source.
	GobindingsPackage string           `yaml:"gobindings_package"`
	Functions         []FunctionConfig `yaml:"functions"`
}

// FunctionConfig selects a contract function to expose as a CLD operation.
type FunctionConfig struct {
	Name string `yaml:"name"`
	// Access controls the caller-authorization check: "public" (anyone) or "role".
	Access string `yaml:"access,omitempty"`
	// Role is the OpenZeppelin-style role name used when Access is "role".
	// Accepted formats:
	//   - DEFAULT_ADMIN_ROLE  → all-zero bytes32
	//   - SOME_ROLE           → keccak256("SOME_ROLE"), matching Solidity's bytes32 constant
	//   - 64 hex chars        → raw bytes32 value
	Role string `yaml:"role,omitempty"`
}
