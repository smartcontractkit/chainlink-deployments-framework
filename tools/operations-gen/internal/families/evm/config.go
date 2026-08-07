package evm

// EvmContractConfig is the EVM-specific contract configuration decoded from YAML.
type EvmContractConfig struct {
	Name              string            `yaml:"contract_name"`
	Version           string            `yaml:"version"`
	VersionPath       string            `yaml:"version_path,omitempty"` // Optional: override folder path derived from version
	PackageName       string            `yaml:"package_name,omitempty"` // Optional: override package name
	OmitDeploy        bool              `yaml:"omit_deploy,omitempty"`  // Optional: skip Deploy operation
	GobindingsPackage string            `yaml:"gobindings_package"`     // Optional: override the derived gobindings import path or relative filesystem path for this contract.
	ZkSyncBytecode    ZkSyncBytecodeRef `yaml:"zksync_bytecode,omitempty"`
	// DeployContractTypes lists ContractType labels (e.g. "ProposerManyChainMultiSig") that share
	// the same ABI and bytecode as this contract but need separate datastore entries.
	// Labels must be valid Go exported identifiers. When non-nil, ONLY these labels appear as
	// BytecodeByTypeAndVersion keys and each gets <Label>ContractType + <Label>TypeAndVersion vars.
	// An empty list is rejected. Cannot be set when omit_deploy is true.
	DeployContractTypes []string            `yaml:"deploy_contract_types,omitempty"`
	Functions           []EvmFunctionConfig `yaml:"functions"`
	ConfigDir           string              `yaml:"-"`
}

type EvmInputConfig struct {
	// GobindingsPackage is the required parent Go import path, or relative filesystem path,
	// containing versioned abigen packages, unless every contract provides its own override.
	// Contract packages default to:
	//   <gobindings_package>/<version_path>/<package_name>
	GobindingsPackage string `yaml:"gobindings_package"`
	// ZkSyncBindingsPackage is the default Go import path for zkSync VM deploy bytecode.
	// Used when a contract sets zksync_bytecode to a symbol only.
	ZkSyncBindingsPackage string `yaml:"zksync_bindings_package,omitempty"`
}

// EvmFunctionConfig selects a contract function and assigns its access control.
type EvmFunctionConfig struct {
	Name   string `yaml:"name"`
	Access string `yaml:"access,omitempty"` // "owner", "role", "authorized", "private", "workflows_owner" or "public"
	// Role is the OpenZeppelin-style role name used when Access is "role".
	// Accepted formats:
	//   - DEFAULT_ADMIN_ROLE                 → all-zero bytes32
	//   - SOME_ROLE                          → keccak256("SOME_ROLE"), matching Solidity's bytes32 constant
	Role string `yaml:"role,omitempty"`
}

type EvmOutputConfig struct {
	BasePath string `yaml:"base_path"`
}
