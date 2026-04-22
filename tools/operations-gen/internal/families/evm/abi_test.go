package evm_test

import (
	"testing"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/families/evm"
)

func TestSolidityToGoType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		solidity string
		want     string
	}{
		{"uint256", "*big.Int"},
		{"address", "common.Address"},
		{"bool", "bool"},
		{"string", "string"},
		{"bytes32", "[32]byte"},
		// dynamic arrays
		{"uint256[]", "[]*big.Int"},
		{"address[]", "[]common.Address"},
		// fixed-size arrays
		{"uint8[32]", "[32]uint8"},
		{"bytes32[4]", "[4][32]byte"},
		// intermediate uint sizes
		{"uint40", "uint64"},
		{"uint48", "uint64"},
		{"uint56", "uint64"},
		// unknown scalar → any
		{"uint512", "any"},
		// unknown scalar fixed-size array → any
		{"uint512[4]", "any"},
		// malformed arrays should never panic and should degrade to any
		{"[", "any"},
		{"uint8[", "any"},
		// tuple → any
		{"tuple", "any"},
		{"tuple[]", "any"},
	}
	for _, tc := range cases {
		t.Run(tc.solidity, func(t *testing.T) {
			t.Parallel()
			if got := evm.SolidityToGoType(tc.solidity, evm.EvmTypeMap); got != tc.want {
				t.Errorf("solidityToGoType(%q) = %q, want %q", tc.solidity, got, tc.want)
			}
		})
	}
}

func TestExtractStructName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		internalType string
		want         string
	}{
		{"struct IOnRamp.DestChainConfig", "DestChainConfig"},
		{"struct IOnRamp.DestChainConfig[]", "DestChainConfig"},
		{"struct MyContract.Foo", "Foo"},
		// no dot — whole string minus [] suffix
		{"DestChainConfig", "DestChainConfig"},
		// "struct " prefix without a module qualifier
		{"struct MyStruct", "MyStruct"},
		{"struct MyStruct[]", "MyStruct"},
		// anonymous tuples — no struct name, caller falls back to any
		{"tuple", ""},
		{"tuple[]", ""},
		// empty
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.internalType, func(t *testing.T) {
			t.Parallel()
			if got := evm.ExtractStructName(tc.internalType); got != tc.want {
				t.Errorf("ExtractStructName(%q) = %q, want %q", tc.internalType, got, tc.want)
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

func TestSanitizeParamName(t *testing.T) {
	t.Parallel()
	cases := []struct{ input, want string }{
		{"_to", "to"},
		{"_value", "value"},
		{"_spender", "spender"},
		{"__foo", "foo"},
		// No underscore — lowercase first char
		{"Balance", "balance"},
		{"owner", "owner"},
		// Leading underscore followed by digit — result starts with digit, invalid Go identifier
		{"_1", ""},
		// Empty
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			if got := evm.SanitizeParamName(tc.input); got != tc.want {
				t.Errorf("SanitizeParamName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFindFunctionInABIOverloads(t *testing.T) {
	t.Parallel()
	entries := []evm.ABIEntry{
		{Type: "function", Name: "transfer", Inputs: []evm.ABIParam{{Name: "to", Type: "address"}, {Name: "amount", Type: "uint256"}}, StateMutability: "nonpayable"},
		{Type: "function", Name: "transfer", Inputs: []evm.ABIParam{{Name: "to", Type: "address"}, {Name: "amount", Type: "uint256"}, {Name: "data", Type: "bytes"}}, StateMutability: "nonpayable"},
		{Type: "function", Name: "transfer", Inputs: []evm.ABIParam{{Name: "to", Type: "address"}, {Name: "amount", Type: "uint256"}, {Name: "data", Type: "bytes"}, {Name: "extra", Type: "bytes32"}}, StateMutability: "nonpayable"},
	}

	results := evm.FindFunctionInABI(entries, "transfer", "mypkg", evm.EvmTypeMap)

	if len(results) != 3 {
		t.Fatalf("expected 3 overloads, got %d", len(results))
	}
	// First overload: no suffix
	if results[0].Name != "Transfer" || results[0].CallMethod != "transfer" {
		t.Errorf("overload[0]: got Name=%q CallMethod=%q", results[0].Name, results[0].CallMethod)
	}
	// Second overload: suffix "0"
	if results[1].Name != "Transfer0" || results[1].CallMethod != "transfer0" {
		t.Errorf("overload[1]: got Name=%q CallMethod=%q", results[1].Name, results[1].CallMethod)
	}
	// Third overload: suffix "1"
	if results[2].Name != "Transfer1" || results[2].CallMethod != "transfer1" {
		t.Errorf("overload[2]: got Name=%q CallMethod=%q", results[2].Name, results[2].CallMethod)
	}
}

func TestReadABIAndBytecodeInvalidABIFileSuffix(t *testing.T) {
	t.Parallel()
	cfg := evm.EvmContractConfig{ABIFile: "contract.abi"}
	_, _, err := evm.ReadABIAndBytecode(cfg, "contract", "v1_0_0", evm.EvmInputConfig{
		ABIBasePath:      t.TempDir(),
		BytecodeBasePath: t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected error for abi_file without .json suffix, got nil")
	}
}

func TestFindFunctionInABINotFound(t *testing.T) {
	t.Parallel()
	entries := []evm.ABIEntry{
		{Type: "function", Name: "transfer"},
	}
	if got := evm.FindFunctionInABI(entries, "mint", "pkg", evm.EvmTypeMap); got != nil {
		t.Errorf("expected nil for missing function, got %v", got)
	}
}
