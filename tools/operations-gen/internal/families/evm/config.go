package evm

// EvmContractConfig is the EVM-specific contract configuration decoded from YAML.
type EvmContractConfig struct {
	Name        string              `yaml:"contract_name"`
	Version     string              `yaml:"version"`
	VersionPath string              `yaml:"version_path,omitempty"` // Optional: override folder path derived from version
	PackageName string              `yaml:"package_name,omitempty"` // Optional: override package name
	ABIFile     string              `yaml:"abi_file,omitempty"`     // Optional: override ABI file name
	OmitDeploy  bool                `yaml:"omit_deploy,omitempty"`  // Optional: skip Deploy operation
	Functions   []evmFunctionConfig `yaml:"functions"`
}

// evmFunctionConfig selects a contract function and assigns its access control.
type evmFunctionConfig struct {
	Name   string `yaml:"name"`
	Access string `yaml:"access,omitempty"` // "owner" or "public"
}

type EvmInputConfig struct {
	ABIBasePath      string `yaml:"abi_base_path"`
	BytecodeBasePath string `yaml:"bytecode_base_path"`
}

type EvmOutputConfig struct {
	BasePath string `yaml:"base_path"`
}
