package proposalanalysis

import (
	experimentalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

// EngineOption configures the analyzer engine using the functional options pattern
type EngineOption func(*engineConfig)

// engineConfig holds configuration for the analyzer engine
type engineConfig struct {
	evmRegistry    experimentalanalyzer.EVMABIRegistry
	solanaRegistry experimentalanalyzer.SolanaDecoderRegistry
}

// WithEVMRegistry allows injecting an EVM ABI registry into the analyzer engine
// The registry will be made available to all analyzers through the AnalyzerContext
//
// Example:
//
//	evmRegistry, _ := experimentalanalyzer.NewEnvironmentEVMRegistry(env, map[string]string{
//	    "MyContract": "/path/to/abi.json",
//	})
//	engine := internal.NewAnalyzerEngine(analyzer.WithEVMRegistry(evmRegistry))
func WithEVMRegistry(registry experimentalanalyzer.EVMABIRegistry) EngineOption {
	return func(cfg *engineConfig) {
		cfg.evmRegistry = registry
	}
}

// WithSolanaRegistry allows injecting a Solana decoder registry into the analyzer engine
// The registry will be made available to all analyzers through the AnalyzerContext
//
// Example:
//
//	solanaRegistry, _ := experimentalanalyzer.NewEnvironmentSolanaRegistry(env, map[string]DecodeInstructionFn{
//	    "MyProgram": myDecoder,
//	})
//	engine := internal.NewAnalyzerEngine(analyzer.WithSolanaRegistry(solanaRegistry))
func WithSolanaRegistry(registry experimentalanalyzer.SolanaDecoderRegistry) EngineOption {
	return func(cfg *engineConfig) {
		cfg.solanaRegistry = registry
	}
}

// ApplyEngineOptions applies all engine options and returns the configuration
// This is used internally by the engine implementation
func ApplyEngineOptions(opts ...EngineOption) *engineConfig {
	cfg := &engineConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// GetEVMRegistry returns the EVM registry from the config
func (cfg *engineConfig) GetEVMRegistry() experimentalanalyzer.EVMABIRegistry {
	return cfg.evmRegistry
}

// GetSolanaRegistry returns the Solana registry from the config
func (cfg *engineConfig) GetSolanaRegistry() experimentalanalyzer.SolanaDecoderRegistry {
	return cfg.solanaRegistry
}
