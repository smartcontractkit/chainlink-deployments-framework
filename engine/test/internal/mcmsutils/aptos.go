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
	// actionToAptosRole maps MCMS timelock actions to their corresponding Aptos timelock roles.
	// This mapping is used to determine the appropriate role when creating Aptos inspectors
	// and executors for different types of timelock operations.
	//
	// Note: Only Aptos requires a role to instantiate an inspector. All other chains implement
	// their contracts in a sane way so that they can be inspected without a role.
	actionToAptosRole = map[mcmstypes.TimelockAction]mcmsaptossdk.TimelockRole{
		mcmstypes.TimelockActionSchedule: mcmsaptossdk.TimelockRoleProposer,
		mcmstypes.TimelockActionBypass:   mcmsaptossdk.TimelockRoleBypasser,
		mcmstypes.TimelockActionCancel:   mcmsaptossdk.TimelockRoleCanceller,
	}
)

var _ InspectorFactory = &aptosInspectorFactory{}

// aptosInspectorFactory is a factory for creating Aptos-specific MCMS inspectors.
// It implements the InspectorFactory interface and is responsible for creating
// inspectors that can examine the state of MCMS and Timelock contracts on the Aptos blockchain.
type aptosInspectorFactory struct {
	chain  fchainaptos.Chain        // The Aptos chain configuration and client
	action mcmstypes.TimelockAction // The timelock action that determines the inspector role
}

// newAptosInspectorFactory creates a new Aptos inspector factory.
func newAptosInspectorFactory(
	chain fchainaptos.Chain, action mcmstypes.TimelockAction,
) *aptosInspectorFactory {
	return &aptosInspectorFactory{
		chain:  chain,
		action: action,
	}
}

// Make creates and returns a new Aptos MCMS inspector.
func (f *aptosInspectorFactory) Make() (mcmssdk.Inspector, error) {
	role, ok := actionToAptosRole[f.action]
	if !ok {
		return nil, fmt.Errorf("invalid action [%s]: must be one of %v",
			f.action, slices.Collect(maps.Keys(actionToAptosRole)),
		)
	}

	return mcmsaptossdk.NewInspector(f.chain.Client, role), nil
}

var _ ConverterFactory = &aptosConverterFactory{}

// aptosConverterFactory is a factory for creating Aptos-specific timelock converters.
// It implements the ConverterFactory interface and creates converters that can
// transform MCMS timelock proposals into a standard MCMS proposal.
type aptosConverterFactory struct{}

// newAptosConverterFactory creates a new Aptos converter factory.
func newAptosConverterFactory() *aptosConverterFactory {
	return &aptosConverterFactory{}
}

// Make creates and returns a new Aptos timelock converter.
func (f *aptosConverterFactory) Make() (mcmssdk.TimelockConverter, error) {
	return mcmsaptossdk.NewTimelockConverter(), nil
}

var _ ExecutorFactory = &aptosExecutorFactory{}

// aptosExecutorFactory is a factory for creating Aptos-specific MCMS executors.
// It implements the ExecutorFactory interface and creates executors that can
// execute MCMS operations on the Aptos blockchain.
type aptosExecutorFactory struct {
	chain   fchainaptos.Chain     // The Aptos chain configuration and client
	encoder *mcmsaptossdk.Encoder // The encoder for creating Aptos-specific transaction data
}

// newAptosExecutorFactory creates a new Aptos executor factory.
func newAptosExecutorFactory(
	chain fchainaptos.Chain, encoder *mcmsaptossdk.Encoder,
) *aptosExecutorFactory {
	return &aptosExecutorFactory{
		chain:   chain,
		encoder: encoder,
	}
}

// Make creates and returns a new Aptos MCMS executor.
func (f *aptosExecutorFactory) Make() (mcmssdk.Executor, error) {
	return mcmsaptossdk.NewExecutor(
		f.chain.Client,
		f.chain.DeployerSigner,
		f.encoder,
		mcmsaptossdk.TimelockRoleProposer,
	), nil
}

var _ TimelockExecutorFactory = &aptosTimelockExecutorFactory{}

// aptosTimelockExecutorFactory is a factory for creating Aptos-specific timelock executors.
// It implements the TimelockExecutorFactory interface and creates executors specifically
// designed for executing Timelock operations on the Aptos blockchain.
type aptosTimelockExecutorFactory struct {
	chain fchainaptos.Chain // The Aptos chain configuration and client
}

// newAptosTimelockExecutorFactory creates a new Aptos timelock executor factory.
func newAptosTimelockExecutorFactory(chain fchainaptos.Chain) *aptosTimelockExecutorFactory {
	return &aptosTimelockExecutorFactory{chain: chain}
}

// Make creates and returns a new Aptos timelock executor.
func (f *aptosTimelockExecutorFactory) Make() (mcmssdk.TimelockExecutor, error) {
	return mcmsaptossdk.NewTimelockExecutor(f.chain.Client, f.chain.DeployerSigner), nil
}
