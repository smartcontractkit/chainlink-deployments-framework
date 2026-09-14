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

	t.Run("bytes", func(t *testing.T) {
		t.Parallel()

		got := stellarScValField(scval.BytesToScVal([]byte{0xde, 0xad, 0xbe, 0xef}))
		require.Equal(t, BytesField{Value: []byte{0xde, 0xad, 0xbe, 0xef}}, got)
	})

	t.Run("vector recurses", func(t *testing.T) {
		t.Parallel()

		got := stellarScValField(scval.VecToScVal([]xdr.ScVal{
			scval.Uint32ToScVal(1),
			scval.AddressToScVal(stellarTimelock),
		}))
		require.Equal(t, ArrayField{Elements: []FieldValue{
			SimpleField{Value: "1"},
			AddressField{Value: stellarTimelock},
		}}, got)
	})

	t.Run("symbol falls back to the scalar rendering", func(t *testing.T) {
		t.Parallel()

		got := stellarScValField(scval.SymbolToScVal("schedule"))
		require.Equal(t, SimpleField{Value: "schedule"}, got)
	})
}

func TestStellarScValTypeName(t *testing.T) {
	t.Parallel()

	require.Equal(t, "Address", stellarScValTypeName(scval.AddressToScVal(stellarTimelock)))
	require.Equal(t, "U32", stellarScValTypeName(scval.Uint32ToScVal(1)))
	require.Equal(t, "Bytes", stellarScValTypeName(scval.BytesToScVal([]byte{0x01})))
}
