package mcms

import (
	"github.com/smartcontractkit/mcms/chainwrappers"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	cldfmcmsadapters "github.com/smartcontractkit/chainlink-deployments-framework/chain/mcms/adapters"
)

func buildSolanaInspector(
	acc *cldfmcmsadapters.ChainAccessAdapter,
	action types.TimelockAction,
	chainSelector types.ChainSelector,
	chainMetadata types.ChainMetadata,
) (sdk.Inspector, error) {
	return chainwrappers.BuildInspector(acc, chainSelector, action, chainMetadata)
}

func buildSolanaExecutor(
	acc *cldfmcmsadapters.ChainAccessAdapter,
	action types.TimelockAction,
	chainSelector types.ChainSelector,
	encoder sdk.Encoder,
	chainMetadata types.ChainMetadata,
) (sdk.Executor, error) {
	return chainwrappers.BuildExecutor(acc, chainSelector, encoder, action, chainMetadata)
}

func buildSolanaTimelockExecutor(
	acc *cldfmcmsadapters.ChainAccessAdapter,
	action types.TimelockAction,
	chainSelector types.ChainSelector,
	chainMetadata types.ChainMetadata,
) (sdk.TimelockExecutor, error) {
	return chainwrappers.BuildTimelockExecutor(acc, chainSelector, action, chainMetadata)
}
