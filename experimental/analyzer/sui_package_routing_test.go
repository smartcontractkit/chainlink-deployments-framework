package analyzer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"
	mcmssuisdk "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func TestSuiPackageRoutingInputs(t *testing.T) {
	t.Parallel()

	chainSelector := chainsel.SUI_TESTNET.Selector
	genesisPackageID := "0x5ef4b483da6644c84aa78eae4f51a9bfb1fb4554d5134ac98892e931fcbdd6bf"
	latestPackageID := "0x4356995ccf08c9e6310991285249c3e382d4228a7419559b6af4a34a3a43dfa1"

	ctx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{
			chainSelector: {
				genesisPackageID: deployment.MustTypeAndVersionFromString("SuiCCIP 1.0.0"),
				latestPackageID:  deployment.MustTypeAndVersionFromString("SuiLatestCCIPPackageID 1.0.0"),
			},
		},
	}

	tests := []struct {
		name              string
		identityPackageID string
		fields            mcmssuisdk.AdditionalFields
		wantNames         []string
	}{
		{
			name:              "no routing metadata when latest matches identity",
			identityPackageID: genesisPackageID,
			fields: mcmssuisdk.AdditionalFields{
				LatestPackageID: genesisPackageID,
			},
		},
		{
			name:              "no routing metadata when latest is empty",
			identityPackageID: genesisPackageID,
			fields:            mcmssuisdk.AdditionalFields{},
		},
		{
			name:              "batch-level upgraded package routing",
			identityPackageID: genesisPackageID,
			fields: mcmssuisdk.AdditionalFields{
				LatestPackageID: latestPackageID,
			},
			wantNames: []string{
				suiExecutionPackageIDFieldName,
				suiExecutionContractTypeFieldName,
				suiExecutionContractVersionFieldName,
			},
		},
		{
			name:              "per-call upgraded package routing",
			identityPackageID: genesisPackageID,
			fields: mcmssuisdk.AdditionalFields{
				InternalLatestPackageIDs: []string{latestPackageID, ""},
			},
			wantNames: []string{suiExecutionPackageIDsFieldName},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := suiPackageRoutingInputs(ctx, chainSelector, tt.identityPackageID, tt.fields)
			require.Len(t, got, len(tt.wantNames))

			for i, name := range tt.wantNames {
				require.Equal(t, name, got[i].Name)
			}

			if len(tt.wantNames) >= 3 && tt.wantNames[0] == suiExecutionPackageIDFieldName {
				pkgField, ok := got[0].Value.(AddressField)
				require.True(t, ok)
				require.Equal(t, latestPackageID, pkgField.Value)

				typeField, ok := got[1].Value.(SimpleField)
				require.True(t, ok)
				require.Equal(t, "SuiLatestCCIPPackageID", typeField.Value)

				versionField, ok := got[2].Value.(SimpleField)
				require.True(t, ok)
				require.Equal(t, "1.0.0", versionField.Value)
			}
		})
	}
}

func TestAnalyzeSuiTransaction_UpgradedPackageRouting(t *testing.T) {
	t.Parallel()

	chainSelector := chainsel.SUI_TESTNET.Selector
	genesisPackageID := "0x5ef4b483da6644c84aa78eae4f51a9bfb1fb4554d5134ac98892e931fcbdd6bf"
	latestPackageID := "0x4356995ccf08c9e6310991285249c3e382d4228a7419559b6af4a34a3a43dfa1"

	ctx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{
			chainSelector: {
				genesisPackageID: deployment.MustTypeAndVersionFromString("SuiCCIP 1.0.0"),
				latestPackageID:  deployment.MustTypeAndVersionFromString("SuiLatestCCIPPackageID 1.0.0"),
			},
		},
	}

	mcmsTx := types.Transaction{
		To:   genesisPackageID,
		Data: []byte{0x01},
		AdditionalFields: json.RawMessage(`{
			"module_name":"rmn_remote",
			"function":"curse_multiple_with_curser_cap",
			"latest_package_id":"` + latestPackageID + `"
		}`),
	}

	result, err := AnalyzeSuiTransaction(ctx, mcmssuisdk.NewDecoder(), chainSelector, mcmsTx)
	require.NoError(t, err)
	require.Equal(t, genesisPackageID, result.Address)
	require.Equal(t, "SuiCCIP", result.ContractType)

	require.GreaterOrEqual(t, len(result.Inputs), 3)
	require.Equal(t, suiExecutionPackageIDFieldName, result.Inputs[0].Name)
	require.Equal(t, latestPackageID, result.Inputs[0].Value.(AddressField).Value)
	require.Equal(t, suiExecutionContractTypeFieldName, result.Inputs[1].Name)
	require.Equal(t, "SuiLatestCCIPPackageID", result.Inputs[1].Value.(SimpleField).Value)
	require.Equal(t, suiExecutionContractVersionFieldName, result.Inputs[2].Name)
	require.Equal(t, "1.0.0", result.Inputs[2].Value.(SimpleField).Value)
}
