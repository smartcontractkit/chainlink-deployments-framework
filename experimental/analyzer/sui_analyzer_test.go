package analyzer

import (
	"encoding/json"
	"reflect"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

const suiTestAddress = "0x4e825a4758064df713762e431c3a16b8105857195214469db0d6985b7d70266d"
const suiAddressTitle = "address of MCMSUser 1.0.0 from sui-testnet"

func TestAnalyzeSuiTransactions(t *testing.T) {
	t.Parallel()

	defaultProposalCtx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{
			chainsel.SUI_TESTNET.Selector: {
				suiTestAddress: deployment.MustTypeAndVersionFromString("MCMSUser 1.0.0"),
			},
		},
	}

	tests := []struct {
		name       string
		operations []types.BatchOperation
		want       [][]string
		wantErr    bool
	}{
		{
			name: "analyze Sui transactions",
			operations: []types.BatchOperation{
				{
					ChainSelector: types.ChainSelector(chainsel.SUI_TESTNET.Selector),
					Transactions:  getMcmsTxs(suiTestAddress),
				},
			},
			want: [][]string{{
				expectedOutput("mcms_user::function_one", suiTestAddress, suiAddressTitle, [][]string{
					{"user_data", "0x8bc59c2842f436c1221691a359dc42941c1f25eca13f4bad79f7b00e8df4b968"},
					{"owner_cap", "0x5b97db59e5e5d7d2d5e0421173aaee6511dbb494bd23ba98d463591c5e8e4887"},
					{"arg1", "Updated Field A"},
					{"arg2", "0x0102030405060708090a"},
				}),
				expectedOutput("mcms_user::function_two", suiTestAddress, suiAddressTitle, [][]string{
					{"user_data", "0x8bc59c2842f436c1221691a359dc42941c1f25eca13f4bad79f7b00e8df4b968"},
					{"owner_cap", "0x5b97db59e5e5d7d2d5e0421173aaee6511dbb494bd23ba98d463591c5e8e4887"},
					{"arg1", "0x5b97db59e5e5d7d2d5e0421173aaee6511dbb494bd23ba98d463591c5e8e4887"},
					{"arg2", "2048"},
				}),
			}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, gotErr := describeBatchOperations(defaultProposalCtx, tt.operations)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("AnalyzeSuiTransactions() failed: %v", gotErr)
				}

				return
			}
			if tt.wantErr {
				t.Fatal("AnalyzeSuiTransactions() succeeded unexpectedly")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AnalyzeSuiTransactions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func getMcmsTxs(suiTestAddress string) []types.Transaction {
	return []types.Transaction{
		{
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
		{
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
	}
}
