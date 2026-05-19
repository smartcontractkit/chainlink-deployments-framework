package analyzer

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/lib/access/rbac"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tlbe"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

const testAddress = "EQDtFpEwcFAEcRe5mLVh2N6C0x-_hJEM7W61_JLnSF74p4q2"

func TestAnalyzeTONTransaction(t *testing.T) {
	t.Parallel()

	setup := newTestTONSetup(t)
	ctx := &DefaultProposalContext{}
	decoder := ton.NewDecoder(bindings.Registry)

	tests := []struct {
		name           string
		decoder        sdk.Decoder
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
			mcmsTx:         makeInvalidTx(bindings.ShortMCMS, bindings.TypeMCMS),
			want:           &DecodedCall{Address: testAddress},
			wantErrContain: "invalid cell BOC data",
		},
		{
			name: "unknown contract type",
			mcmsTx: types.Transaction{
				OperationMetadata: types.OperationMetadata{ContractType: "unknown.type", ContractVersion: semver.MustParse("1.2.3")},
				To:                testAddress,
				Data: func() []byte {
					c := cell.BeginCell()
					c.MustStoreBinarySnake([]byte{0x01, 0x02})

					return c.EndCell().ToBOC()
				}(),
				AdditionalFields: json.RawMessage(`{"value":0, "contractTypeFull":"com.example.package.unknown.contract"}`),
			},
			want:           &DecodedCall{Address: testAddress},
			wantErrContain: "unknown contract interface: com.example.package.unknown.contract",
		},
		{
			name: "fail - unknown version",
			mcmsTx: func() types.Transaction {
				tx := setup.makeGrantRoleTx(t, 1)
				tx.ContractVersion = semver.MustParse("0.0.0")

				return tx
			}(),
			want:           &DecodedCall{Address: setup.targetAddr.String()},
			wantErrContain: "unknown contract interface: link.chain.ton.lib.access.RBAC@0.0.0",
		},
		{
			name: "success - GrantRole with version",
			decoder: ton.NewDecoder(tvm.ContractTLBRegistry{
				bindings.TypeRBAC + "@1.2.3": bindings.Registry[bindings.TypeRBAC],
			}),
			mcmsTx: func() types.Transaction {
				tx := setup.makeGrantRoleTx(t, 1)
				tx.ContractVersion = semver.MustParse("1.2.3")

				return tx
			}(),
			want: setup.expectedGrantRoleCall(1, "1.2.3"),
		},
		{
			name: "empty data",
			mcmsTx: types.Transaction{
				OperationMetadata: types.OperationMetadata{ContractType: bindings.ShortMCMS},
				To:                testAddress,
				Data:              []byte{},
				AdditionalFields:  json.RawMessage(`{"value":0, "contractTypeFull":"` + bindings.TypeMCMS + `"}`),
			},
			want:           &DecodedCall{Address: testAddress},
			wantErrContain: "invalid cell BOC data",
		},
		{
			name: "empty cell",
			mcmsTx: types.Transaction{
				OperationMetadata: types.OperationMetadata{ContractType: bindings.ShortMCMS},
				To:                testAddress,
				Data:              tvm.EmptyCell.ToBOC(),
				AdditionalFields:  json.RawMessage(`{"value":0}`),
			},
			want: &DecodedCall{
				Address:         testAddress,
				Method:          "::(0x0)",
				Inputs:          []NamedField{},
				Outputs:         []NamedField{},
				ContractType:    bindings.ShortMCMS,
				ContractVersion: "",
			},
			wantErrContain: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testDecoder := decoder
			if tt.decoder != nil {
				testDecoder = tt.decoder
			}
			result, err := AnalyzeTONTransaction(ctx, testDecoder, 0, tt.mcmsTx)
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
				makeInvalidTx(bindings.ShortMCMS, bindings.TypeMCMS),
				setup.makeGrantRoleTx(t, 1),
				makeInvalidTx(bindings.ShortTimelock, bindings.TypeTimelock),
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
				makeInvalidTx(bindings.ShortMCMS, bindings.TypeMCMS),
				makeInvalidTx(bindings.ShortTimelock, bindings.TypeTimelock),
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

			results, err := AnalyzeTONTransactions(ctx, 0, tt.txs)
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

func TestAnalyzeTONTransactionResolveContractInfo(t *testing.T) {
	t.Parallel()

	setup := newTestTONSetup(t)
	decoder := ton.NewDecoder(bindings.Registry)
	chainSelector := uint64(123456)
	tx := setup.makeGrantRoleTx(t, 1)

	t.Run("resolves contract type and version from proposal context", func(t *testing.T) {
		t.Parallel()

		ctx := &DefaultProposalContext{
			AddressesByChain: deployment.AddressesByChain{
				chainSelector: {
					tx.To: deployment.MustTypeAndVersionFromString("TONRBAC 1.2.3"),
				},
			},
		}

		result, err := AnalyzeTONTransaction(ctx, decoder, chainSelector, tx)
		require.NoError(t, err)
		require.Equal(t, "TONRBAC", result.ContractType)
		require.Equal(t, "1.2.3", result.ContractVersion)
	})

	t.Run("falls back when address is not in proposal context", func(t *testing.T) {
		t.Parallel()

		ctx := &DefaultProposalContext{
			AddressesByChain: deployment.AddressesByChain{
				chainSelector: {
					"EQCxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx": deployment.MustTypeAndVersionFromString("OtherContract 9.9.9"),
				},
			},
		}

		result, err := AnalyzeTONTransaction(ctx, decoder, chainSelector, tx)
		require.NoError(t, err)
		require.Equal(t, tx.ContractType, result.ContractType)
		require.Empty(t, result.ContractVersion)
	})
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
		bindings.ShortRBAC,
		nil,
		bindings.TypeRBAC,
		[]string{"grantRole"},
	)
	require.NoError(t, err)

	return tx
}

func (s *testTONSetup) expectedGrantRoleCall(queryID uint64, version ...string) *DecodedCall {
	methodPrefix := string(bindings.TypeRBAC)
	versionStr := ""
	if len(version) > 0 {
		versionStr = version[0]
		methodPrefix = methodPrefix + "@" + versionStr
	}

	return &DecodedCall{
		Address:         s.targetAddr.String(),
		Method:          methodPrefix + "::GrantRole(0x95cd540f)",
		ContractType:    string(bindings.ShortRBAC),
		ContractVersion: versionStr,
		Inputs: []NamedField{
			{Name: "QueryID", Value: SimpleField{Value: bigIntStr(queryID)}, RawValue: queryID},
			{Name: "Role", Value: SimpleField{Value: s.exampleRoleBig.String()}, RawValue: tlbe.NewUint256(s.exampleRoleBig)},
			{Name: "Account", Value: SimpleField{Value: s.targetAddr.String()}, RawValue: s.targetAddr},
		},
		Outputs: []NamedField{},
	}
}

func bigIntStr(v uint64) string {
	return new(big.Int).SetUint64(v).String()
}

func makeInvalidTx(contractType string, contractTypeFull tvm.FullyQualifiedName) types.Transaction {
	return types.Transaction{
		OperationMetadata: types.OperationMetadata{ContractType: contractType},
		To:                testAddress,
		Data:              []byte{0xFF, 0xFF},
		AdditionalFields:  json.RawMessage(`{"value":0, "contractTypeFull":"` + contractTypeFull + `"}`),
	}
}
