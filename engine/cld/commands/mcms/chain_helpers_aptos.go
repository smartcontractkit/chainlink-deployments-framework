package mcms

import (
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms/chainwrappers"
	"github.com/smartcontractkit/mcms/sdk"
	mcmsaptos "github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/types"

	cldfmcmsadapters "github.com/smartcontractkit/chainlink-deployments-framework/chain/mcms/adapters"
)

func init() {
	registerChainFamilyBuilders(
		chainsel.FamilyAptos,
		buildAptosInspector,
		buildAptosExecutor,
		buildAptosTimelockExecutor,
	)
}

// aptosRoleFromProposal maps the timelock action to the Aptos MCMS role. Use this (or extend it)
// when CLDF needs Aptos-specific role logic without going through chainwrappers.
func aptosRoleFromProposal(action types.TimelockAction) (mcmsaptos.TimelockRole, error) {
	return mcmsaptos.AptosRoleFromAction(action)
}

func buildAptosInspector(
	acc *cldfmcmsadapters.ChainAccessAdapter,
	action types.TimelockAction,
	chainSelector types.ChainSelector,
	chainMetadata types.ChainMetadata,
) (sdk.Inspector, error) {
	if _, err := aptosRoleFromProposal(action); err != nil {
		return nil, err
	}

	return chainwrappers.BuildInspector(acc, chainSelector, action, chainMetadata)
}

func buildAptosExecutor(
	acc *cldfmcmsadapters.ChainAccessAdapter,
	action types.TimelockAction,
	chainSelector types.ChainSelector,
	encoder sdk.Encoder,
	chainMetadata types.ChainMetadata,
) (sdk.Executor, error) {
	if _, err := aptosRoleFromProposal(action); err != nil {
		return nil, err
	}

	return chainwrappers.BuildExecutor(acc, chainSelector, encoder, action, chainMetadata)
}

func buildAptosTimelockExecutor(
	acc *cldfmcmsadapters.ChainAccessAdapter,
	action types.TimelockAction,
	chainSelector types.ChainSelector,
	chainMetadata types.ChainMetadata,
) (sdk.TimelockExecutor, error) {
	if _, err := aptosRoleFromProposal(action); err != nil {
		return nil, err
	}

	return chainwrappers.BuildTimelockExecutor(acc, chainSelector, action, chainMetadata)
}
