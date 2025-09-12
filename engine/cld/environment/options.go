package environment

import (
	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

// LoadConfig contains configuration parameters for loading an environment.
//
// This struct holds all the configurable options that affect how an environment is loaded,
// including which components to initialize, which chains to load, and various behavioral flags.
type LoadConfig struct {
	// reporter specifies the operations reporter to use for the environment.
	// Defaults to operations.MemoryReporter() if not specified
	reporter operations.Reporter

	// operationRegistry provides the registry of available operations for the environment.
	// Defaults to operations.NewOperationRegistry() if not specified.
	operationRegistry *operations.OperationRegistry

	// migrationString identifies a specific migration when using OnlyLoadChainsFor.
	// Used in conjunction with chainSelectorsToLoad to limit environment loading
	// to specific chains for a particular migration.
	migrationString string

	// chainSelectorsToLoad specifies which chain selectors to load when using
	// OnlyLoadChainsFor. If empty, all chains are loaded by default.
	chainSelectorsToLoad []uint64

	// withoutJD determines whether to skip Job Distributor initialization.
	// When true, the Environment's Offchain field will be nil.
	// Useful for migrations that don't require Job Distributor functionality.
	withoutJD bool

	// anvilKeyAsDeployer determines whether to use Anvil's default private key
	// as the deployer key in forked environments. Only applicable for local testing.
	anvilKeyAsDeployer bool

	// lggr is the logger instance used throughout environment loading and operations.
	// Defaults to a new logger instance if not specified.
	lggr logger.Logger

	// useDryRunJobDistributor configures the environment to use a dry-run Job Distributor
	// that allows read operations but performs noop write operations.
	useDryRunJobDistributor bool
}

// Configure applies a slice of LoadEnvironmentOption functions to the LoadConfig.
//
// This method is used internally by the Load function to apply user-provided
// configuration options to the default LoadConfig instance.
func (c *LoadConfig) Configure(opts []LoadEnvironmentOption) {
	for _, opt := range opts {
		opt(c)
	}
}

// newLoadConfig creates a new LoadConfig instance with sensible default values.
func newLoadConfig() (*LoadConfig, error) {
	lggr, err := logger.New()
	if err != nil {
		return nil, err
	}

	// Default options
	return &LoadConfig{
		reporter:          operations.NewMemoryReporter(),
		operationRegistry: operations.NewOperationRegistry(),
		lggr:              lggr,
	}, nil
}

// LoadEnvironmentOption is a functional option type for configuring environment loading.
type LoadEnvironmentOption func(*LoadConfig)

// WithAnvilKeyAsDeployer configures the environment to use Anvil's default private key as the EVM deployer key.
//
// This option is intended for local development and testing with forked EVM environments.
// When enabled, the environment will use Anvil's well-known private key instead of generating
// or loading a different deployer key.
//
// Warning: This should NEVER be used in production environments as the Anvil key is publicly known.
func WithAnvilKeyAsDeployer() LoadEnvironmentOption {
	return func(o *LoadConfig) {
		o.anvilKeyAsDeployer = true
	}
}

// WithReporter configures a custom operations reporter for environment loading.
//
// The reporter is responsible for tracking and recording operations performed during
// environment loading and subsequent operations. By default, a memory-based reporter
// is used, but this option allows you to provide a custom implementation.
func WithReporter(reporter operations.Reporter) LoadEnvironmentOption {
	return func(o *LoadConfig) {
		o.reporter = reporter
	}
}

// WithoutJD configures the environment to skip Job Distributor initialization.
//
// By default, the environment loading process initializes the Job Distributor component.
// This option disables that initialization, which can be useful for:
//   - Changeset executions that don't require offchain components
//   - Faster environment loading when JD is not needed
//   - Testing scenarios where JD dependencies are not available
//
// WARNING: When this option is used, env.Offchain will be nil. Any code that
// attempts to use env.Offchain will panic. Ensure your migration or operation
// does not depend on Job Distributor functionality.
func WithoutJD() LoadEnvironmentOption {
	return func(o *LoadConfig) {
		o.withoutJD = true
	}
}

// OnlyLoadChainsFor configures the environment to load only specified chains for a migration.
//
// This option optimizes environment loading by restricting it to only the chains
// required for a specific migration. This can significantly reduce loading time
// and resource usage when working with environments that support many chains.
//
// By default, all available chains in the environment are loaded. This option
// allows you to specify exactly which chains are needed.
func OnlyLoadChainsFor(migrationKey string, chainsSelectors []uint64) LoadEnvironmentOption {
	return func(o *LoadConfig) {
		o.migrationString = migrationKey
		o.chainSelectorsToLoad = chainsSelectors
	}
}

// WithOperationRegistry configures the environment to use a custom operation registry.
//
// The operation registry contains all available operations that can be executed
// within the environment. By default, a new empty registry is created. This option allows you to
// provide a pre-configured registry with custom operations or modified behavior.
func WithOperationRegistry(registry *operations.OperationRegistry) LoadEnvironmentOption {
	return func(o *LoadConfig) {
		o.operationRegistry = registry
	}
}

// WithLogger configures the environment to use a custom logger instance.
//
// The logger is used throughout the environment loading process and subsequent
// operations for debugging, informational messages, and error reporting.
// By default, a new logger instance is created automatically.
//
// This option is useful when you need to:
//   - Use a specific logger configuration (log level, format, output)
//   - Integrate with existing logging infrastructure
//   - Use a test logger for unit tests
//   - Share a logger instance across multiple components
func WithLogger(lggr logger.Logger) LoadEnvironmentOption {
	return func(o *LoadConfig) {
		o.lggr = lggr
	}
}

// WithDryRunJobDistributor configures the environment to use a dry-run Job Distributor.
//
// The dry-run Job Distributor is a special mode that allows safe testing of operations
// that would normally modify the Job Distributor state. In this mode:
//   - Read operations are forwarded to the real Job Distributor backend
//   - Write operations are stubbed out and logged but not executed
//   - This allows testing of migration logic without affecting production systems
//
// This option is particularly useful for:
//   - Running fork tests without affecting the production environment
//   - Testing migrations against production environments safely
//   - Validating operation logic before actual deployment
//   - Debugging issues without side effects
func WithDryRunJobDistributor() LoadEnvironmentOption {
	return func(o *LoadConfig) {
		o.useDryRunJobDistributor = true
	}
}
