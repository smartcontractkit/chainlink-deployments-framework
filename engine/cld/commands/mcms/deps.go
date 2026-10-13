package mcms

import (
	"context"
	"math/big"
	"os"

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

// EnvironmentBuilderFunc builds a deployment environment from BuildParams.
type EnvironmentBuilderFunc func(
	ctx context.Context,
	params cldfenvironment.BuildParams,
	opts ...cldfenvironment.BuildOption,
) (cldf.Environment, error)

// ForkEnvironmentLoaderFunc loads a fork environment.
type ForkEnvironmentLoaderFunc func(
	ctx context.Context,
	domain domain.Domain,
	envKey string,
	blockNumbers map[uint64]*big.Int,
	opts ...cldfenvironment.LoadEnvironmentOption,
) (cldfenvironment.ForkedEnvironment, error)

// ProposalLoaderFunc loads a proposal from a file.
type ProposalLoaderFunc func(kind types.ProposalKind, path string) (mcms.ProposalInterface, error)

// Deps holds optional dependencies that can be overridden for testing.
type Deps struct {
	// EnvironmentLoader loads a deployment environment.
	EnvironmentLoader EnvironmentLoaderFunc

	// EnvironmentBuilder is used instead of EnvironmentLoader when a BuildParams
	// document is supplied.
	EnvironmentBuilder EnvironmentBuilderFunc

	// Getenv reads environment variables.
	Getenv func(string) string

	// ForkEnvironmentLoader loads a deployment environment.
	ForkEnvironmentLoader ForkEnvironmentLoaderFunc

	// ProposalLoader loads a proposal from a file.
	ProposalLoader ProposalLoaderFunc
}

// applyDefaults fills in nil fields with production implementations.
func (d *Deps) applyDefaults() {
	if d.EnvironmentLoader == nil {
		d.EnvironmentLoader = defaultEnvironmentLoader
	}
	if d.EnvironmentBuilder == nil {
		d.EnvironmentBuilder = cldfenvironment.Build
	}
	if d.Getenv == nil {
		d.Getenv = os.Getenv
	}
	if d.ForkEnvironmentLoader == nil {
		d.ForkEnvironmentLoader = cldfenvironment.LoadFork
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
