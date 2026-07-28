package proposalutils

import (
	mcmschainwrappers "github.com/smartcontractkit/mcms/chainwrappers"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"

	mcmstypes "github.com/smartcontractkit/mcms/types"

	cldfmcmsadapters "github.com/smartcontractkit/chainlink-deployments-framework/chain/mcms/adapters"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

type mcmsInspectorOptions struct {
	TimelockAction mcmstypes.TimelockAction
}

// MCMSInspectorOption configures how MCMS inspectors are built.
type MCMSInspectorOption func(*mcmsInspectorOptions)

// WithTimelockAction sets the timelock action used by the inspector.
// When omitted, the default action is TimelockActionSchedule.
func WithTimelockAction(action mcmstypes.TimelockAction) MCMSInspectorOption {
	return func(opts *mcmsInspectorOptions) {
		opts.TimelockAction = action
	}
}

// McmsInspectorForChain builds an mcmssdk.Inspector for a single chain in the given environment.
// The chain must be present in env.BlockChains, otherwise an error is returned.
func McmsInspectorForChain(env cldf.Environment, chain uint64, opts ...MCMSInspectorOption) (mcmssdk.Inspector, error) {
	var options mcmsInspectorOptions
	for _, opt := range opts {
		opt(&options)
	}

	action := mcmstypes.TimelockActionSchedule
	if options.TimelockAction != "" {
		action = options.TimelockAction
	}

	chainAccessor := cldfmcmsadapters.Wrap(env.BlockChains)

	return mcmschainwrappers.BuildInspector(&chainAccessor, mcmstypes.ChainSelector(chain), action,
		mcmstypes.ChainMetadata{})
}

// McmsInspectors builds an mcmssdk.Inspector for each chain in the environment
// that can be constructed with empty chain metadata, keyed by uint64 chain
// selector. All inspectors use the default TimelockActionSchedule action.
//
// Chains that require chain-specific metadata the framework does not hold, such
// as Sui which needs AdditionalFields populated from onchain MCMS state, are
// skipped rather than failing the whole set. Proposal building only consumes
// inspectors for chains that have batch operations, so a skipped chain that is
// later needed still surfaces as a missing-inspector error.
func McmsInspectors(env cldf.Environment) (map[uint64]mcmssdk.Inspector, error) {
	chainAccessor := cldfmcmsadapters.Wrap(env.BlockChains)

	inspectors := make(map[uint64]mcmssdk.Inspector)
	for chainSelector := range env.BlockChains.All() {
		inspector, err := mcmschainwrappers.BuildInspector(
			&chainAccessor,
			mcmstypes.ChainSelector(chainSelector),
			mcmstypes.TimelockActionSchedule,
			mcmstypes.ChainMetadata{},
		)
		if err != nil {
			continue
		}
		inspectors[uint64(chainSelector)] = inspector
	}

	return inspectors, nil
}
