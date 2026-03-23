package pipeline

import (
	"context"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
)

// EnvironmentLoaderFunc loads a deployment environment.
type EnvironmentLoaderFunc func(
	ctx context.Context,
	dom domain.Domain,
	envKey string,
	opts ...environment.LoadEnvironmentOption,
) (fdeployment.Environment, error)

// Deps holds optional dependencies that can be overridden for testing.
type Deps struct {
	// EnvironmentLoader loads a deployment environment. Default: environment.Load
	EnvironmentLoader EnvironmentLoaderFunc
}

// DefaultEnvironmentLoader is used when Deps.EnvironmentLoader is nil.
// Tests can override this to inject a mock.
var DefaultEnvironmentLoader = environment.Load

// applyDefaults fills in nil fields with production implementations.
func (d *Deps) applyDefaults() {
	if d.EnvironmentLoader == nil {
		d.EnvironmentLoader = DefaultEnvironmentLoader
	}
}
