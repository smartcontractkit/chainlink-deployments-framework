package analyzer

import (
	"encoding/json"
	"os"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	aptosbind "github.com/smartcontractkit/chainlink-aptos/bindings/bind"
	mcmsaptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

const aptosTestAddress = "0xe86f0e5a8b9cb6ab31b656baa83a0d2eb761b32eb31b9a9c74abb7d0cffd26fa"

func TestAnalyzeAptosTransactions(t *testing.T) {
	t.Parallel()

	defaultProposalCtx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{
			chainsel.APTOS_TESTNET.Selector: {
				aptosTestAddress: deployment.MustTypeAndVersionFromString("TestCCIP 1.0.0"),
			},
		},
	}

	chainSelector := chainsel.APTOS_TESTNET.Selector

	tests := []struct {
		name       string
		operations []types.BatchOperation
		want       []*DecodedCall
		wantErr    bool
	}{
		{
			name:       "Single operation",
			operations: getOperations(1),
			want: []*DecodedCall{
				{
					Address: aptosTestAddress,
					Method:  "ccip_onramp::onramp::initialize",
					Inputs: []NamedField{
						{Name: "chain_selector", Value: SimpleField{Value: "4457093679053095497"}},
						{Name: "fee_aggregator", Value: AddressField{Value: "0x13a9f1a109368730f2e355d831ba8fbf5942fb82321863d55de54cb4ebe5d18f"}},
						{Name: "allowlist_admin", Value: AddressField{Value: "0x13a9f1a109368730f2e355d831ba8fbf5942fb82321863d55de54cb4ebe5d18f"}},
						{Name: "dest_chain_selectors", Value: ArrayField{Elements: nil}},
						{Name: "dest_chain_routers", Value: ArrayField{Elements: nil}},
						{Name: "dest_chain_allowlist_enabled", Value: ArrayField{Elements: nil}},
					},
				},
				{
					Address: aptosTestAddress,
					Method:  "ccip_offramp::offramp::initialize",
					Inputs: []NamedField{
						{Name: "chain_selector", Value: SimpleField{Value: "4457093679053095497"}},
						{Name: "permissionless_execution_threshold_seconds", Value: SimpleField{Value: "28800"}},
						{Name: "source_chains_selector", Value: ArrayField{Elements: []FieldValue{SimpleField{Value: "11155111"}}}},
						{Name: "source_chains_is_enabled", Value: ArrayField{Elements: []FieldValue{SimpleField{Value: "true"}}}},
						{Name: "source_chains_is_rmn_verification_disabled", Value: ArrayField{Elements: []FieldValue{SimpleField{Value: "false"}}}},
						{Name: "source_chains_on_ramp", Value: ArrayField{Elements: []FieldValue{BytesField{Value: []byte{0xb, 0xf3, 0xde, 0x8c, 0x5d, 0x3e, 0x8a, 0x2b, 0x34, 0xd2, 0xbe, 0xeb, 0x17, 0xab, 0xfc, 0xeb, 0xaf, 0x36, 0x3a, 0x59}}}}},
					},
				},
				{
					Address: aptosTestAddress,
					Method:  "ccip::rmn_remote::initialize",
					Inputs: []NamedField{
						{Name: "local_chain_selector", Value: SimpleField{Value: "4457093679053095497"}},
					},
				},
				{
					Address: aptosTestAddress,
					Method:  "ccip_token_pool::token_pool::initialize",
					Inputs: []NamedField{
						{Name: "local_token", Value: AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000003"}},
						{Name: "allowlist", Value: ArrayField{Elements: []FieldValue{AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000001"}, AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000002"}}}},
					},
				},
				{
					Address: aptosTestAddress,
					Method:  "ccip_offramp::offramp::apply_source_chain_config_updates",
					Inputs: []NamedField{
						{Name: "source_chains_selector", Value: ArrayField{Elements: []FieldValue{SimpleField{Value: "743186221051783445"}, SimpleField{Value: "16015286601757825753"}}}},
						{Name: "source_chains_is_enabled", Value: ArrayField{Elements: []FieldValue{SimpleField{Value: "true"}, SimpleField{Value: "false"}}}},
						{Name: "source_chains_is_rmn_verification_disabled", Value: ArrayField{Elements: []FieldValue{SimpleField{Value: "true"}, SimpleField{Value: "true"}}}},
						{Name: "source_chains_on_ramp", Value: ArrayField{Elements: []FieldValue{BytesField{Value: []byte{0xc2, 0x30, 0x71, 0xa8, 0xae, 0x83, 0x67, 0x1f, 0x37, 0xbd, 0xa1, 0xda, 0xdb, 0xc7, 0x45, 0xa9, 0x78, 0xf, 0x63, 0x2a}}, BytesField{Value: []byte{0x1c, 0x17, 0x9c, 0x2c, 0x67, 0x95, 0x34, 0x78, 0x96, 0x6a, 0x6b, 0x46, 0xa, 0xb4, 0x87, 0x35, 0x85, 0xb2, 0xf3, 0x41}}}}},
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "Multiple operations",
			operations: getOperations(2),
			want: []*DecodedCall{
				{
					Address: aptosTestAddress,
					Method:  "ccip_onramp::onramp::initialize",
					Inputs: []NamedField{
						{Name: "chain_selector", Value: SimpleField{Value: "4457093679053095497"}},
						{Name: "fee_aggregator", Value: AddressField{Value: "0x13a9f1a109368730f2e355d831ba8fbf5942fb82321863d55de54cb4ebe5d18f"}},
						{Name: "allowlist_admin", Value: AddressField{Value: "0x13a9f1a109368730f2e355d831ba8fbf5942fb82321863d55de54cb4ebe5d18f"}},
						{Name: "dest_chain_selectors", Value: ArrayField{Elements: nil}},
						{Name: "dest_chain_routers", Value: ArrayField{Elements: nil}},
						{Name: "dest_chain_allowlist_enabled", Value: ArrayField{Elements: nil}},
					},
				},
				{
					Address: aptosTestAddress,
					Method:  "ccip_offramp::offramp::initialize",
					Inputs: []NamedField{
						{Name: "chain_selector", Value: SimpleField{Value: "4457093679053095497"}},
						{Name: "permissionless_execution_threshold_seconds", Value: SimpleField{Value: "28800"}},
						{Name: "source_chains_selector", Value: ArrayField{Elements: []FieldValue{SimpleField{Value: "11155111"}}}},
						{Name: "source_chains_is_enabled", Value: ArrayField{Elements: []FieldValue{SimpleField{Value: "true"}}}},
						{Name: "source_chains_is_rmn_verification_disabled", Value: ArrayField{Elements: []FieldValue{SimpleField{Value: "false"}}}},
						{Name: "source_chains_on_ramp", Value: ArrayField{Elements: []FieldValue{BytesField{Value: []byte{0xb, 0xf3, 0xde, 0x8c, 0x5d, 0x3e, 0x8a, 0x2b, 0x34, 0xd2, 0xbe, 0xeb, 0x17, 0xab, 0xfc, 0xeb, 0xaf, 0x36, 0x3a, 0x59}}}}},
					},
				},
				{
					Address: aptosTestAddress,
					Method:  "ccip::rmn_remote::initialize",
					Inputs: []NamedField{
						{Name: "local_chain_selector", Value: SimpleField{Value: "4457093679053095497"}},
					},
				},
				{
					Address: aptosTestAddress,
					Method:  "ccip_token_pool::token_pool::initialize",
					Inputs: []NamedField{
						{Name: "local_token", Value: AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000003"}},
						{Name: "allowlist", Value: ArrayField{Elements: []FieldValue{AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000001"}, AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000002"}}}},
					},
				},
				{
					Address: aptosTestAddress,
					Method:  "ccip_offramp::offramp::apply_source_chain_config_updates",
					Inputs: []NamedField{
						{Name: "source_chains_selector", Value: ArrayField{Elements: []FieldValue{SimpleField{Value: "743186221051783445"}, SimpleField{Value: "16015286601757825753"}}}},
						{Name: "source_chains_is_enabled", Value: ArrayField{Elements: []FieldValue{SimpleField{Value: "true"}, SimpleField{Value: "false"}}}},
						{Name: "source_chains_is_rmn_verification_disabled", Value: ArrayField{Elements: []FieldValue{SimpleField{Value: "true"}, SimpleField{Value: "true"}}}},
						{Name: "source_chains_on_ramp", Value: ArrayField{Elements: []FieldValue{BytesField{Value: []byte{0xc2, 0x30, 0x71, 0xa8, 0xae, 0x83, 0x67, 0x1f, 0x37, 0xbd, 0xa1, 0xda, 0xdb, 0xc7, 0x45, 0xa9, 0x78, 0xf, 0x63, 0x2a}}, BytesField{Value: []byte{0x1c, 0x17, 0x9c, 0x2c, 0x67, 0x95, 0x34, 0x78, 0x96, 0x6a, 0x6b, 0x46, 0xa, 0xb4, 0x87, 0x35, 0x85, 0xb2, 0xf3, 0x41}}}}},
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "Unknown module - non-blocking",
			operations: getBadOperations(),
			want: []*DecodedCall{
				{
					Address: aptosTestAddress,
					Method:  "failed to decode Aptos transaction: could not find function info for ccip_offramp::bad_module::initialize",
					Inputs:  []NamedField{},
				},
				{
					Address: aptosTestAddress,
					Method:  "ccip::rmn_remote::initialize",
					Inputs: []NamedField{
						{Name: "local_chain_selector", Value: SimpleField{Value: "4457093679053095497"}},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var allResults []*DecodedCall
			var hasError bool
			for _, operation := range tt.operations {
				results, err := AnalyzeAptosTransactions(defaultProposalCtx, chainSelector, operation.Transactions)
				if err != nil {
					// For unknown modules, we expect some transactions to fail but others to succeed
					hasError = true
					continue
				}
				allResults = append(allResults, results...)
			}

			if tt.wantErr {
				require.True(t, hasError, "AnalyzeAptosTransactions() should have failed")
				return
			}

			require.Len(t, allResults, len(tt.want), "Number of decoded calls should match")

			// Compare each DecodedCall
			for i, result := range allResults {
				expected := tt.want[i]
				require.Equal(t, expected.Address, result.Address, "Address mismatch for call %d", i)
				require.Equal(t, expected.Method, result.Method, "Method mismatch for call %d", i)
				require.Len(t, result.Inputs, len(expected.Inputs), "Number of inputs should match for call %d", i)

				// Compare each input
				for j, input := range result.Inputs {
					expectedInput := expected.Inputs[j]
					require.Equal(t, expectedInput.Name, input.Name, "Input name mismatch for call %d, input %d", i, j)
					require.Equal(t, expectedInput.Value.GetType(), input.Value.GetType(), "Input value type mismatch for call %d, input %d", i, j)

					switch expectedField := expectedInput.Value.(type) {
					case SimpleField:
						if actualField, ok := input.Value.(SimpleField); ok {
							require.Equal(t, expectedField.GetValue(), actualField.GetValue(), "SimpleField value mismatch for call %d, input %d", i, j)
						} else {
							t.Errorf("Expected SimpleField but got %T for call %d, input %d", input.Value, i, j)
						}
					default:
						require.Equal(t, expectedInput.Value, input.Value, "Field value mismatch for call %d, input %d", i, j)
					}
				}
			}
		})
	}
}

func getOperations(n int) []types.BatchOperation {
	// Mock operation values were got from Aptos changesets unit tests.
	mcmsTxs := []types.Transaction{
		{
			OperationMetadata: types.OperationMetadata{},
			To:                aptosTestAddress,
			Data: []byte{
				0x49, 0x42, 0x99, 0x1e, 0x16, 0xc7, 0xda, 0x3d, 0x13, 0xa9, 0xf1, 0xa1, 0x09, 0x36, 0x87, 0x30,
				0xf2, 0xe3, 0x55, 0xd8, 0x31, 0xba, 0x8f, 0xbf, 0x59, 0x42, 0xfb, 0x82, 0x32, 0x18, 0x63, 0xd5,
				0x5d, 0xe5, 0x4c, 0xb4, 0xeb, 0xe5, 0xd1, 0x8f, 0x13, 0xa9, 0xf1, 0xa1, 0x09, 0x36, 0x87, 0x30,
				0xf2, 0xe3, 0x55, 0xd8, 0x31, 0xba, 0x8f, 0xbf, 0x59, 0x42, 0xfb, 0x82, 0x32, 0x18, 0x63, 0xd5,
				0x5d, 0xe5, 0x4c, 0xb4, 0xeb, 0xe5, 0xd1, 0x8f, 0x00, 0x00, 0x00,
			},
			AdditionalFields: json.RawMessage(`{"package_name":"ccip_onramp","module_name":"onramp","function":"initialize"}`),
		},
		{
			OperationMetadata: types.OperationMetadata{},
			To:                aptosTestAddress,
			Data: []byte{
				0x49, 0x42, 0x99, 0x1e, 0x16, 0xc7, 0xda, 0x3d, 0x80, 0x70, 0x00, 0x00, 0x01, 0xa7, 0x36, 0xaa,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01, 0x00, 0x01, 0x14, 0x0b, 0xf3, 0xde, 0x8c, 0x5d,
				0x3e, 0x8a, 0x2b, 0x34, 0xd2, 0xbe, 0xeb, 0x17, 0xab, 0xfc, 0xeb, 0xaf, 0x36, 0x3a, 0x59,
			},
			AdditionalFields: json.RawMessage(`{"package_name":"ccip_offramp","module_name":"offramp","function":"initialize"}`),
		},
		{
			OperationMetadata: types.OperationMetadata{},
			To:                aptosTestAddress,
			Data:              []byte{0x49, 0x42, 0x99, 0x1e, 0x16, 0xc7, 0xda, 0x3d},
			AdditionalFields:  json.RawMessage(`{"package_name":"ccip","module_name":"rmn_remote","function":"initialize"}`),
		},
		{
			OperationMetadata: types.OperationMetadata{},
			To:                aptosTestAddress,
			Data: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03,
				0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x02,
			},
			AdditionalFields: json.RawMessage(`{"package_name":"ccip_token_pool","module_name":"token_pool","function":"initialize"}`),
		},
		{
			OperationMetadata: types.OperationMetadata{},
			To:                aptosTestAddress,
			Data: []byte{
				0x02, 0x15, 0xa9, 0xc1, 0x33, 0xee, 0x53, 0x50, 0x0a, 0xd9, 0x1a, 0xd9, 0xc9, 0x4f, 0xba, 0x41,
				0xde, 0x02, 0x01, 0x00, 0x02, 0x01, 0x01, 0x02, 0x14, 0xc2, 0x30, 0x71, 0xa8, 0xae, 0x83, 0x67,
				0x1f, 0x37, 0xbd, 0xa1, 0xda, 0xdb, 0xc7, 0x45, 0xa9, 0x78, 0x0f, 0x63, 0x2a, 0x14, 0x1c, 0x17,
				0x9c, 0x2c, 0x67, 0x95, 0x34, 0x78, 0x96, 0x6a, 0x6b, 0x46, 0x0a, 0xb4, 0x87, 0x35, 0x85, 0xb2,
				0xf3, 0x41,
			},
			AdditionalFields: json.RawMessage(`{"package_name":"ccip_offramp","module_name":"offramp","function":"apply_source_chain_config_updates"}`),
		},
	}
	switch n {
	case 1:
		return []types.BatchOperation{{
			ChainSelector: types.ChainSelector(chainsel.APTOS_TESTNET.Selector),
			Transactions:  mcmsTxs,
		}}
	case 2:
		return []types.BatchOperation{
			{
				ChainSelector: types.ChainSelector(chainsel.APTOS_TESTNET.Selector),
				Transactions:  mcmsTxs[:2],
			},
			{
				ChainSelector: types.ChainSelector(chainsel.APTOS_TESTNET.Selector),
				Transactions:  mcmsTxs[2:],
			},
		}
	default:
		return []types.BatchOperation{}
	}
}

func getBadOperations() []types.BatchOperation {
	// Mock operation with one bad module in it and one good module.
	mcmsTxs := []types.Transaction{
		{
			OperationMetadata: types.OperationMetadata{},
			To:                aptosTestAddress,
			Data: []byte{
				0x49, 0x42, 0x99, 0x1e, 0x16, 0xc7, 0xda, 0x3d, 0x80, 0x70, 0x00, 0x00, 0x01, 0xa7, 0x36, 0xaa,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01, 0x00, 0x01, 0x14, 0x0b, 0xf3, 0xde, 0x8c, 0x5d,
				0x3e, 0x8a, 0x2b, 0x34, 0xd2, 0xbe, 0xeb, 0x17, 0xab, 0xfc, 0xeb, 0xaf, 0x36, 0x3a, 0x59,
			},
			AdditionalFields: json.RawMessage(`{"package_name":"ccip_offramp","module_name":"bad_module","function":"initialize"}`),
		},
		{
			OperationMetadata: types.OperationMetadata{},
			To:                aptosTestAddress,
			Data:              []byte{0x49, 0x42, 0x99, 0x1e, 0x16, 0xc7, 0xda, 0x3d},
			AdditionalFields:  json.RawMessage(`{"package_name":"ccip","module_name":"rmn_remote","function":"initialize"}`),
		},
	}

	return []types.BatchOperation{{
		ChainSelector: types.ChainSelector(chainsel.APTOS_TESTNET.Selector),
		Transactions:  mcmsTxs,
	}}
}

// aptosCCIPV2FunctionInfo returns a lookup shaped like a caller passing chainlink-ccip-aptos
// bindings.GetFunctionInfo(...).String(): the JSON of the matching entries, "null" for none.
// The testdata file and the call data below come from chainlink-ccip-aptos bindings at 882eb7ec.
func aptosCCIPV2FunctionInfo(t *testing.T) AptosFunctionInfoLookup {
	t.Helper()

	raw, err := os.ReadFile("testdata/aptos_ccip_v2_function_info.json")
	require.NoError(t, err)
	infos, err := aptosbind.ParseFunctionInfo(string(raw))
	require.NoError(t, err)

	return func(packageName, moduleName, function string) string {
		var found aptosbind.FunctionInfos
		for _, info := range infos {
			if packageName == info.Package && moduleName == info.Module && function == info.Name {
				found = append(found, info)
			}
		}

		return found.String()
	}
}

func newAptosTestProposalContext(t *testing.T, typeAndVersion string, opts ...proposalCtxOption) ProposalContext {
	t.Helper()

	env := deployment.Environment{
		ExistingAddresses: deployment.NewMemoryAddressBook(),
		DataStore:         datastore.NewMemoryDataStore().Seal(),
	}
	proposalCtx, err := NewDefaultProposalContext(env, opts...)
	require.NoError(t, err)
	defaultCtx, ok := proposalCtx.(*DefaultProposalContext)
	require.True(t, ok)
	defaultCtx.AddressesByChain = deployment.AddressesByChain{
		chainsel.APTOS_TESTNET.Selector: {
			aptosTestAddress: deployment.MustTypeAndVersionFromString(typeAndVersion),
		},
	}

	return defaultCtx
}

func aptosTestTransaction(packageName, moduleName, function string, data []byte) types.Transaction {
	return types.Transaction{
		To:               aptosTestAddress,
		Data:             data,
		AdditionalFields: json.RawMessage(`{"package_name":"` + packageName + `","module_name":"` + moduleName + `","function":"` + function + `"}`),
	}
}

func TestAnalyzeAptosTransaction_WithAptosFunctionInfo(t *testing.T) {
	t.Parallel()

	proposalCtx := newAptosTestProposalContext(t, "TestCCIP 2.0.0", WithAptosFunctionInfo(aptosCCIPV2FunctionInfo(t)))

	tests := []struct {
		name string
		tx   types.Transaction
		want []NamedField
	}{
		{
			name: "committee_verifier::committee_verifier::set_allowed_finality_config",
			tx: aptosTestTransaction("committee_verifier", "committee_verifier", "set_allowed_finality_config", []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
				0x03, 0x00, 0x00, 0x00,
			}),
			want: []NamedField{
				{Name: "ccv_addr", Value: AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000001"}},
				{Name: "allowed_finality_bits", Value: SimpleField{Value: "3"}},
			},
		},
		{
			name: "versioned_ccv_resolver::versioned_ccv_resolver::apply_outbound_implementation_updates",
			tx: aptosTestTransaction("versioned_ccv_resolver", "versioned_ccv_resolver", "apply_outbound_implementation_updates", []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
				0x00, 0x01, 0xd9, 0x1a, 0xd9, 0xc9, 0x4f, 0xba, 0x41, 0xde, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
			}),
			want: []NamedField{
				{Name: "handle", Value: AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000001"}},
				{Name: "remove", Value: ArrayField{Elements: nil}},
				{Name: "add_dest_chain_selectors", Value: ArrayField{Elements: []FieldValue{SimpleField{Value: "16015286601757825753"}}}},
				{Name: "add_implementations", Value: ArrayField{Elements: []FieldValue{AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000002"}}}},
			},
		},
		{
			name: "ccip_executor::base_executor::set_dynamic_config",
			tx: aptosTestTransaction("ccip_executor", "base_executor", "set_dynamic_config", []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
				0x07, 0x00, 0x00, 0x00, 0x01,
			}),
			want: []NamedField{
				{Name: "executor_addr", Value: AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000001"}},
				{Name: "fee_aggregator", Value: AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000002"}},
				{Name: "allowed_finality_bits", Value: SimpleField{Value: "7"}},
				{Name: "ccv_allowlist_enabled", Value: SimpleField{Value: "true"}},
			},
		},
		{
			name: "ccip_onramp_2_0_0::onramp::initialize",
			tx: aptosTestTransaction("ccip_onramp_2_0_0", "onramp", "initialize", []byte{
				0x49, 0x42, 0x99, 0x1e, 0x16, 0xc7, 0xda, 0x3d, 0x10, 0x27, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
			}),
			want: []NamedField{
				{Name: "chain_selector", Value: SimpleField{Value: "4457093679053095497"}},
				{Name: "max_usd_cents_per_message", Value: SimpleField{Value: "10000"}},
				{Name: "fee_aggregator", Value: AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000002"}},
			},
		},
		{
			name: "ccip_offramp_2_0_0::offramp::apply_source_chain_config_updates",
			tx: aptosTestTransaction("ccip_offramp_2_0_0", "offramp", "apply_source_chain_config_updates", []byte{
				0xd9, 0x1a, 0xd9, 0xc9, 0x4f, 0xba, 0x41, 0xde, 0x01, 0x01, 0x02, 0xc2, 0x30, 0x01, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
			}),
			want: []NamedField{
				{Name: "source_chain_selector", Value: SimpleField{Value: "16015286601757825753"}},
				{Name: "is_enabled", Value: SimpleField{Value: "true"}},
				{Name: "on_ramps", Value: ArrayField{Elements: []FieldValue{BytesField{Value: []byte{0xc2, 0x30}}}}},
				{Name: "default_ccvs", Value: ArrayField{Elements: []FieldValue{AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000001"}}}},
				{Name: "lane_mandated_ccvs", Value: ArrayField{Elements: nil}},
			},
		},
		{
			name: "burn_mint_token_pool::burn_mint_token_pool_v2::add_remote_pool",
			tx: aptosTestTransaction("burn_mint_token_pool", "burn_mint_token_pool_v2", "add_remote_pool", []byte{
				0xd9, 0x1a, 0xd9, 0xc9, 0x4f, 0xba, 0x41, 0xde, 0x02, 0xab, 0xcd,
			}),
			want: []NamedField{
				{Name: "remote_chain_selector", Value: SimpleField{Value: "16015286601757825753"}},
				{Name: "remote_pool_address", Value: BytesField{Value: []byte{0xab, 0xcd}}},
			},
		},
		{
			name: "lock_release_token_pool::lock_release_token_pool_v2::initialize",
			tx: aptosTestTransaction("lock_release_token_pool", "lock_release_token_pool_v2", "initialize", []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04,
			}),
			want: []NamedField{
				{Name: "lock_box", Value: AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000004"}},
			},
		},
		{
			name: "managed_token_pool::managed_token_pool_v2::set_allowed_finality_config",
			tx: aptosTestTransaction("managed_token_pool", "managed_token_pool_v2", "set_allowed_finality_config", []byte{
				0x05, 0x00, 0x00, 0x00,
			}),
			want: []NamedField{
				{Name: "allowed_finality_bits", Value: SimpleField{Value: "5"}},
			},
		},
		{
			name: "regulated_token_pool::regulated_token_pool_v2::remove_remote_pool",
			tx: aptosTestTransaction("regulated_token_pool", "regulated_token_pool_v2", "remove_remote_pool", []byte{
				0xd9, 0x1a, 0xd9, 0xc9, 0x4f, 0xba, 0x41, 0xde, 0x02, 0xab, 0xcd,
			}),
			want: []NamedField{
				{Name: "remote_chain_selector", Value: SimpleField{Value: "16015286601757825753"}},
				{Name: "remote_pool_address", Value: BytesField{Value: []byte{0xab, 0xcd}}},
			},
		},
		{
			name: "ccip_token_pool::token_pool_v2::fee_account_address",
			tx: aptosTestTransaction("ccip_token_pool", "token_pool_v2", "fee_account_address", []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
			}),
			want: []NamedField{
				{Name: "pool", Value: AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000001"}},
			},
		},
		{
			name: "siloed_lock_release_token_pool::siloed_lock_release_token_pool::configure_lock_boxes",
			tx: aptosTestTransaction("siloed_lock_release_token_pool", "siloed_lock_release_token_pool", "configure_lock_boxes", []byte{
				0x01, 0xd9, 0x1a, 0xd9, 0xc9, 0x4f, 0xba, 0x41, 0xde, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04,
			}),
			want: []NamedField{
				{Name: "remote_chain_selectors", Value: ArrayField{Elements: []FieldValue{SimpleField{Value: "16015286601757825753"}}}},
				{Name: "lock_boxes", Value: ArrayField{Elements: []FieldValue{AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000004"}}}},
			},
		},
		{
			name: "lock_box::lock_box::apply_allowed_caller_updates",
			tx: aptosTestTransaction("lock_box", "lock_box", "apply_allowed_caller_updates", []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04,
				0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x05,
			}),
			want: []NamedField{
				{Name: "lock_box", Value: AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000004"}},
				{Name: "removes", Value: ArrayField{Elements: nil}},
				{Name: "adds", Value: ArrayField{Elements: []FieldValue{AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000005"}}}},
			},
		},
		{
			name: "ccip_token_pool::advanced_pool_hooks::set_threshold_amount",
			tx: aptosTestTransaction("ccip_token_pool", "advanced_pool_hooks", "set_threshold_amount", []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
				0x40, 0x42, 0x0f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			}),
			want: []NamedField{
				{Name: "config", Value: AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000001"}},
				{Name: "threshold_amount", Value: SimpleField{Value: "1000000"}},
			},
		},
		{
			name: "ccip::token_admin_registry_v2::set_pool",
			tx: aptosTestTransaction("ccip", "token_admin_registry_v2", "set_pool", []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
			}),
			want: []NamedField{
				{Name: "local_token", Value: AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000003"}},
				{Name: "token_pool_address", Value: AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000001"}},
			},
		},
		{
			name: "ccip::fee_quoter::apply_dest_chain_config_v2_updates",
			tx: aptosTestTransaction("ccip", "fee_quoter", "apply_dest_chain_config_v2_updates", []byte{
				0xd9, 0x1a, 0xd9, 0xc9, 0x4f, 0xba, 0x41, 0xde, 0x19, 0x00, 0x00, 0x00, 0x90, 0x5f, 0x01, 0x00,
				0x01,
			}),
			want: []NamedField{
				{Name: "dest_chain_selector", Value: SimpleField{Value: "16015286601757825753"}},
				{Name: "default_token_fee_usd_cents", Value: SimpleField{Value: "25"}},
				{Name: "default_token_dest_gas_overhead", Value: SimpleField{Value: "90000"}},
				{Name: "is_enabled", Value: SimpleField{Value: "true"}},
			},
		},
		{
			name: "mcms::mcms::execute uses registered info before chainlink-aptos",
			tx: aptosTestTransaction("mcms", "mcms", "execute", []byte{
				0x01, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x01, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x0a, 0x66, 0x65, 0x65, 0x5f, 0x71, 0x75,
				0x6f, 0x74, 0x65, 0x72, 0x13, 0x61, 0x70, 0x70, 0x6c, 0x79, 0x5f, 0x70, 0x72, 0x69, 0x63, 0x65,
				0x5f, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x73, 0x01, 0x01, 0x01, 0x01, 0xaa,
			}),
			want: []NamedField{
				{Name: "role", Value: SimpleField{Value: "1"}},
				{Name: "chain_id", Value: SimpleField{Value: "2"}},
				{Name: "multisig_addr", Value: AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000001"}},
				{Name: "nonce", Value: SimpleField{Value: "3"}},
				{Name: "to", Value: AddressField{Value: "0x0000000000000000000000000000000000000000000000000000000000000002"}},
				{Name: "module_name", Value: SimpleField{Value: "fee_quoter"}},
				{Name: "function_name", Value: SimpleField{Value: "apply_price_updates"}},
				{Name: "data", Value: BytesField{Value: []byte{0x1}}},
				{Name: "merkle_proof", Value: ArrayField{Elements: []FieldValue{BytesField{Value: []byte{0xaa}}}}},
			},
		},
		{
			name: "1.6 call falls back to chainlink-aptos bindings",
			tx:   getOperations(1)[0].Transactions[2],
			want: []NamedField{
				{Name: "local_chain_selector", Value: SimpleField{Value: "4457093679053095497"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := AnalyzeAptosTransaction(proposalCtx, mcmsaptossdk.NewDecoder(), chainsel.APTOS_TESTNET.Selector, tt.tx)
			require.NoError(t, err)

			var fields mcmsaptossdk.AdditionalFields
			require.NoError(t, json.Unmarshal(tt.tx.AdditionalFields, &fields))
			require.Equal(t, fields.PackageName+"::"+fields.ModuleName+"::"+fields.Function, got.Method)
			require.Equal(t, aptosTestAddress, got.Address)
			require.Equal(t, "TestCCIP", got.ContractType)
			require.Equal(t, "2.0.0", got.ContractVersion)
			require.Len(t, got.Inputs, len(tt.want))
			for i, input := range got.Inputs {
				require.Equal(t, tt.want[i].Name, input.Name)
				require.Equal(t, tt.want[i].Value, input.Value, "input %s", input.Name)
			}
		})
	}
}

func TestAnalyzeAptosTransaction_AptosFunctionInfoFallback(t *testing.T) {
	t.Parallel()

	invalidJSON := WithAptosFunctionInfo(func(string, string, string) string { return "{" })
	wrongShape := WithAptosFunctionInfo(func(string, string, string) string { return `[{"Foo":"bar"}]` })
	v16Tx := getOperations(1)[0].Transactions[2]
	unknownTx := aptosTestTransaction("committee_verifier", "committee_verifier", "unknown_function", []byte{0x01})
	v2Tx := aptosTestTransaction("managed_token_pool", "managed_token_pool_v2", "set_allowed_finality_config", []byte{0x05, 0x00, 0x00, 0x00})

	tests := []struct {
		name       string
		opts       []proposalCtxOption
		tx         types.Transaction
		wantMethod string
	}{
		{
			name:       "no extra lookup renders failed method",
			tx:         unknownTx,
			wantMethod: "failed to decode Aptos transaction: could not find function info for committee_verifier::committee_verifier::unknown_function",
		},
		{
			name:       "extra lookup without a match renders failed method",
			opts:       []proposalCtxOption{WithAptosFunctionInfo(aptosCCIPV2FunctionInfo(t))},
			tx:         unknownTx,
			wantMethod: "failed to decode Aptos transaction: could not find function info for committee_verifier::committee_verifier::unknown_function",
		},
		{
			name:       "invalid JSON falls back to chainlink-aptos",
			opts:       []proposalCtxOption{invalidJSON},
			tx:         v16Tx,
			wantMethod: "ccip::rmn_remote::initialize",
		},
		{
			name:       "invalid JSON without fallback renders failed method",
			opts:       []proposalCtxOption{invalidJSON},
			tx:         unknownTx,
			wantMethod: "failed to decode Aptos transaction: aptos function info lookup for committee_verifier::committee_verifier::unknown_function: unexpected end of JSON input",
		},
		{
			name:       "wrong shape falls back to chainlink-aptos",
			opts:       []proposalCtxOption{wrongShape},
			tx:         v16Tx,
			wantMethod: "ccip::rmn_remote::initialize",
		},
		{
			name:       "wrong shape without fallback renders failed method",
			opts:       []proposalCtxOption{wrongShape},
			tx:         unknownTx,
			wantMethod: "failed to decode Aptos transaction: aptos function info lookup returned entries that do not name committee_verifier::committee_verifier::unknown_function",
		},
		{
			name:       "error falls through to the next lookup",
			opts:       []proposalCtxOption{wrongShape, WithAptosFunctionInfo(aptosCCIPV2FunctionInfo(t))},
			tx:         v2Tx,
			wantMethod: "managed_token_pool::managed_token_pool_v2::set_allowed_finality_config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			proposalCtx := newAptosTestProposalContext(t, "TestCCIP 2.0.0", tt.opts...)
			got, err := AnalyzeAptosTransaction(proposalCtx, mcmsaptossdk.NewDecoder(), chainsel.APTOS_TESTNET.Selector, tt.tx)
			require.NoError(t, err)
			require.Equal(t, aptosTestAddress, got.Address)
			require.Equal(t, tt.wantMethod, got.Method)
			require.Equal(t, "TestCCIP", got.ContractType)
			require.Equal(t, "2.0.0", got.ContractVersion)
		})
	}
}
