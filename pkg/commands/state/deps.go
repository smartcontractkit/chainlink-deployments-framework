// Package state provides CLI commands for state management operations.
package state

import (
	"context"
	"encoding/json"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
)

// EnvironmentLoaderFunc loads a deployment environment for the given domain and environment key.
type EnvironmentLoaderFunc func(
	ctx context.Context,
	dom domain.Domain,
	envKey string,
	opts ...environment.LoadEnvironmentOption,
) (fdeployment.Environment, error)

// StateLoaderFunc loads the previous state from the environment directory.
// Returns the state as a JSONSerializer, or an error if loading fails.
// If the state file does not exist, implementations should return empty JSON.
type StateLoaderFunc func(envdir domain.EnvDir) (domain.JSONSerializer, error)

// StateSaverFunc saves the generated state to a file.
// If outputPath is empty, it should use the default path in the environment directory.
type StateSaverFunc func(envdir domain.EnvDir, outputPath string, state json.Marshaler) error

// ViewStateFunc is an alias for deployment.ViewStateV2 for clarity.
// It generates the current state view from the environment.
// It takes the environment and optionally the previous state for incremental updates.
type ViewStateFunc = fdeployment.ViewStateV2

// defaultEnvironmentLoader is the production implementation that loads an environment.
func defaultEnvironmentLoader(
	ctx context.Context,
	dom domain.Domain,
	envKey string,
	opts ...environment.LoadEnvironmentOption,
) (fdeployment.Environment, error) {
	return environment.Load(ctx, dom, envKey, opts...)
}

// defaultStateLoader loads state from the environment directory.
func defaultStateLoader(envdir domain.EnvDir) (domain.JSONSerializer, error) {
	return envdir.LoadState()
}

// defaultStateSaver saves state to the environment directory or custom path.
func defaultStateSaver(envdir domain.EnvDir, outputPath string, state json.Marshaler) error {
	if outputPath != "" {
		return domain.SaveViewState(outputPath, state)
	}

	return envdir.SaveViewState(state)
}

// Deps holds the injectable dependencies for state commands.
// All fields are optional; nil values will use production defaults.
// Users can override these to provide custom implementations for their domain.
type Deps struct {
	// EnvironmentLoader loads a deployment environment.
	// Default: environment.Load
	EnvironmentLoader EnvironmentLoaderFunc

	// StateLoader loads the previous state from the environment directory.
	// Default: envdir.LoadState
	StateLoader StateLoaderFunc

	// StateSaver saves the generated state.
	// Default: envdir.SaveViewState or domain.SaveViewState
	StateSaver StateSaverFunc
}

// applyDefaults fills in nil dependencies with production defaults.
func (d *Deps) applyDefaults() {
	if d.EnvironmentLoader == nil {
		d.EnvironmentLoader = defaultEnvironmentLoader
	}
	if d.StateLoader == nil {
		d.StateLoader = defaultStateLoader
	}
	if d.StateSaver == nil {
		d.StateSaver = defaultStateSaver
	}
}
