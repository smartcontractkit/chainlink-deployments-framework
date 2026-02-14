package proposalanalysis

import (
	"time"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// Default timeout for analyzer execution
const DefaultAnalyzerTimeout = 5 * time.Minute

// EngineOption configures the analyzer engine using the functional options pattern
type EngineOption func(*engineConfig)

// engineConfig holds configuration for the analyzer engine
type engineConfig struct {
	logger          logger.Logger
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

// WithLogger allows injecting a logger into the analyzer engine
// The logger will be used for logging errors and debug information during analysis
// If not provided, the engine will use a no-op logger
//
// Example:
//
//	lggr, _ := logger.New()
//	engine := proposalanalysis.NewAnalyzerEngine(proposalanalysis.WithLogger(lggr))
func WithLogger(lggr logger.Logger) EngineOption {
	return func(cfg *engineConfig) {
		cfg.logger = lggr
	}
}

// GetLogger returns the logger from the config
// Returns a no-op logger if none was provided
func (cfg *engineConfig) GetLogger() logger.Logger {
	if cfg.logger == nil {
		return logger.Nop()
	}
	return cfg.logger
}

// WithAnalyzerTimeout allows configuring the timeout for analyzer execution
// Each analyzer will be given this amount of time to complete before being cancelled
// This is important for analyzers that make network calls or other long-running operations
// Default is 5 minutes if not specified
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
// Returns DefaultAnalyzerTimeout (5 minutes) if none was provided
func (cfg *engineConfig) GetAnalyzerTimeout() time.Duration {
	if cfg.analyzerTimeout == 0 {
		return DefaultAnalyzerTimeout
	}
	return cfg.analyzerTimeout
}
