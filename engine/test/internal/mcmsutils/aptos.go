package mcmsutils

import (
	"fmt"
	"maps"
	"slices"

	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	mcmsaptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	fchainaptos "github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
)

var (
	actionToAptosRole = map[mcmstypes.TimelockAction]mcmsaptossdk.TimelockRole{
		mcmstypes.TimelockActionSchedule: mcmsaptossdk.TimelockRoleProposer,
		mcmstypes.TimelockActionBypass:   mcmsaptossdk.TimelockRoleBypasser,
		mcmstypes.TimelockActionCancel:   mcmsaptossdk.TimelockRoleCanceller,
	}
)

var _ InspectorFactory = &aptosInspectorFactory{}

type aptosInspectorFactory struct {
	chain  fchainaptos.Chain
	action mcmstypes.TimelockAction
}

func newAptosInspectorFactory(
	chain fchainaptos.Chain, action mcmstypes.TimelockAction,
) *aptosInspectorFactory {
	return &aptosInspectorFactory{
		chain:  chain,
		action: action,
	}
}

func (f *aptosInspectorFactory) Make() (mcmssdk.Inspector, error) {
	role, ok := actionToAptosRole[f.action]
	if !ok {
		return nil, fmt.Errorf("invalid action [%s]: must be one of %v",
			f.action, slices.Collect(maps.Keys(actionToAptosRole)),
		)
	}

	return mcmsaptossdk.NewInspector(f.chain.Client, role), nil
}

//------------------------------------------------------------------------------

var _ ConverterFactory = &aptosConverterFactory{}

type aptosConverterFactory struct{}

func newAptosConverterFactory() *aptosConverterFactory {
	return &aptosConverterFactory{}
}

func (f *aptosConverterFactory) Make() (mcmssdk.TimelockConverter, error) {
	return mcmsaptossdk.NewTimelockConverter(), nil
}

//------------------------------------------------------------------------------

var _ ExecutorFactory = &aptosExecutorFactory{}

type aptosExecutorFactory struct {
	chain   fchainaptos.Chain
	encoder *mcmsaptossdk.Encoder
}

func newAptosExecutorFactory(
	chain fchainaptos.Chain, encoder *mcmsaptossdk.Encoder,
) *aptosExecutorFactory {
	return &aptosExecutorFactory{
		chain:   chain,
		encoder: encoder,
	}
}

func (f *aptosExecutorFactory) Make() (mcmssdk.Executor, error) {
	return mcmsaptossdk.NewExecutor(
		f.chain.Client,
		f.chain.DeployerSigner,
		f.encoder,
		mcmsaptossdk.TimelockRoleProposer,
	), nil
}

//------------------------------------------------------------------------------

var _ TimelockExecutorFactory = &aptosTimelockExecutorFactory{}

type aptosTimelockExecutorFactory struct {
	chain fchainaptos.Chain
}

func newAptosTimelockExecutorFactory(chain fchainaptos.Chain) *aptosTimelockExecutorFactory {
	return &aptosTimelockExecutorFactory{chain: chain}
}

func (f *aptosTimelockExecutorFactory) Make() (mcmssdk.TimelockExecutor, error) {
	return mcmsaptossdk.NewTimelockExecutor(f.chain.Client, f.chain.DeployerSigner), nil
}
