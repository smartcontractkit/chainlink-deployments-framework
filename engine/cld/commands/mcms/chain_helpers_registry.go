package mcms

import (
	"fmt"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	cldfmcmsadapters "github.com/smartcontractkit/chainlink-deployments-framework/chain/mcms/adapters"
)

type chainInspectorBuilder func(
	acc *cldfmcmsadapters.ChainAccessAdapter,
	action types.TimelockAction,
	chainSelector types.ChainSelector,
	chainMetadata types.ChainMetadata,
) (sdk.Inspector, error)

type chainExecutorBuilder func(
	acc *cldfmcmsadapters.ChainAccessAdapter,
	action types.TimelockAction,
	chainSelector types.ChainSelector,
	encoder sdk.Encoder,
	chainMetadata types.ChainMetadata,
) (sdk.Executor, error)

type chainTimelockExecutorBuilder func(
	acc *cldfmcmsadapters.ChainAccessAdapter,
	action types.TimelockAction,
	chainSelector types.ChainSelector,
	chainMetadata types.ChainMetadata,
) (sdk.TimelockExecutor, error)

var (
	chainFamilyInspectorBuilders        = map[string]chainInspectorBuilder{}
	chainFamilyExecutorBuilders         = map[string]chainExecutorBuilder{}
	chainFamilyTimelockExecutorBuilders = map[string]chainTimelockExecutorBuilder{}
)

// registerChainFamilyBuilders wires one chain family's builders into the dispatch maps.
// Call from init() in each chain_helpers_<family>.go file.
func registerChainFamilyBuilders(
	family string,
	inspector chainInspectorBuilder,
	executor chainExecutorBuilder,
	timelockExecutor chainTimelockExecutorBuilder,
) {
	if _, dup := chainFamilyInspectorBuilders[family]; dup {
		panic(fmt.Sprintf("mcms: duplicate chain family registration: %q", family))
	}
	chainFamilyInspectorBuilders[family] = inspector
	chainFamilyExecutorBuilders[family] = executor
	chainFamilyTimelockExecutorBuilders[family] = timelockExecutor
}
