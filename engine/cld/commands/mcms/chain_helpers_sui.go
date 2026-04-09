package mcms

import (
	"github.com/smartcontractkit/mcms/chainwrappers"
	"github.com/smartcontractkit/mcms/sdk"
	mcmssui "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"

	cldfmcmsadapters "github.com/smartcontractkit/chainlink-deployments-framework/chain/mcms/adapters"
)

// suiMetadataFromProposal parses Sui-specific fields from MCMS chain metadata.
func suiMetadataFromProposal(metadata types.ChainMetadata) (mcmssui.AdditionalFieldsMetadata, error) {
	return mcmssui.SuiMetadata(metadata)
}

func buildSuiInspector(
	acc *cldfmcmsadapters.ChainAccessAdapter,
	action types.TimelockAction,
	chainSelector types.ChainSelector,
	chainMetadata types.ChainMetadata,
) (sdk.Inspector, error) {
	if _, err := suiMetadataFromProposal(chainMetadata); err != nil {
		return nil, err
	}

	return chainwrappers.BuildInspector(acc, chainSelector, action, chainMetadata)
}

func buildSuiExecutor(
	acc *cldfmcmsadapters.ChainAccessAdapter,
	action types.TimelockAction,
	chainSelector types.ChainSelector,
	encoder sdk.Encoder,
	chainMetadata types.ChainMetadata,
) (sdk.Executor, error) {
	if _, err := suiMetadataFromProposal(chainMetadata); err != nil {
		return nil, err
	}

	return chainwrappers.BuildExecutor(acc, chainSelector, encoder, action, chainMetadata)
}

func buildSuiTimelockExecutor(
	acc *cldfmcmsadapters.ChainAccessAdapter,
	action types.TimelockAction,
	chainSelector types.ChainSelector,
	chainMetadata types.ChainMetadata,
) (sdk.TimelockExecutor, error) {
	if _, err := suiMetadataFromProposal(chainMetadata); err != nil {
		return nil, err
	}

	return chainwrappers.BuildTimelockExecutor(acc, chainSelector, action, chainMetadata)
}
