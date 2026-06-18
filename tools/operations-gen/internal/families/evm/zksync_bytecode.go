package evm

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// ZkSyncBytecodeRef identifies zkSync VM deploy bytecode for a contract.
// YAML accepts a shorthand symbol string or {package, symbol}.
type ZkSyncBytecodeRef struct {
	Package string
	Symbol  string
}

func (z *ZkSyncBytecodeRef) UnmarshalYAML(value *yaml.Node) error {
	*z = ZkSyncBytecodeRef{}
	if value == nil {
		return nil
	}

	switch value.Kind {
	case yaml.ScalarNode:
		if isYAMLNullScalar(value) {
			return nil
		}
		z.Symbol = value.Value

		return nil
	case yaml.MappingNode:
		var raw struct {
			Package string `yaml:"package"`
			Symbol  string `yaml:"symbol"`
		}
		if err := value.Decode(&raw); err != nil {
			return err
		}
		z.Package = raw.Package
		z.Symbol = raw.Symbol
		if z.Symbol == "" {
			return errors.New("zksync_bytecode mapping requires symbol")
		}

		return nil
	case yaml.DocumentNode, yaml.SequenceNode, yaml.AliasNode:
		return fmt.Errorf("zksync_bytecode must be a string or mapping, got %v", value.ShortTag())
	}

	return fmt.Errorf("zksync_bytecode must be a string or mapping, got %v", value.ShortTag())
}

func isYAMLNullScalar(value *yaml.Node) bool {
	return value.Tag == "!!null" || value.Value == "" || value.Value == "~"
}

func (z ZkSyncBytecodeRef) IsZero() bool {
	return z.Symbol == ""
}

func resolveZkSyncBytecode(
	cfg EvmContractConfig,
	input EvmInputConfig,
	gobindingsPackage string,
) (packagePath string, symbol string, err error) {
	if cfg.ZkSyncBytecode.IsZero() {
		return "", "", nil
	}

	symbol = cfg.ZkSyncBytecode.Symbol
	pkgPath := cfg.ZkSyncBytecode.Package
	if pkgPath == "" {
		pkgPath = input.ZkSyncBindingsPackage
	}
	if pkgPath == "" {
		return gobindingsPackage, symbol, nil
	}

	resolved, err := resolveGobindingsImportPath(pkgPath, cfg.ConfigDir)
	if err != nil {
		return "", "", fmt.Errorf("resolve zksync_bytecode package for contract %q: %w", cfg.Name, err)
	}

	return resolved, symbol, nil
}
