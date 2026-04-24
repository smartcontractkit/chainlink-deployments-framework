package evm

import (
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/core"
)

const (
	accessOwner  = "owner"
	accessPublic = "public"
)

// ---- Handler ----

// Handler implements ChainFamilyHandler for EVM (Solidity/go-ethereum) chains.
type Handler struct{}

// Generate decodes each YAML node as an EvmContractConfig, extracts contract info,
// and writes a generated operations file for each contract.
func (h Handler) Generate(config core.Config, tmpl *template.Template) error {
	var output EvmOutputConfig
	if err := config.Output.Decode(&output); err != nil {
		return fmt.Errorf("failed to decode EVM output config: %w", err)
	}
	if config.ConfigDir != "" {
		output.BasePath = filepath.Join(config.ConfigDir, output.BasePath)
	}

	for _, node := range config.Contracts.Content {
		if node == nil {
			continue
		}
		var cfg EvmContractConfig
		if err := node.Decode(&cfg); err != nil {
			return fmt.Errorf("failed to decode EVM contract config: %w", err)
		}
		cfg.ConfigDir = config.ConfigDir

		info, err := extractContractInfo(cfg, output)
		if err != nil {
			return fmt.Errorf("error extracting info for %s: %w", cfg.Name, err)
		}

		if err := generateOperationsFile(info, tmpl); err != nil {
			return fmt.Errorf("error generating file for %s: %w", cfg.Name, err)
		}

		fmt.Printf("✓ Generated operations for %s at %s\n", info.Name, info.OutputPath)
	}

	return nil
}
