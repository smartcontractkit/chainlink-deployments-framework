package mcms

import (
	"context"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/types"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldfenvironment "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// EnvironmentLoaderFunc loads a deployment environment.
type EnvironmentLoaderFunc func(
	ctx context.Context,
	dom domain.Domain,
	envKey string,
	lggr logger.Logger,
	opts ...cldfenvironment.LoadEnvironmentOption,
) (cldf.Environment, error)

// ProposalLoaderFunc loads a proposal from a file.
type ProposalLoaderFunc func(kind types.ProposalKind, path string) (mcms.ProposalInterface, error)

// Deps holds optional dependencies that can be overridden for testing.
type Deps struct {
	// EnvironmentLoader loads a deployment environment.
	EnvironmentLoader EnvironmentLoaderFunc

	// ProposalLoader loads a proposal from a file.
	ProposalLoader ProposalLoaderFunc
}

// applyDefaults fills in nil fields with production implementations.
func (d *Deps) applyDefaults() {
	if d.EnvironmentLoader == nil {
		d.EnvironmentLoader = defaultEnvironmentLoader
	}
	if d.ProposalLoader == nil {
		d.ProposalLoader = mcms.LoadProposal
	}
}

// defaultEnvironmentLoader wraps cldfenvironment.Load with our signature.
func defaultEnvironmentLoader(
	ctx context.Context,
	dom domain.Domain,
	envKey string,
	lggr logger.Logger,
	opts ...cldfenvironment.LoadEnvironmentOption,
) (cldf.Environment, error) {
	// Always add the logger option
	allOpts := append([]cldfenvironment.LoadEnvironmentOption{cldfenvironment.WithLogger(lggr)}, opts...)

	return cldfenvironment.Load(ctx, dom, envKey, allOpts...)
}
