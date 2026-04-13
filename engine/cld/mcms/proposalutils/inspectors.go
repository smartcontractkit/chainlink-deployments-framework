package proposalutils

import (
	"fmt"

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

// McmsInspectors builds an mcmssdk.Inspector for every chain in the environment,
// returning them keyed by uint64 chain selector. All inspectors use the default
// TimelockActionSchedule action.
func McmsInspectors(env cldf.Environment) (map[uint64]mcmssdk.Inspector, error) {
	chainsMetadata := map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{}
	for chainSelector := range env.BlockChains.All() {
		chainsMetadata[mcmstypes.ChainSelector(chainSelector)] = mcmstypes.ChainMetadata{}
	}

	chainAccessor := cldfmcmsadapters.Wrap(env.BlockChains)

	mcmsInspectors, err := mcmschainwrappers.BuildInspectors(&chainAccessor, chainsMetadata, mcmstypes.TimelockActionSchedule)
	if err != nil {
		return nil, fmt.Errorf("failed to build inspectors: %w", err)
	}

	inspectors := make(map[uint64]mcmssdk.Inspector, len(mcmsInspectors))
	for chainSelector, inspector := range mcmsInspectors {
		inspectors[uint64(chainSelector)] = inspector
	}

	return inspectors, nil
}
