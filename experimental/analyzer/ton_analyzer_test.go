package analyzer

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/lib/access/rbac"
	"github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

const tonTestAddress = "EQDtFpEwcFAEcRe5mLVh2N6C0x-_hJEM7W61_JLnSF74p4q2"

// testTONSetup contains common test fixtures for TON analyzer tests.
type testTONSetup struct {
	targetAddr     *address.Address
	exampleRoleBig *big.Int
}

func newTestTONSetup(t *testing.T) *testTONSetup {
	t.Helper()

	exampleRole := crypto.Keccak256Hash([]byte("EXAMPLE_ROLE"))
	exampleRoleBig, _ := cell.BeginCell().
		MustStoreBigInt(new(big.Int).SetBytes(exampleRole[:]), 257).
		EndCell().
		ToBuilder().
		ToSlice().
		LoadBigInt(256)

	return &testTONSetup{
		targetAddr:     address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"),
		exampleRoleBig: exampleRoleBig,
	}
}

func (s *testTONSetup) makeGrantRoleTx(t *testing.T, queryID uint64) types.Transaction {
	t.Helper()

	grantRoleData, err := tlb.ToCell(rbac.GrantRole{
		QueryID: queryID,
		Role:    s.exampleRoleBig,
		Account: s.targetAddr,
	})
	require.NoError(t, err)

	tx, err := ton.NewTransaction(
		s.targetAddr,
		grantRoleData.ToBuilder().ToSlice(),
		big.NewInt(0),
		"com.chainlink.ton.lib.access.RBAC",
		[]string{"grantRole"},
	)
	require.NoError(t, err)

	return tx
}

func (s *testTONSetup) expectedGrantRoleCall(queryID uint64) *DecodedCall {
	return &DecodedCall{
		Address: s.targetAddr.String(),
		Method:  "com.chainlink.ton.lib.access.RBAC::GrantRole(0x0)",
		Inputs: []NamedField{
			{Name: "QueryID", Value: SimpleField{Value: bigIntStr(queryID)}},
			{Name: "Role", Value: SimpleField{Value: s.exampleRoleBig.String()}},
			{Name: "Account", Value: SimpleField{Value: s.targetAddr.String()}},
		},
	}
}

func bigIntStr(v uint64) string {
	return new(big.Int).SetUint64(v).String()
}

func makeInvalidTx(contractType string) types.Transaction {
	return types.Transaction{
		OperationMetadata: types.OperationMetadata{ContractType: contractType},
		To:                tonTestAddress,
		Data:              []byte{0xFF, 0xFF},
		AdditionalFields:  json.RawMessage(`{"value":0}`),
	}
}

func TestAnalyzeTONTransaction(t *testing.T) {
	t.Parallel()

	setup := newTestTONSetup(t)
	ctx := &DefaultProposalContext{}

	tests := []struct {
		name           string
		mcmsTx         types.Transaction
		want           *DecodedCall
		wantErrContain string
	}{
		{
			name:   "success - RBAC GrantRole",
			mcmsTx: setup.makeGrantRoleTx(t, 1),
			want:   setup.expectedGrantRoleCall(1),
		},
		{
			name:           "invalid data",
			mcmsTx:         makeInvalidTx("com.chainlink.ton.mcms.MCMS"),
			want:           &DecodedCall{Address: tonTestAddress},
			wantErrContain: "invalid cell BOC data",
		},
		{
			name: "unknown contract type",
			mcmsTx: types.Transaction{
				OperationMetadata: types.OperationMetadata{ContractType: "unknown.type"},
				To:                tonTestAddress,
				Data:              []byte{0x01, 0x02},
				AdditionalFields:  json.RawMessage(`{"value":0}`),
			},
			want:           &DecodedCall{Address: tonTestAddress},
			wantErrContain: "unknown contract interface: unknown.type",
		},
		{
			name: "empty data",
			mcmsTx: types.Transaction{
				OperationMetadata: types.OperationMetadata{ContractType: "com.chainlink.ton.mcms.MCMS"},
				To:                tonTestAddress,
				Data:              []byte{},
				AdditionalFields:  json.RawMessage(`{"value":0}`),
			},
			want:           &DecodedCall{Address: tonTestAddress},
			wantErrContain: "invalid cell BOC data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := AnalyzeTONTransaction(ctx, tt.mcmsTx)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, tt.want.Address, result.Address)

			if tt.wantErrContain != "" {
				require.Contains(t, result.Method, tt.wantErrContain)
				require.Nil(t, result.Inputs)

				return
			}

			assertDecodedCallEqual(t, tt.want, result)
		})
	}
}

