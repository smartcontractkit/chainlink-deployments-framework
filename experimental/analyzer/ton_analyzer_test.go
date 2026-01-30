package analyzer

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/lib/access/rbac"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tlbe"
	"github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
)

const testAddress = "EQDtFpEwcFAEcRe5mLVh2N6C0x-_hJEM7W61_JLnSF74p4q2"

func TestAnalyzeTONTransaction(t *testing.T) {
	t.Parallel()

	setup := newTestTONSetup(t)
	ctx := &DefaultProposalContext{}
	decoder := ton.NewDecoder(bindings.Registry)

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
			want:           &DecodedCall{Address: testAddress},
			wantErrContain: "invalid cell BOC data",
		},
		{
			name: "unknown contract type",
			mcmsTx: types.Transaction{
				OperationMetadata: types.OperationMetadata{ContractType: "unknown.type"},
				To:                testAddress,
				Data:              []byte{0x01, 0x02},
				AdditionalFields:  json.RawMessage(`{"value":0}`),
			},
			want:           &DecodedCall{Address: testAddress},
			wantErrContain: "unknown contract interface: unknown.type",
		},
		{
			name: "empty data",
			mcmsTx: types.Transaction{
				OperationMetadata: types.OperationMetadata{ContractType: "com.chainlink.ton.mcms.MCMS"},
				To:                testAddress,
				Data:              []byte{},
				AdditionalFields:  json.RawMessage(`{"value":0}`),
			},
			want:           &DecodedCall{Address: testAddress},
			wantErrContain: "invalid cell BOC data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := AnalyzeTONTransaction(ctx, decoder, tt.mcmsTx)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, tt.want.Address, result.Address)

			if tt.wantErrContain != "" {
				require.Contains(t, result.Method, tt.wantErrContain)
				require.Nil(t, result.Inputs)

				return
			}

			require.Equal(t, tt.want, result)
		})
	}
}

// testTONSetup contains common test fixtures for TON analyzer tests.
type testTONSetup struct {
	targetAddr     *address.Address
	exampleRoleBig *big.Int
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
				{Address: testAddress},
				setup.expectedGrantRoleCall(1),
				{Address: testAddress},
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
				{Address: testAddress},
				{Address: testAddress},
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

				require.Equal(t, tt.want[i], result)
			}
		})
	}
}

func newTestTONSetup(t *testing.T) *testTONSetup {
	t.Helper()

	exampleRole := crypto.Keccak256Hash([]byte("EXAMPLE_ROLE"))
	exampleRoleBig := tlbe.NewUint256(new(big.Int).SetBytes(exampleRole[:]))

	return &testTONSetup{
		targetAddr:     address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"),
		exampleRoleBig: exampleRoleBig.Value(),
	}
}

func (s *testTONSetup) makeGrantRoleTx(t *testing.T, queryID uint64) types.Transaction {
	t.Helper()

	grantRoleData, err := tlb.ToCell(rbac.GrantRole{
		QueryID: queryID,
		Role:    tlbe.NewUint256(s.exampleRoleBig),
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
		Method:  "com.chainlink.ton.lib.access.RBAC::GrantRole(0x95cd540f)",
		Inputs: []NamedField{
			{Name: "QueryID", Value: SimpleField{Value: bigIntStr(queryID)}},
			{Name: "Role", Value: SimpleField{Value: s.exampleRoleBig.String()}},
			{Name: "Account", Value: SimpleField{Value: s.targetAddr.String()}},
		},
		Outputs: []NamedField{},
	}
}

func bigIntStr(v uint64) string {
	return new(big.Int).SetUint64(v).String()
}

func makeInvalidTx(contractType string) types.Transaction {
	return types.Transaction{
		OperationMetadata: types.OperationMetadata{ContractType: contractType},
		To:                testAddress,
		Data:              []byte{0xFF, 0xFF},
		AdditionalFields:  json.RawMessage(`{"value":0}`),
	}
}
