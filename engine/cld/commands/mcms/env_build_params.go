package mcms

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldfenvironment "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment/envparams"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

const environmentFlagUsage = "Deployment environment (required unless --" + envparams.FlagName + " or " +
	envparams.EnvVar + " supplies it)"

// withEnvBuildParams adds the build params flag to cmd and returns a getter for
// the resolved params (nil when none were supplied). Call it after flags.Environment.
//
// It uses PreRunE, which cobra runs before required-flag validation, and which
// cannot shadow a persistent hook defined by the embedding binary.
func withEnvBuildParams(cmd *cobra.Command, deps *Deps) func() *cldfenvironment.BuildParams {
	var params *cldfenvironment.BuildParams

	envparams.RegisterFlag(cmd.Flags())
	if f := cmd.Flags().Lookup(envparams.EnvFlagName); f != nil {
		f.Usage = environmentFlagUsage
	}

	cmd.PreRunE = func(cmd *cobra.Command, _ []string) error {
		resolved, err := envparams.Resolve(cmd.Flags(), deps.Getenv)
		if err != nil {
			return err
		}
		params = resolved

		return nil
	}

	return func() *cldfenvironment.BuildParams { return params }
}

// buildEnvironment builds the environment from params. All listed networks are
// dialed, so chainSelectors is only used to check each needed chain is present:
// first that it is listed, before building, then that it loaded.
func buildEnvironment(
	ctx context.Context,
	lggr logger.Logger,
	dom domain.Domain,
	deps *Deps,
	params cldfenvironment.BuildParams,
	chainSelectors []uint64,
) (cldf.Environment, error) {
	if params.Domain != dom.Key() {
		return cldf.Environment{}, fmt.Errorf(
			"build params are for domain %q, but this binary is for domain %q", params.Domain, dom.Key(),
		)
	}

	listed := make(map[uint64]bool, len(params.Networks))
	for _, network := range params.Networks {
		listed[network.ChainSelector] = true
	}
	for _, selector := range chainSelectors {
		if !listed[selector] {
			return cldf.Environment{}, fmt.Errorf(
				"chain selector %d is not listed under networks in the build params", selector,
			)
		}
	}

	env, err := deps.EnvironmentBuilder(ctx, params, cldfenvironment.BuildWithLogger(lggr))
	if err != nil {
		return cldf.Environment{}, fmt.Errorf("failed to build environment from build params: %w", err)
	}

	for _, selector := range chainSelectors {
		if _, err = env.BlockChains.GetBySelector(selector); err != nil {
			return cldf.Environment{}, fmt.Errorf(
				"chain selector %d is listed in the build params but was not loaded, "+
					"check that a signing key is configured for its chain family: %w", selector, err,
			)
		}
	}

	if env.DataStore == nil {
		return cldf.Environment{}, errors.New("built environment has no datastore")
	}
	refs, err := env.DataStore.Addresses().Fetch()
	if err != nil {
		return cldf.Environment{}, fmt.Errorf("failed to read addresses from the datastore: %w", err)
	}
	if len(refs) == 0 {
		lggr.Warnw("The catalog returned no addresses for this domain and environment, "+
			"so contracts will not be resolved by address",
			"domain", params.Domain, "environment", params.Environment)
	}

	return env, nil
}
