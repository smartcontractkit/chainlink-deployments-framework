package analyzer

import (
	"encoding/json"
	"testing"

	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-stellar/bindings"
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

const (
	// stellarProposerMCM is the Stellar testnet proposer ManyChainMultiSig.
	stellarProposerMCM = "CB66PWVWBA765OSEMEPL3766WXEFPVGVMVS5UHNPEP266Y6QZGFJQG4B"
	// stellarTimelock is the Stellar testnet RBACTimelock.
	stellarTimelock = "CAZPM2APSKKYZEGGS3E3V6MONKF2ZQZSZVCFSZOSDYPQK3XQ2XUO2LQW"

	stellarAdditionalFields = `{"family":"stellar","encodingVersion":1}`
)

func stellarProposalContext() *DefaultProposalContext {
	return &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{
			chainsel.STELLAR_TESTNET.Selector: {
				stellarProposerMCM: deployment.MustTypeAndVersionFromString("ProposerManyChainMultiSig 1.0.0"),
			},
		},
	}
}

func mustEncodeStellarPayload(t *testing.T, function string, args ...xdr.ScVal) []byte {
	t.Helper()

	data, err := bindings.EncodeSorobanInvokePayload(function, args)
	require.NoError(t, err)

	return data
}

func mustStructScVal(t *testing.T, fields map[string]xdr.ScVal) xdr.ScVal {
	t.Helper()

	val, err := scval.BuildStructScVal(fields)
	require.NoError(t, err)

	return val
}

func TestAnalyzeStellarTransactions(t *testing.T) {
	t.Parallel()

	chainSelector := chainsel.STELLAR_TESTNET.Selector

	tests := []struct {
		name   string
		mcmsTx types.Transaction
		want   *DecodedCall
	}{
		{
			// The exact payload emitted by stellar_mcms_transfer_ownership.
			name: "no-arg call resolves the contract type from the datastore",
			mcmsTx: types.Transaction{
				To:               stellarProposerMCM,
				Data:             mustEncodeStellarPayload(t, "accept_ownership"),
				AdditionalFields: json.RawMessage(stellarAdditionalFields),
			},
			want: &DecodedCall{
				Address:         stellarProposerMCM,
				Method:          "accept_ownership",
				Inputs:          []NamedField{},
				Outputs:         []NamedField{},
				ContractType:    "ProposerManyChainMultiSig",
				ContractVersion: "1.0.0",
			},
		},
		{
			name: "address and integer arguments",
			mcmsTx: types.Transaction{
				To: stellarProposerMCM,
				Data: mustEncodeStellarPayload(t, "transfer_ownership",
					scval.AddressToScVal(stellarTimelock),
					scval.Uint32ToScVal(12345),
				),
				AdditionalFields: json.RawMessage(stellarAdditionalFields),
			},
			want: &DecodedCall{
				Address: stellarProposerMCM,
				Method:  "transfer_ownership",
				Inputs: []NamedField{
					{Name: "arg0", TypeName: "Address", Value: AddressField{Value: stellarTimelock}},
					{Name: "arg1", TypeName: "U32", Value: SimpleField{Value: "12345"}},
				},
				Outputs:         []NamedField{},
				ContractType:    "ProposerManyChainMultiSig",
				ContractVersion: "1.0.0",
			},
		},
		{
			name: "malformed payload is reported per call, not fatal",
			mcmsTx: types.Transaction{
				To:               stellarProposerMCM,
				Data:             []byte{0x01, 0x02, 0x03},
				AdditionalFields: json.RawMessage(stellarAdditionalFields),
			},
			want: nil, // asserted separately below
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := AnalyzeStellarTransactions(
				stellarProposalContext(), chainSelector, []types.Transaction{tt.mcmsTx},
			)
			require.NoError(t, err)
			require.Len(t, got, 1)

			if tt.want == nil {
				require.Contains(t, got[0].Method, "failed to decode Stellar transaction")
				require.Equal(t, stellarProposerMCM, got[0].Address)
				require.Equal(t, "ProposerManyChainMultiSig", got[0].ContractType)

				return
			}

			// RawValue holds the original ScVal, which is not part of the expectation.
			for i := range got[0].Inputs {
				got[0].Inputs[i].RawValue = nil
			}
			require.Equal(t, tt.want, got[0])
		})
	}
}

func TestStellarScValField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  xdr.ScVal
		want FieldValue
	}{
		{
			name: "address",
			val:  scval.AddressToScVal(stellarTimelock),
			want: AddressField{Value: stellarTimelock},
		},
		{
			name: "bytes",
			val:  scval.BytesToScVal([]byte{0xde, 0xad, 0xbe, 0xef}),
			want: BytesField{Value: []byte{0xde, 0xad, 0xbe, 0xef}},
		},
		{
			name: "vector recurses into its elements",
			val: scval.VecToScVal([]xdr.ScVal{
				scval.Uint32ToScVal(1),
				scval.AddressToScVal(stellarTimelock),
			}),
			want: ArrayField{Elements: []FieldValue{
				SimpleField{Value: "1"},
				AddressField{Value: stellarTimelock},
			}},
		},
		{
			name: "map recurses and keeps its keys as field names",
			val:  mustStructScVal(t, map[string]xdr.ScVal{"description": scval.StringToScVal("BTC/USD")}),
			want: StructField{Fields: []NamedField{
				{
					Name:     "description",
					TypeName: "String",
					Value:    SimpleField{Value: "BTC/USD"},
					RawValue: scval.StringToScVal("BTC/USD"),
				},
			}},
		},
		{
			name: "symbol falls back to the scalar rendering",
			val:  scval.SymbolToScVal("schedule"),
			want: SimpleField{Value: "schedule"},
		},
		{
			name: "u64 falls back to the scalar rendering",
			val:  scval.Uint64ToScVal(1234567890),
			want: SimpleField{Value: "1234567890"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, stellarScValField(tt.val))
		})
	}
}

func TestStellarScValTypeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  xdr.ScVal
		want string
	}{
		{name: "address", val: scval.AddressToScVal(stellarTimelock), want: "Address"},
		{name: "u32", val: scval.Uint32ToScVal(1), want: "U32"},
		{name: "bytes", val: scval.BytesToScVal([]byte{0x01}), want: "Bytes"},
		{name: "symbol", val: scval.SymbolToScVal("schedule"), want: "Symbol"},
		{name: "vec", val: scval.VecToScVal([]xdr.ScVal{}), want: "Vec"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, stellarScValTypeName(tt.val))
		})
	}
}
