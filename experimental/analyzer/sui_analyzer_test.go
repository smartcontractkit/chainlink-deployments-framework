package analyzer

import (
	"encoding/json"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

const suiTestAddress = "0x4e825a4758064df713762e431c3a16b8105857195214469db0d6985b7d70266d"

func TestAnalyzeSuiTransactions(t *testing.T) {
	t.Parallel()

	defaultProposalCtx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{
			chainsel.SUI_TESTNET.Selector: {
				suiTestAddress: deployment.MustTypeAndVersionFromString("MCMSUser 1.0.0"),
			},
		},
	}

	chainSelector := chainsel.SUI_TESTNET.Selector

	tests := []struct {
		name    string
		mcmsTx  types.Transaction
		want    *DecodedCall
		wantErr bool
	}{
		{
			name: "analyze Sui function_one transaction",
			mcmsTx: types.Transaction{
				OperationMetadata: types.OperationMetadata{},
				To:                suiTestAddress,
				Data: []byte{
					0x8b, 0xc5, 0x9c, 0x28, 0x42, 0xf4, 0x36, 0xc1, 0x22, 0x16, 0x91, 0xa3, 0x59, 0xdc, 0x42, 0x94, 0x1c,
					0x1f, 0x25, 0xec, 0xa1, 0x3f, 0x4b, 0xad, 0x79, 0xf7, 0xb0, 0xe, 0x8d, 0xf4, 0xb9, 0x68, 0x5b, 0x97,
					0xdb, 0x59, 0xe5, 0xe5, 0xd7, 0xd2, 0xd5, 0xe0, 0x42, 0x11, 0x73, 0xaa, 0xee, 0x65, 0x11, 0xdb, 0xb4,
					0x94, 0xbd, 0x23, 0xba, 0x98, 0xd4, 0x63, 0x59, 0x1c, 0x5e, 0x8e, 0x48, 0x87, 0xf, 0x55, 0x70, 0x64,
					0x61, 0x74, 0x65, 0x64, 0x20, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x20, 0x41, 0xa, 0x1, 0x2, 0x3, 0x4, 0x5,
					0x6, 0x7, 0x8, 0x9, 0xa,
				},
				AdditionalFields: json.RawMessage(`{"module_name":"mcms_user","function":"function_one","state_obj":"0x8bc59c2842f436c1221691a359dc42941c1f25eca13f4bad79f7b00e8df4b968"}`),
			},
			want: &DecodedCall{
				Address: suiTestAddress,
				Method:  "mcms_user::function_one",
				Inputs: []NamedField{
					{
						Name:  "user_data",
						Value: AddressField{Value: "0x8bc59c2842f436c1221691a359dc42941c1f25eca13f4bad79f7b00e8df4b968"},
					},
					{
						Name:  "owner_cap",
						Value: AddressField{Value: "0x5b97db59e5e5d7d2d5e0421173aaee6511dbb494bd23ba98d463591c5e8e4887"},
					},
					{
						Name:  "arg1",
						Value: SimpleField{Value: "Updated Field A"},
					},
					{
						Name:  "arg2",
						Value: BytesField{Value: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "analyze Sui function_two transaction",
			mcmsTx: types.Transaction{
				OperationMetadata: types.OperationMetadata{},
				To:                suiTestAddress,
				Data: []byte{
					0x8b, 0xc5, 0x9c, 0x28, 0x42, 0xf4, 0x36, 0xc1, 0x22, 0x16, 0x91, 0xa3, 0x59, 0xdc, 0x42, 0x94, 0x1c,
					0x1f, 0x25, 0xec, 0xa1, 0x3f, 0x4b, 0xad, 0x79, 0xf7, 0xb0, 0xe, 0x8d, 0xf4, 0xb9, 0x68, 0x5b, 0x97,
					0xdb, 0x59, 0xe5, 0xe5, 0xd7, 0xd2, 0xd5, 0xe0, 0x42, 0x11, 0x73, 0xaa, 0xee, 0x65, 0x11, 0xdb, 0xb4,
					0x94, 0xbd, 0x23, 0xba, 0x98, 0xd4, 0x63, 0x59, 0x1c, 0x5e, 0x8e, 0x48, 0x87, 0x5b, 0x97, 0xdb, 0x59,
					0xe5, 0xe5, 0xd7, 0xd2, 0xd5, 0xe0, 0x42, 0x11, 0x73, 0xaa, 0xee, 0x65, 0x11, 0xdb, 0xb4, 0x94, 0xbd,
					0x23, 0xba, 0x98, 0xd4, 0x63, 0x59, 0x1c, 0x5e, 0x8e, 0x48, 0x87, 0x0, 0x8, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
					0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				},
				AdditionalFields: json.RawMessage(`{"module_name":"mcms_user","function":"function_two","state_obj":"0x8bc59c2842f436c1221691a359dc42941c1f25eca13f4bad79f7b00e8df4b968"}`),
			},
			want: &DecodedCall{
				Address: suiTestAddress,
				Method:  "mcms_user::function_two",
				Inputs: []NamedField{
					{
						Name:  "user_data",
						Value: AddressField{Value: "0x8bc59c2842f436c1221691a359dc42941c1f25eca13f4bad79f7b00e8df4b968"},
					},
					{
						Name:  "owner_cap",
						Value: AddressField{Value: "0x5b97db59e5e5d7d2d5e0421173aaee6511dbb494bd23ba98d463591c5e8e4887"},
					},
					{
						Name:  "arg1",
						Value: AddressField{Value: "0x5b97db59e5e5d7d2d5e0421173aaee6511dbb494bd23ba98d463591c5e8e4887"},
					},
					{
						Name:  "arg2",
						Value: SimpleField{Value: "2048"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := AnalyzeSuiTransaction(defaultProposalCtx, chainSelector, tt.mcmsTx)

			if tt.wantErr {
				require.Error(t, err, "AnalyzeSuiTransaction() should have failed")
				return
			}

			require.NoError(t, err, "AnalyzeSuiTransaction() should not have failed")
			require.NotNil(t, result, "Result should not be nil")

			// Compare the DecodedCall directly
			require.Equal(t, tt.want.Address, result.Address, "Address mismatch")
			require.Equal(t, tt.want.Method, result.Method, "Method mismatch")
			require.Len(t, result.Inputs, len(tt.want.Inputs), "Number of inputs should match")

			// Compare each input
			for i, input := range result.Inputs {
				expectedInput := tt.want.Inputs[i]
				require.Equal(t, expectedInput.Name, input.Name, "Input name mismatch for input %d", i)
				require.Equal(t, expectedInput.Value.GetType(), input.Value.GetType(), "Input value type mismatch for input %d", i)

				// Compare field values based on type
				switch expectedField := expectedInput.Value.(type) {
				case SimpleField:
					if actualField, ok := input.Value.(SimpleField); ok {
						require.Equal(t, expectedField.GetValue(), actualField.GetValue(), "SimpleField value mismatch for input %d", i)
					} else {
						t.Errorf("Expected SimpleField but got %T for input %d", input.Value, i)
					}
				case AddressField:
					if actualField, ok := input.Value.(AddressField); ok {
						require.Equal(t, expectedField.GetValue(), actualField.GetValue(), "AddressField value mismatch for input %d", i)
					} else {
						t.Errorf("Expected AddressField but got %T for input %d", input.Value, i)
					}
				default:
					require.Equal(t, expectedInput.Value, input.Value, "Field value mismatch for input %d", i)
				}
			}
		})
	}
}

func TestAnalyzeSuiTransactionWithErrors(t *testing.T) {
	t.Parallel()

	defaultProposalCtx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{
			chainsel.SUI_TESTNET.Selector: {
				suiTestAddress: deployment.MustTypeAndVersionFromString("MCMSUser 1.0.0"),
			},
		},
	}

	chainSelector := chainsel.SUI_TESTNET.Selector

	tests := []struct {
		name        string
		mcmsTx      types.Transaction
		wantAddress string
		wantMethod  string
		wantError   bool
		errorSubstr string
	}{
		{
			name: "invalid JSON in AdditionalFields",
			mcmsTx: types.Transaction{
				To:               suiTestAddress,
				Data:             []byte("some data"),
				AdditionalFields: json.RawMessage(`invalid json`),
			},
			wantError:   true,
			errorSubstr: "failed to unmarshal Sui additional fields",
		},
		{
			name: "unknown module name",
			mcmsTx: types.Transaction{
				To:               suiTestAddress,
				Data:             []byte("some data"),
				AdditionalFields: json.RawMessage(`{"module_name":"unknown_module","function":"some_function","state_obj":"0x123"}`),
			},
			wantAddress: suiTestAddress,
			wantMethod:  "no function info found for module unknown_module on chain selector",
			wantError:   false,
		},
		{
			name: "decoder decode failure with empty data",
			mcmsTx: types.Transaction{
				To:               suiTestAddress,
				Data:             []byte{}, // Empty data will likely cause decode failure
				AdditionalFields: json.RawMessage(`{"module_name":"mcms_user","function":"function_one","state_obj":"0x8bc59c2842f436c1221691a359dc42941c1f25eca13f4bad79f7b00e8df4b968"}`),
			},
			wantAddress: suiTestAddress,
			wantMethod:  "failed to decode Sui transaction:",
			wantError:   false,
		},
		{
			name: "decoder decode failure with invalid data",
			mcmsTx: types.Transaction{
				To:               suiTestAddress,
				Data:             []byte{0xFF, 0xFF, 0xFF, 0xFF}, // Invalid/corrupted data
				AdditionalFields: json.RawMessage(`{"module_name":"mcms_user","function":"function_one","state_obj":"0x8bc59c2842f436c1221691a359dc42941c1f25eca13f4bad79f7b00e8df4b968"}`),
			},
			wantAddress: suiTestAddress,
			wantMethod:  "failed to decode Sui transaction:",
			wantError:   false,
		},
		{
			name: "successful decode",
			mcmsTx: types.Transaction{
				To: suiTestAddress,
				Data: []byte{
					0x8b, 0xc5, 0x9c, 0x28, 0x42, 0xf4, 0x36, 0xc1, 0x22, 0x16, 0x91, 0xa3, 0x59, 0xdc, 0x42, 0x94, 0x1c,
					0x1f, 0x25, 0xec, 0xa1, 0x3f, 0x4b, 0xad, 0x79, 0xf7, 0xb0, 0xe, 0x8d, 0xf4, 0xb9, 0x68, 0x5b, 0x97,
					0xdb, 0x59, 0xe5, 0xe5, 0xd7, 0xd2, 0xd5, 0xe0, 0x42, 0x11, 0x73, 0xaa, 0xee, 0x65, 0x11, 0xdb, 0xb4,
					0x94, 0xbd, 0x23, 0xba, 0x98, 0xd4, 0x63, 0x59, 0x1c, 0x5e, 0x8e, 0x48, 0x87, 0xf, 0x55, 0x70, 0x64,
					0x61, 0x74, 0x65, 0x64, 0x20, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x20, 0x41, 0xa, 0x1, 0x2, 0x3, 0x4, 0x5,
					0x6, 0x7, 0x8, 0x9, 0xa,
				},
				AdditionalFields: json.RawMessage(`{"module_name":"mcms_user","function":"function_one","state_obj":"0x8bc59c2842f436c1221691a359dc42941c1f25eca13f4bad79f7b00e8df4b968"}`),
			},
			wantAddress: suiTestAddress,
			wantMethod:  "mcms_user::function_one",
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := AnalyzeSuiTransaction(defaultProposalCtx, chainSelector, tt.mcmsTx)

			if tt.wantError {
				require.Error(t, err, "AnalyzeSuiTransaction() should have failed")
				if tt.errorSubstr != "" {
					require.Contains(t, err.Error(), tt.errorSubstr, "Error should contain expected substring")
				}

				return
			}

			require.NoError(t, err, "AnalyzeSuiTransaction() should not have failed")
			require.NotNil(t, result, "Result should not be nil")
			require.Equal(t, tt.wantAddress, result.Address, "Address mismatch")

			if tt.wantMethod != "" {
				switch tt.name {
				case "unknown module name":
					// For unknown module, just check that it contains the expected substring
					require.Contains(t, result.Method, tt.wantMethod, "Method should contain expected substring")
				case "decoder decode failure with empty data", "decoder decode failure with invalid data":
					// For decode failure, just check that it starts with the expected prefix
					require.True(t, hasPrefix(result.Method, tt.wantMethod),
						"Method %q should start with prefix %q", result.Method, tt.wantMethod)
				default:
					require.Equal(t, tt.wantMethod, result.Method, "Method mismatch")
				}
			}
		})
	}
}

// Helper function for string prefix checking
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
