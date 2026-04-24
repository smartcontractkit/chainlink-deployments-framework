package evm_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"

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
			if got := evm.AbiToGoType(tc.solidity); got != tc.want {
				t.Errorf("solidityToGoType(%q) = %q, want %q", tc.solidity, got, tc.want)
			}
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
			if got := evm.SanitizeFieldName(tc.input); got != tc.want {
				t.Errorf("SanitizeFieldName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestReadABI(t *testing.T) {
	t.Parallel()
	cfg := evm.EvmContractConfig{
		Name:              "LinkToken",
		GobindingsPackage: "github.com/smartcontractkit/chainlink-evm/gethwrappers/shared/generated/initial/link_token",
	}

	parsedABI, err := evm.ReadABI(cfg)
	if err != nil {
		t.Fatalf("ReadABI returned error: %v", err)
	}
	if parsedABI == nil {
		t.Fatal("expected parsed ABI, got nil")
	}
	if _, ok := parsedABI.Methods["transfer"]; !ok {
		t.Fatal("expected transfer method in ABI")
	}
}

func TestFindFunctionInABINotFound(t *testing.T) {
	t.Parallel()
	parsed := abi.ABI{
		Methods: map[string]abi.Method{
			"transfer": {Name: "transfer", RawName: "transfer"},
		},
	}
	if got := evm.FindFunctionInABI(parsed, "mint"); got != nil {
		t.Errorf("expected nil for missing function, got %v", got)
	}
}
