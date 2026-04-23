package evm

import (
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/core"
)

const (
	// anyType is the fallback Go type for unknown source types.
	anyType = "any"
	// emptyReturnType is the Go type used for read functions with no return values.
	emptyReturnType = "struct{}"

	abiTypeFunction         = "function"
	abiTypeConstructor      = "constructor"
	stateMutabilityView     = "view"
	stateMutabilityPure     = "pure"
	accessOwner             = "owner"
	accessPublic            = "public"
	accessControlAllCallers = "AllCallersAllowed"
	accessControlOnlyOwner  = "OnlyOwner"
)

// ---- Handler ----

// Handler implements ChainFamilyHandler for EVM (Solidity/go-ethereum) chains.
type Handler struct{}

// Generate decodes each YAML node as an EvmContractConfig, extracts contract info,
// and writes a generated operations file for each contract.
func (h Handler) Generate(config core.Config, tmpl *template.Template) error {
	var input EvmInputConfig
	if err := config.Input.Decode(&input); err != nil {
		return fmt.Errorf("failed to decode EVM input config: %w", err)
	}
	var output EvmOutputConfig
	if err := config.Output.Decode(&output); err != nil {
		return fmt.Errorf("failed to decode EVM output config: %w", err)
	}
	if config.ConfigDir != "" {
		input.ABIBasePath = filepath.Join(config.ConfigDir, input.ABIBasePath)
		input.BytecodeBasePath = filepath.Join(config.ConfigDir, input.BytecodeBasePath)
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

		info, err := extractContractInfo(cfg, input, output)
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
