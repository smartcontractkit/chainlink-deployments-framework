package evm_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/families/evm"
)

func TestSolidityToGoType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		solidity abi.Type
		want     string
	}{
		{abi.Type{T: abi.AddressTy}, "common.Address"},
		{abi.Type{T: abi.BoolTy}, "bool"},
		{abi.Type{T: abi.StringTy}, "string"},
		{abi.Type{T: abi.FixedBytesTy, Size: 32}, "[32]byte"},
		// dynamic arrays
		{abi.Type{T: abi.ArrayTy, Elem: &abi.Type{T: abi.UintTy, Size: 256}}, "[]*big.Int"},
		{abi.Type{T: abi.ArrayTy, Elem: &abi.Type{T: abi.AddressTy}}, "[]common.Address"},
		// fixed-size arrays
		{abi.Type{T: abi.ArrayTy, Size: 32, Elem: &abi.Type{T: abi.UintTy, Size: 8}}, "[32]uint8"},
		{abi.Type{T: abi.ArrayTy, Size: 4, Elem: &abi.Type{T: abi.FixedBytesTy, Size: 32}}, "[4][32]byte"},
		// native-width unsigned ints
		{abi.Type{T: abi.UintTy, Size: 8}, "uint8"},
		{abi.Type{T: abi.UintTy, Size: 16}, "uint16"},
		{abi.Type{T: abi.UintTy, Size: 32}, "uint32"},
		{abi.Type{T: abi.UintTy, Size: 64}, "uint64"},
		// non-native unsigned widths fall back to *big.Int to match abigen
		{abi.Type{T: abi.UintTy, Size: 24}, "*big.Int"},
		{abi.Type{T: abi.UintTy, Size: 40}, "*big.Int"},
		{abi.Type{T: abi.UintTy, Size: 48}, "*big.Int"},
		{abi.Type{T: abi.UintTy, Size: 56}, "*big.Int"},
		{abi.Type{T: abi.UintTy, Size: 72}, "*big.Int"},
		{abi.Type{T: abi.UintTy, Size: 128}, "*big.Int"},
		{abi.Type{T: abi.UintTy, Size: 256}, "*big.Int"},
		// native-width signed ints
		{abi.Type{T: abi.IntTy, Size: 8}, "int8"},
		{abi.Type{T: abi.IntTy, Size: 16}, "int16"},
		{abi.Type{T: abi.IntTy, Size: 32}, "int32"},
		{abi.Type{T: abi.IntTy, Size: 64}, "int64"},
		// non-native signed widths fall back to *big.Int to match abigen
		{abi.Type{T: abi.IntTy, Size: 24}, "*big.Int"},
		{abi.Type{T: abi.IntTy, Size: 40}, "*big.Int"},
		{abi.Type{T: abi.IntTy, Size: 48}, "*big.Int"},
		{abi.Type{T: abi.IntTy, Size: 56}, "*big.Int"},
		{abi.Type{T: abi.IntTy, Size: 72}, "*big.Int"},
		{abi.Type{T: abi.IntTy, Size: 128}, "*big.Int"},
		{abi.Type{T: abi.IntTy, Size: 256}, "*big.Int"},
		// tuples resolve to the gobindings-generated struct name
		{abi.Type{T: abi.TupleTy, TupleRawName: "TestStruct"}, "gobindings.TestStruct"},
		{abi.Type{T: abi.ArrayTy, Elem: &abi.Type{T: abi.TupleTy, TupleRawName: "TestStruct"}}, "[]gobindings.TestStruct"},
		// anonymous tuples have no gobindings struct — must degrade to `any`
		// rather than emitting the invalid identifier "gobindings."
		{abi.Type{T: abi.TupleTy, TupleRawName: ""}, "any"},
		{abi.Type{T: abi.SliceTy, Elem: &abi.Type{T: abi.TupleTy, TupleRawName: ""}}, "[]any"},
		{abi.Type{T: abi.ArrayTy, Size: 3, Elem: &abi.Type{T: abi.TupleTy, TupleRawName: ""}}, "[3]any"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, evm.AbiToGoType(tc.solidity), "solidityToGoType(%q)", tc.solidity)
		})
	}
}

func TestSanitizeFieldName(t *testing.T) {
	t.Parallel()
	cases := []struct{ input, want string }{
		// ABI params with leading underscores (common Solidity convention)
		{"_to", "To"},
		{"_value", "Value"},
		{"_spender", "Spender"},
		// Multiple leading underscores
		{"__foo", "Foo"},
		// No underscore — plain capitalize
		{"balance", "Balance"},
		{"owner", "Owner"},
		// Already exported
		{"Amount", "Amount"},
		// Leading underscore followed by digit — result starts with digit, invalid Go identifier
		{"_1", ""},
		{"__2foo", ""},
		// Empty
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, evm.SanitizeFieldName(tc.input), "SanitizeFieldName(%q)", tc.input)
		})
	}
}

func TestReadABI(t *testing.T) {
	t.Parallel()
	cfg := evm.EvmContractConfig{
		Name:              "LinkToken",
		GobindingsPackage: "github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/testdata/evm/gobindings/v1_0_0/link_token",
	}

	parsedABI, err := evm.ReadABI(cfg)
	require.NoError(t, err, "ReadABI returned error")
	require.NotNil(t, parsedABI, "expected parsed ABI, got nil")
	_, ok := parsedABI.Methods["transfer"]
	require.True(t, ok, "expected transfer method in ABI")
}

func TestReadABIPopulatesTupleRawNamesFromGobindings(t *testing.T) {
	t.Parallel()
	cfg := evm.EvmContractConfig{
		Name:              "ManyChainMultiSig",
		GobindingsPackage: "github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/testdata/evm/gobindings/v1_0_0/many_chain_multi_sig",
	}

	parsedABI, err := evm.ReadABI(cfg)
	require.NoError(t, err, "ReadABI returned error")

	setRoot, ok := parsedABI.Methods["setRoot"]
	require.True(t, ok, "expected setRoot method in ABI")

	require.Equal(t, "ManyChainMultiSigRootMetadata", setRoot.Inputs[2].Type.TupleRawName)

	signaturesType := setRoot.Inputs[4].Type
	require.NotNil(t, signaturesType.Elem, "expected signatures type to have element type")
	require.Equal(t, "ManyChainMultiSigSignature", signaturesType.Elem.TupleRawName)
}

func TestFindFunctionInABINotFound(t *testing.T) {
	t.Parallel()
	parsed := abi.ABI{
		Methods: map[string]abi.Method{
			"transfer": {Name: "transfer", RawName: "transfer"},
		},
	}
	require.Nil(t, evm.FindFunctionInABI(&parsed, "mint"))
}

func TestNormalizeStructInternalTypes(t *testing.T) {
	t.Parallel()

	input := `[
		{"internalType":"structFeeAggregator.ConstructorParams","type":"tuple"},
		{"internalType":"structClient.EVMTokenAmount[]","type":"tuple[]"},
		{"internalType":"struct Common.AssetAmount[]","type":"tuple[]"}
	]`

	got := evm.NormalizeStructInternalTypes(input)

	wantContains := []string{
		`"internalType":"struct FeeAggregator.ConstructorParams"`,
		`"internalType":"struct Client.EVMTokenAmount[]"`,
		`"internalType":"struct Common.AssetAmount[]"`,
	}
	for _, want := range wantContains {
		require.Contains(t, got, want)
	}
}