func TestAnalyzeTONTransactions(t *testing.T) {
	t.Parallel()

	setup := newTestTONSetup(t)
	ctx := &DefaultProposalContext{}

	tests := []struct {
		name            string
		txs             []types.Transaction
		want            []*DecodedCall
		wantErrContains []string
	}{
		{
			name: "multiple valid transactions",
			txs: []types.Transaction{
				setup.makeGrantRoleTx(t, 1),
				setup.makeGrantRoleTx(t, 2),
				setup.makeGrantRoleTx(t, 3),
			},
			want: []*DecodedCall{
				setup.expectedGrantRoleCall(1),
				setup.expectedGrantRoleCall(2),
				setup.expectedGrantRoleCall(3),
			},
		},
		{
			name: "mixed valid and invalid",
			txs: []types.Transaction{
				makeInvalidTx("com.chainlink.ton.mcms.MCMS"),
				setup.makeGrantRoleTx(t, 1),
				makeInvalidTx("com.chainlink.ton.mcms.Timelock"),
			},
			want: []*DecodedCall{
				{Address: tonTestAddress},
				setup.expectedGrantRoleCall(1),
				{Address: tonTestAddress},
			},
			wantErrContains: []string{"invalid cell BOC data", "", "invalid cell BOC data"},
		},
		{
			name: "all decode failures",
			txs: []types.Transaction{
				makeInvalidTx("com.chainlink.ton.mcms.MCMS"),
				makeInvalidTx("com.chainlink.ton.mcms.Timelock"),
			},
			want: []*DecodedCall{
				{Address: tonTestAddress},
				{Address: tonTestAddress},
			},
			wantErrContains: []string{"invalid cell BOC data", "invalid cell BOC data"},
		},
		{
			name: "empty list",
			txs:  []types.Transaction{},
			want: []*DecodedCall{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			results, err := AnalyzeTONTransactions(ctx, tt.txs)
			require.NoError(t, err)
			require.Len(t, results, len(tt.want))

			for i, result := range results {
				require.Equal(t, tt.want[i].Address, result.Address, "call %d", i)

				if len(tt.wantErrContains) > i && tt.wantErrContains[i] != "" {
					require.Contains(t, result.Method, tt.wantErrContains[i], "call %d", i)
					require.Nil(t, result.Inputs, "call %d", i)

					continue
				}

				assertDecodedCallEqual(t, tt.want[i], result)
			}
		})
	}
}

func TestAnalyzeTONTransactions_BatchOperations(t *testing.T) {
	t.Parallel()

	setup := newTestTONSetup(t)
	ctx := &DefaultProposalContext{}
	chainSelector := chainsel.TON_TESTNET.Selector

	batchOps := []types.BatchOperation{
		{
			ChainSelector: types.ChainSelector(chainSelector),
			Transactions: []types.Transaction{
				setup.makeGrantRoleTx(t, 1),
				setup.makeGrantRoleTx(t, 2),
			},
		},
		{
			ChainSelector: types.ChainSelector(chainSelector),
			Transactions: []types.Transaction{
				setup.makeGrantRoleTx(t, 3),
				makeInvalidTx("com.chainlink.ton.lib.access.RBAC"),
			},
		},
	}

	want := []*DecodedCall{
		setup.expectedGrantRoleCall(1),
		setup.expectedGrantRoleCall(2),
		setup.expectedGrantRoleCall(3),
		{Address: tonTestAddress},
	}
	wantErrContains := []string{"", "", "", "invalid cell BOC data"}

	var allResults []*DecodedCall
	for _, batch := range batchOps {
		results, err := AnalyzeTONTransactions(ctx, batch.Transactions)
		require.NoError(t, err)
		allResults = append(allResults, results...)
	}

	require.Len(t, allResults, len(want))

	for i, result := range allResults {
		require.Equal(t, want[i].Address, result.Address, "call %d", i)

		if wantErrContains[i] != "" {
			require.Contains(t, result.Method, wantErrContains[i], "call %d", i)
			require.Nil(t, result.Inputs, "call %d", i)
		} else {
			assertDecodedCallEqual(t, want[i], result)
		}
	}
}

// assertDecodedCallEqual compares two DecodedCall structs for equality.
func assertDecodedCallEqual(t *testing.T, expected, actual *DecodedCall) {
	t.Helper()

	require.Equal(t, expected.Method, actual.Method)
	require.Len(t, actual.Inputs, len(expected.Inputs))

	for j, input := range actual.Inputs {
		expectedInput := expected.Inputs[j]
		require.Equal(t, expectedInput.Name, input.Name, "input %d", j)

		expectedField, ok := expectedInput.Value.(SimpleField)
		require.True(t, ok, "expected SimpleField for input %d", j)

		actualField, ok := input.Value.(SimpleField)
		require.True(t, ok, "expected SimpleField but got %T for input %d", input.Value, j)

		require.Equal(t, expectedField.GetValue(), actualField.GetValue(), "input %d", j)
	}
}
