package proposalanalysis

import (
	"time"
)

// DefaultAnalyzerTimeout is the default timeout budget for a single analyzer
// invocation, including both CanAnalyze and Analyze.
const DefaultAnalyzerTimeout = 2 * time.Minute

// EngineOption configures the analyzer engine using the functional options pattern
type EngineOption func(*engineConfig)

// engineConfig holds configuration for the analyzer engine
type engineConfig struct {
	analyzerTimeout time.Duration
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

// WithAnalyzerTimeout allows configuring the timeout for analyzer execution
// Each analyzer will be given this amount of time to complete before being cancelled
// This is important for analyzers that make network calls or other long-running operations
// Default is 2 minutes if not specified
//
// Example:
//
//	engine := proposalanalysis.NewAnalyzerEngine(
//	    proposalanalysis.WithAnalyzerTimeout(2 * time.Minute),
//	)
func WithAnalyzerTimeout(timeout time.Duration) EngineOption {
	return func(cfg *engineConfig) {
		cfg.analyzerTimeout = timeout
	}
}

// GetAnalyzerTimeout returns the analyzer timeout from the config
// Returns DefaultAnalyzerTimeout (2 minutes) if none was provided
func (cfg *engineConfig) GetAnalyzerTimeout() time.Duration {
	if cfg.analyzerTimeout == 0 {
		return DefaultAnalyzerTimeout
	}

	return cfg.analyzerTimeout
}
