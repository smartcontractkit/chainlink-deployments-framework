package mcms

import (
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms/chainwrappers"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	cldfmcmsadapters "github.com/smartcontractkit/chainlink-deployments-framework/chain/mcms/adapters"
)

func init() {
	registerChainFamilyBuilders(
		chainsel.FamilyTon,
		buildTonInspector,
		buildTonExecutor,
		buildTonTimelockExecutor,
	)
}

func buildTonInspector(
	acc *cldfmcmsadapters.ChainAccessAdapter,
	action types.TimelockAction,
	chainSelector types.ChainSelector,
	chainMetadata types.ChainMetadata,
) (sdk.Inspector, error) {
	return chainwrappers.BuildInspector(acc, chainSelector, action, chainMetadata)
}

func buildTonExecutor(
	acc *cldfmcmsadapters.ChainAccessAdapter,
	action types.TimelockAction,
	chainSelector types.ChainSelector,
	encoder sdk.Encoder,
	chainMetadata types.ChainMetadata,
) (sdk.Executor, error) {
	return chainwrappers.BuildExecutor(acc, chainSelector, encoder, action, chainMetadata)
}

func buildTonTimelockExecutor(
	acc *cldfmcmsadapters.ChainAccessAdapter,
	action types.TimelockAction,
	chainSelector types.ChainSelector,
	chainMetadata types.ChainMetadata,
) (sdk.TimelockExecutor, error) {
	return chainwrappers.BuildTimelockExecutor(acc, chainSelector, action, chainMetadata)
}
