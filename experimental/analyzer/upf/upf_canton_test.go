package upf

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"
	factory "github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/ccip/factory"
	mcmscantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	mcmsanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

// TestBatchOperationsToUpfDecodedCalls_Canton proves the UPF batch-decode path produces a decoded
// Canton call (previously the default branch emitted "canton transaction decoding is not supported").
func TestBatchOperationsToUpfDecodedCalls_Canton(t *testing.T) {
	t.Parallel()

	deployParams := factory.DeployRMNRemoteParams{
		InstanceId: "rmn-remote-1",
		RmnOwner:   "alice::abc123",
		CcipOwner:  "bob::def456",
	}
	rawHex, err := deployParams.MarshalHex()
	require.NoError(t, err)

	af, err := json.Marshal(mcmscantonsdk.AdditionalFields{
		TargetInstanceAddress: "ccip-factory-1@alice::abc123",
		FunctionName:          "DeployRMNRemote",
		TargetTemplateID:      "#pkg:CCIP.Factory:CCIPFactory",
	})
	require.NoError(t, err)

	batches := []mcmstypes.BatchOperation{
		{
			ChainSelector: mcmstypes.ChainSelector(chainsel.CANTON_TESTNET.Selector),
			Transactions: []mcmstypes.Transaction{
				{
					To:               "0xfeed",
					Data:             []byte(rawHex), // go-daml MarshalHex returns the raw operation bytes
					AdditionalFields: af,
				},
			},
		},
	}

	proposalCtx := &mcmsanalyzer.DefaultProposalContext{AddressesByChain: deployment.AddressesByChain{}}

	decoded, err := batchOperationsToUpfDecodedCalls(t.Context(), proposalCtx, deployment.Environment{}, batches)
	require.NoError(t, err)
	require.Len(t, decoded, 1)
	require.Len(t, decoded[0], 1)
	require.NotNil(t, decoded[0][0].Data)
	require.Equal(t, "CCIPFactory::DeployRMNRemote", decoded[0][0].Data.FunctionName)
}
