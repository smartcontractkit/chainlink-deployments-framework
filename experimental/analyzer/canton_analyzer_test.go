package analyzer

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"
	core "github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/ccip/core"
	factory "github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/ccip/factory"
	chainlinkapi "github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/chainlink/chainlinkapi"
	cantontypes "github.com/smartcontractkit/go-daml/pkg/types"
	"github.com/smartcontractkit/mcms"
	mcmscantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

const cantonFactoryAddress = "0x0000000000000000000000000000000000000000000000000000000000000abc"

// cantonOperationData returns the raw operation bytes for a generated choice-argument struct, as
// stored in tx.Data by the Canton MCMS encoder. go-daml's MarshalHex returns the raw encoded bytes
// as a string in the pinned version; decode first if a future version returns a hex string.
func cantonOperationData(t *testing.T, v interface{ MarshalHex() (string, error) }) []byte {
	t.Helper()
	s, err := v.MarshalHex()
	require.NoError(t, err)

	return []byte(s)
}

func TestAnalyzeCantonTransactions(t *testing.T) {
	t.Parallel()

	chainSelector := chainsel.CANTON_TESTNET.Selector
	defaultProposalCtx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{
			chainSelector: {
				cantonFactoryAddress: deployment.MustTypeAndVersionFromString("CCIPFactory 0.1.0"),
			},
		},
	}

	deployParams := factory.DeployRMNRemoteParams{
		InstanceId:      "rmn-remote-1",
		RmnOwner:        "alice::abc123",
		CcipOwner:       "bob::def456",
		CustomObservers: []cantontypes.PARTY{"carol::ghi789"},
		CursedSubjects:  []cantontypes.TEXT{"0x01"},
	}

	tx := types.Transaction{
		To:   cantonFactoryAddress,
		Data: cantonOperationData(t, deployParams),
		AdditionalFields: mustMarshalJSON(t, mcmscantonsdk.AdditionalFields{
			TargetInstanceAddress: "ccip-factory-1@alice::abc123",
			FunctionName:          "DeployRMNRemote",
			TargetTemplateID:      "#pkg:CCIP.Factory:CCIPFactory",
		}),
	}

	decoder := mcmscantonsdk.NewDecoder()
	result, err := AnalyzeCantonTransaction(defaultProposalCtx, decoder, chainSelector, tx)
	require.NoError(t, err)
	require.NotNil(t, result)

	require.Equal(t, cantonFactoryAddress, result.Address)
	require.Equal(t, "CCIPFactory::DeployRMNRemote", result.Method)
	require.Equal(t, "CCIPFactory", result.ContractType)

	require.Len(t, result.Inputs, 5)
	require.Equal(t, "instanceId", result.Inputs[0].Name)
	require.Equal(t, SimpleField{Value: "rmn-remote-1"}, result.Inputs[0].Value)
	require.Equal(t, "rmnOwner", result.Inputs[1].Name)
	require.Equal(t, SimpleField{Value: "alice::abc123"}, result.Inputs[1].Value)
	require.Equal(t, "ccipOwner", result.Inputs[2].Name)
	require.Equal(t, "customObservers", result.Inputs[3].Name)
	require.Equal(t, "ArrayField", result.Inputs[3].Value.GetType())
}

// TestAnalyzeCantonTransaction_DecodeErrorIsNonFatal verifies an undecodable operation surfaces the
// error in the Method field rather than failing the whole proposal.
func TestAnalyzeCantonTransaction_DecodeErrorIsNonFatal(t *testing.T) {
	t.Parallel()

	chainSelector := chainsel.CANTON_TESTNET.Selector
	ctx := &DefaultProposalContext{AddressesByChain: deployment.AddressesByChain{}}

	tx := types.Transaction{
		To:   cantonFactoryAddress,
		Data: []byte{0x01, 0x02, 0x03},
		AdditionalFields: mustMarshalJSON(t, mcmscantonsdk.AdditionalFields{
			TargetInstanceAddress: "x@alice::abc",
			FunctionName:          "NotARealChoice",
			TargetTemplateID:      "#pkg:CCIP.RMNRemote:RMNRemote",
		}),
	}

	result, err := AnalyzeCantonTransaction(ctx, mcmscantonsdk.NewDecoder(), chainSelector, tx)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Method, "NotARealChoice")
}

