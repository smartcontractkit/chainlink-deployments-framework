package runtime

import "github.com/smartcontractkit/chainlink-deployments-framework/engine/test/environment"

// runtimeConfig is the configuration for initializing the runtime.
type runtimeConfig struct {
	envOpts []environment.LoadOpt
}

// RuntimeOption is a functional option type for configuring runtime.
type RuntimeOption func(*runtimeConfig)

// WithEnvironmentOptions adds environment options to the runtime. This is used to load the
// environment with the given options.
func WithEnvOpts(opts ...environment.LoadOpt) RuntimeOption {
	return func(c *runtimeConfig) {
		c.envOpts = opts
	}
}
