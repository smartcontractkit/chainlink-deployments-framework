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

	stellarAdditionalFieldsJSON = `{"family":"stellar","encodingVersion":1}`
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
				AdditionalFields: json.RawMessage(stellarAdditionalFieldsJSON),
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
				AdditionalFields: json.RawMessage(stellarAdditionalFieldsJSON),
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
				AdditionalFields: json.RawMessage(stellarAdditionalFieldsJSON),
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

// TestAnalyzeStellarTransaction_AdditionalFields pins that an encoding this analyzer does not
// implement is reported rather than decoded under v1 assumptions: a future wire format could
// still parse as an ScVal vector and show a reviewer a confident, wrong method name.
func TestAnalyzeStellarTransaction_AdditionalFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		additionalFields string
		wantMethod       string
	}{
		{
			name:             "current encoding decodes",
			additionalFields: stellarAdditionalFieldsJSON,
			wantMethod:       "accept_ownership",
		},
		{
			name:             "absent additional fields fall back to best-effort decoding",
			additionalFields: "",
			wantMethod:       "accept_ownership",
		},
		{
			name:             "unparseable additional fields fall back to best-effort decoding",
			additionalFields: "not json",
			wantMethod:       "accept_ownership",
		},
		{
			name:             "fields present but empty decode",
			additionalFields: `{}`,
			wantMethod:       "accept_ownership",
		},
		{
			name:             "a future encoding version is reported, not decoded",
			additionalFields: `{"family":"stellar","encodingVersion":2}`,
			wantMethod:       "unsupported Stellar MCMS encoding version 2",
		},
		{
			name:             "a mismatched family is reported, not decoded",
			additionalFields: `{"family":"aptos","encodingVersion":1}`,
			wantMethod:       `unexpected transaction family "aptos"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := AnalyzeStellarTransaction(
				stellarProposalContext(), chainsel.STELLAR_TESTNET.Selector,
				types.Transaction{
					To:               stellarProposerMCM,
					Data:             mustEncodeStellarPayload(t, "accept_ownership"),
					AdditionalFields: json.RawMessage(tt.additionalFields),
				},
			)
			require.NoError(t, err)
			require.Contains(t, got.Method, tt.wantMethod)
			require.Equal(t, "ProposerManyChainMultiSig", got.ContractType)
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

// TestStellarScValField_MapKeyNames pins that distinct map keys never collapse into one field
// name. UPF serializes a StructField into a map[string]any keyed by NamedField.Name, so a
// collision silently drops entries from the reviewer-facing YAML.
func TestStellarScValField_MapKeyNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		keys      []xdr.ScVal
		wantNames []string
	}{
		{
			name:      "symbol keys keep their bare text",
			keys:      []xdr.ScVal{scval.SymbolToScVal("description"), scval.SymbolToScVal("data_id")},
			wantNames: []string{"description", "data_id"},
		},
		{
			name: "keys that render identically stay distinct",
			keys: []xdr.ScVal{
				scval.Uint32ToScVal(1),
				scval.StringToScVal("1"),
				scval.SymbolToScVal("1"),
			},
			wantNames: []string{"U32(1)", "String(1)", "1"},
		},
		{
			name: "a symbol shaped like a qualified key is still disambiguated",
			keys: []xdr.ScVal{
				scval.Uint32ToScVal(1),
				scval.SymbolToScVal("U32(1)"),
			},
			wantNames: []string{"U32(1)", "U32(1)#1"},
		},
		{
			// All three keys are distinct and legal. A single base#<index> attempt would give the
			// third entry the second's name, dropping it from the UPF map.
			name: "a suffixed candidate that is itself taken keeps probing",
			keys: []xdr.ScVal{
				scval.Uint32ToScVal(1),
				scval.SymbolToScVal("U32(1)#2"),
				scval.SymbolToScVal("U32(1)"),
			},
			wantNames: []string{"U32(1)", "U32(1)#2", "U32(1)#3"},
		},
		{
			// Duplicate keys are invalid on-chain but the decoder does not validate uniqueness,
			// so a malformed payload must not be able to hide an entry from reviewers.
			name: "duplicate keys each keep a distinct name",
			keys: []xdr.ScVal{
				scval.SymbolToScVal("x"),
				scval.SymbolToScVal("x#2"),
				scval.SymbolToScVal("x"),
			},
			wantNames: []string{"x", "x#2", "x#3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entries := make(xdr.ScMap, 0, len(tt.keys))
			for _, key := range tt.keys {
				entries = append(entries, xdr.ScMapEntry{Key: key, Val: scval.Uint32ToScVal(7)})
			}
			mapPtr := &entries
			val := xdr.ScVal{Type: xdr.ScValTypeScvMap, Map: &mapPtr}

			got, ok := stellarScValField(val).(StructField)
			require.True(t, ok)

			gotNames := make([]string, 0, len(got.Fields))
			for _, field := range got.Fields {
				gotNames = append(gotNames, field.Name)
			}
			require.Equal(t, tt.wantNames, gotNames)
			require.Len(t, gotNames, len(tt.keys), "every entry must survive")

			unique := make(map[string]struct{}, len(gotNames))
			for _, name := range gotNames {
				unique[name] = struct{}{}
			}
			require.Len(t, unique, len(tt.keys), "field names must be collision-free")
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