// TestBuildTimelockReport_Canton exercises the full path: a Canton timelock proposal flows through
// the family dispatcher in analyzeTransactions and produces a decoded call (no longer empty).
func TestBuildTimelockReport_Canton(t *testing.T) {
	t.Parallel()

	chainSelector := chainsel.CANTON_TESTNET.Selector
	proposalCtx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{
			chainSelector: {
				cantonFactoryAddress: deployment.MustTypeAndVersionFromString("CCIPFactory 0.1.0"),
			},
		},
		renderer: NewMarkdownRenderer(),
	}

	deployParams := factory.DeployRMNRemoteParams{
		InstanceId: "rmn-remote-1",
		RmnOwner:   "alice::abc123",
		CcipOwner:  "bob::def456",
	}

	proposal := &mcms.TimelockProposal{
		Operations: []types.BatchOperation{
			{
				ChainSelector: types.ChainSelector(chainSelector),
				Transactions: []types.Transaction{
					{
						To:   cantonFactoryAddress,
						Data: cantonOperationData(t, deployParams),
						AdditionalFields: mustMarshalJSON(t, mcmscantonsdk.AdditionalFields{
							TargetInstanceAddress: "ccip-factory-1@alice::abc123",
							FunctionName:          "DeployRMNRemote",
							TargetTemplateID:      "#pkg:CCIP.Factory:CCIPFactory",
						}),
					},
				},
			},
		},
	}

	report, err := BuildTimelockReport(t.Context(), proposalCtx, deployment.Environment{}, proposal)
	require.NoError(t, err)
	require.Len(t, report.Batches, 1)
	require.Len(t, report.Batches[0].Operations, 1)

	calls := report.Batches[0].Operations[0].Calls
	require.Len(t, calls, 1)
	require.Equal(t, "CCIPFactory::DeployRMNRemote", calls[0].Method)
	require.Equal(t, "CCIPFactory", calls[0].ContractType)
}

//	proposal JSON -> mcms.NewTimelockProposal -> BuildTimelockReport (family dispatch + Canton decode)
//	             -> DescribeTimelockProposal (Markdown render, i.e. the .decoded.md a reviewer sees)
//
// on real GlobalConfig / Executor / FeeQuoter operationData.
func TestBuildTimelockReport_CantonTestProposal(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("testdata/canton_test_proposal.json")
	require.NoError(t, err)

	// Parse exactly as the product does (engine/cld/mcms/proposalanalysis/decoder).
	proposal, err := mcms.NewTimelockProposal(bytes.NewReader(data))
	require.NoError(t, err, "fixture must be a valid MCMS TimelockProposal")

	proposalCtx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{},
		renderer:         NewMarkdownRenderer(),
	}

	// 1) Decode: build the report and check every tx decoded to its Daml choice
	// (MethodName is ContractEntity::Choice) — no decode errors in any Method.
	report, err := BuildTimelockReport(t.Context(), proposalCtx, deployment.Environment{}, proposal)
	require.NoError(t, err)

	wantMethods := []string{
		"GlobalConfig::ApplyDestChainConfigUpdates",
		"Executor::ApplyDestChainUpdates",
		"FeeQuoter::ApplyFeeQuoterDestChainConfigUpdates",
		"FeeQuoter::ApplyPriceUpdatersUpdate",
		"FeeQuoter::UpdatePrices",
		"GlobalConfig::ApplySourceChainConfigUpdates",
	}

	var methods []string
	for _, batch := range report.Batches {
		for _, op := range batch.Operations {
			for _, call := range op.Calls {
				methods = append(methods, call.Method)
			}
		}
	}
	require.ElementsMatch(t, wantMethods, methods)

	// 2) Render: produce the Markdown a reviewer sees (the .decoded.md content) and confirm each
	// decoded call appears in it. This covers the full proposal -> decode -> render pipeline.
	md, err := DescribeTimelockProposal(t.Context(), proposalCtx, deployment.Environment{}, proposal)
	require.NoError(t, err)
	require.NotEmpty(t, md)
	for _, m := range wantMethods {
		require.Contains(t, md, m, "rendered markdown should contain decoded call %q", m)
	}
}

