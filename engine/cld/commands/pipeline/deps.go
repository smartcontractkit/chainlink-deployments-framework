package pipeline

import (
	"context"

	"github.com/smartcontractkit/mcms"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/timelockdelay"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// EnvironmentLoaderFunc loads a deployment environment.
type EnvironmentLoaderFunc func(
	ctx context.Context,
	dom domain.Domain,
	envKey string,
	opts ...environment.LoadEnvironmentOption,
) (fdeployment.Environment, error)

// TimelockDelayCorrectorFunc corrects schedule proposal delays against on-chain minDelay.
type TimelockDelayCorrectorFunc func(
	ctx context.Context,
	lggr logger.Logger,
	blockChains chain.BlockChains,
	proposals []mcms.TimelockProposal,
) error

// Deps holds optional dependencies that can be overridden for testing.
type Deps struct {
	// EnvironmentLoader loads a deployment environment. Default: environment.Load
	EnvironmentLoader EnvironmentLoaderFunc
	// TimelockDelayCorrector corrects timelock proposal delays. Default: timelockdelay.CorrectTimelockDelays
	TimelockDelayCorrector TimelockDelayCorrectorFunc
}

// DefaultEnvironmentLoader is used when Deps.EnvironmentLoader is nil.
// Tests can override this to inject a mock.
var DefaultEnvironmentLoader = environment.Load

// applyDefaults fills in nil fields with production implementations.
func (d *Deps) applyDefaults() {
	if d.EnvironmentLoader == nil {
		d.EnvironmentLoader = DefaultEnvironmentLoader
	}
	if d.TimelockDelayCorrector == nil {
		d.TimelockDelayCorrector = timelockdelay.CorrectTimelockDelays
	}
}