// TestAnalyzeCantonTransactions_PerField is a table-driven test for AnalyzeCantonTransaction that
// covers four distinct structural shapes of Canton operationData:
//
//   - Single scalar field: a Daml type alias (NUMERIC) decodes to a SimpleField.
//   - Scalars + slices: TEXT/PARTY scalars become SimpleField; []PARTY/[]TEXT become ArrayField.
//     Exact string values are asserted to confirm toDisplayArg strips Daml type aliases.
//   - Array of nested struct: []SourceChainConfigArgs (each element has scalars, slices, and a
//     sub-struct) exercises the toDisplayArg recursion through nested Daml records → ArrayField.
//   - Two slice fields: both populated and empty slices decode to ArrayField.
//
// Each case asserts the decoded Method (ContractEntity::FunctionName), input count, each input
// Name, and each input FieldValue type. Deep contents of nested fields are not asserted here —
// those are covered by TestBuildTimelockReport_CantonTestProposal on real proposal bytes.
func TestAnalyzeCantonTransactions_PerField(t *testing.T) {
	t.Parallel()

	const (
		globalConfigAddr = "0x0000000000000000000000000000000000000000000000000000000000000001"
		feeQuoterAddr    = "0x0000000000000000000000000000000000000000000000000000000000000002"
	)
	chainSelector := chainsel.CANTON_TESTNET.Selector
	ctx := &DefaultProposalContext{AddressesByChain: deployment.AddressesByChain{}}
	decoder := mcmscantonsdk.NewDecoder()

	type inputCheck struct {
		name      string
		fieldType string // FieldValue.GetType()
	}

	tests := []struct {
		name   string
		tx     types.Transaction
		method string
		inputs []inputCheck
	}{
		{
			// Simple single-field scalar: NUMERIC("1234") → string "1234" → SimpleField
			name: "IsCursedForChain — single NUMERIC field",
			tx: types.Transaction{
				To:   globalConfigAddr,
				Data: cantonOperationData(t, core.IsCursedForChainMCMSParams{ChainSelector: "1234"}),
				AdditionalFields: mustMarshalJSON(t, mcmscantonsdk.AdditionalFields{
					TargetInstanceAddress: "rmn-remote-1@alice::abc",
					FunctionName:          "IsCursedForChain",
					TargetTemplateID:      "#pkg:CCIP.RMNRemote:RMNRemote",
				}),
			},
			method: "RMNRemote::IsCursedForChain",
			inputs: []inputCheck{
				{name: "chainSelector", fieldType: "SimpleField"},
			},
		},
		{
			// Deploy: TEXT/PARTY scalars + slice fields
			name: "DeployRMNRemote — scalars and slices",
			tx: types.Transaction{
				To: globalConfigAddr,
				Data: cantonOperationData(t, factory.DeployRMNRemoteParams{
					InstanceId:      "rmn-1",
					RmnOwner:        "alice::abc",
					CcipOwner:       "bob::def",
					CustomObservers: []cantontypes.PARTY{"carol::ghi"},
					CursedSubjects:  []cantontypes.TEXT{"0x01"},
				}),
				AdditionalFields: mustMarshalJSON(t, mcmscantonsdk.AdditionalFields{
					TargetInstanceAddress: "factory-1@alice::abc",
					FunctionName:          "DeployRMNRemote",
					TargetTemplateID:      "#pkg:CCIP.Factory:CCIPFactory",
				}),
			},
			method: "CCIPFactory::DeployRMNRemote",
			inputs: []inputCheck{
				{name: "instanceId", fieldType: "SimpleField"},
				{name: "rmnOwner", fieldType: "SimpleField"},
				{name: "ccipOwner", fieldType: "SimpleField"},
				{name: "customObservers", fieldType: "ArrayField"},
				{name: "cursedSubjects", fieldType: "ArrayField"},
			},
		},
		{
			// Nested struct: ApplySourceChainConfigUpdates wraps a []SourceChainConfigArgs,
			// which includes a NUMERIC, BOOL, []TEXT, and []RawInstanceAddress (struct) fields —
			// exercises the array-of-nested-struct path without requiring binary raw-bytes fields.
			name: "ApplySourceChainConfigUpdates — array of nested struct",
			tx: types.Transaction{
				To: globalConfigAddr,
				Data: cantonOperationData(t, core.ApplySourceChainConfigUpdatesParams{
					SourceChainConfigArgs: []core.SourceChainConfigArgs{
						{
							SourceChainSelector: cantontypes.NUMERIC("16015286601757825753"),
							IsEnabled:           cantontypes.BOOL(true),
							OnRampAddresses:     []cantontypes.TEXT{"0xdeadbeef"},
							DefaultCCVs: []chainlinkapi.RawInstanceAddress{
								{Unpack: cantontypes.TEXT("ccv-1@alice::abc")},
							},
							LaneMandatedCCVs: []chainlinkapi.RawInstanceAddress{},
						},
					},
				}),
				AdditionalFields: mustMarshalJSON(t, mcmscantonsdk.AdditionalFields{
					TargetInstanceAddress: "globalconfig-1@alice::abc",
					FunctionName:          "ApplySourceChainConfigUpdates",
					TargetTemplateID:      "#pkg:CCIP.GlobalConfig:GlobalConfig",
				}),
			},
			method: "GlobalConfig::ApplySourceChainConfigUpdates",
			inputs: []inputCheck{
				{name: "sourceChainConfigArgs", fieldType: "ArrayField"},
			},
		},
		{
			// Two scalar slice fields — verify both are decoded
			name: "ApplyPriceUpdatersUpdate — two slice fields",
			tx: types.Transaction{
				To: feeQuoterAddr,
				Data: cantonOperationData(t, core.ApplyPriceUpdatersUpdateParams{
					AddedPriceUpdaters:   []cantontypes.PARTY{"alice::abc"},
					RemovedPriceUpdaters: []cantontypes.PARTY{},
				}),
				AdditionalFields: mustMarshalJSON(t, mcmscantonsdk.AdditionalFields{
					TargetInstanceAddress: "feequoter-1@alice::abc",
					FunctionName:          "ApplyPriceUpdatersUpdate",
					TargetTemplateID:      "#pkg:CCIP.FeeQuoter:FeeQuoter",
				}),
			},
			method: "FeeQuoter::ApplyPriceUpdatersUpdate",
			inputs: []inputCheck{
				{name: "addedPriceUpdaters", fieldType: "ArrayField"},
				{name: "removedPriceUpdaters", fieldType: "ArrayField"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := AnalyzeCantonTransaction(ctx, decoder, chainSelector, tt.tx)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, tt.method, result.Method)
			require.Len(t, result.Inputs, len(tt.inputs), "input count mismatch for %s", tt.method)

			for i, want := range tt.inputs {
				require.Equal(t, want.name, result.Inputs[i].Name, "input[%d] name", i)
				require.Equal(t, want.fieldType, result.Inputs[i].Value.GetType(),
					"input[%d] %q type", i, want.name)
			}
		})
	}
}

func TestAnalyzeCantonTransaction_ErrorCases(t *testing.T) {
	t.Parallel()

	chainSelector := chainsel.CANTON_TESTNET.Selector
	ctx := &DefaultProposalContext{AddressesByChain: deployment.AddressesByChain{}}
	decoder := mcmscantonsdk.NewDecoder()

	tests := []struct {
		name               string
		tx                 types.Transaction
		wantHardErr        bool   // expect err != nil from AnalyzeCantonTransaction
		wantMethodContains string // non-fatal: expect error surfaced in Method field
	}{
		{
			name: "invalid AdditionalFields JSON — hard error",
			tx: types.Transaction{
				To:               "0xaddr",
				Data:             []byte{0x01},
				AdditionalFields: json.RawMessage(`invalid json`),
			},
			wantHardErr: true,
		},
		{
			name: "unknown choice on known contract — non-fatal, choice name in Method",
			tx: types.Transaction{
				To:   "0xaddr",
				Data: []byte{0x01, 0x02},
				AdditionalFields: mustMarshalJSON(t, mcmscantonsdk.AdditionalFields{
					TargetInstanceAddress: "x@alice::abc",
					FunctionName:          "NotARealChoice",
					TargetTemplateID:      "#pkg:CCIP.RMNRemote:RMNRemote",
				}),
			},
			wantMethodContains: "NotARealChoice",
		},
		{
			name: "version-skew — correct choice name, bytes from an older binding — non-fatal",
			// operationData that decodes no current binding type (round-trips against none)
			tx: types.Transaction{
				To:   "0xaddr",
				Data: []byte{0x0e, 0x65, 0x78, 0x65, 0x63, 0x75, 0x74, 0x6f, 0x72},
				AdditionalFields: mustMarshalJSON(t, mcmscantonsdk.AdditionalFields{
					TargetInstanceAddress: "factory-1@alice::abc",
					FunctionName:          "DeployExecutor",
					TargetTemplateID:      "#pkg:CCIP.Factory:CCIPFactory",
				}),
			},
			wantMethodContains: "DeployExecutor",
		},
		{
			name: "empty Data — non-fatal, function name surfaced in Method",
			tx: types.Transaction{
				To:   "0xaddr",
				Data: []byte{},
				AdditionalFields: mustMarshalJSON(t, mcmscantonsdk.AdditionalFields{
					TargetInstanceAddress: "globalconfig-1@alice::abc",
					FunctionName:          "ApplyDestChainConfigUpdates",
					TargetTemplateID:      "#pkg:CCIP.GlobalConfig:GlobalConfig",
				}),
			},
			wantMethodContains: "ApplyDestChainConfigUpdates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := AnalyzeCantonTransaction(ctx, decoder, chainSelector, tt.tx)

			if tt.wantHardErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Method, tt.wantMethodContains)
		})
	}
}

func mustMarshalJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)

	return b
}
