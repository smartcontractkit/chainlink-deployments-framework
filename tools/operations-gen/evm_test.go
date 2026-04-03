package main

import (
	"testing"
)

// TestToSnakeCase covers the EVM name-normalisation helper including
// the nameOverrides entries for contracts with non-standard casing.
func TestToSnakeCase(t *testing.T) {
	t.Parallel()
	cases := []struct{ input, want string }{
		{"OnRamp", "onramp"},
		{"OffRamp", "offramp"},
		{"LinkToken", "link_token"},
		{"FeeQuoter", "fee_quoter"},
		{"EVM2EVMOnRamp", "evm2evm_on_ramp"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			if got := toSnakeCase(tc.input); got != tc.want {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

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
		// arrays
		{"uint256[]", "[]*big.Int"},
		{"address[]", "[]common.Address"},
		// unknown scalar → any
		{"uint512", "any"},
		// tuple → any
		{"tuple", "any"},
		{"tuple[]", "any"},
	}
	for _, tc := range cases {
		t.Run(tc.solidity, func(t *testing.T) {
			t.Parallel()
			if got := solidityToGoType(tc.solidity, evmTypeMap); got != tc.want {
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
		// empty
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.internalType, func(t *testing.T) {
			t.Parallel()
			if got := extractStructName(tc.internalType); got != tc.want {
				t.Errorf("extractStructName(%q) = %q, want %q", tc.internalType, got, tc.want)
			}
		})
	}
}

func TestFindFunctionInABIOverloads(t *testing.T) {
	t.Parallel()
	entries := []ABIEntry{
		{Type: "function", Name: "transfer", Inputs: []ABIParam{{Name: "to", Type: "address"}, {Name: "amount", Type: "uint256"}}, StateMutability: "nonpayable"},
		{Type: "function", Name: "transfer", Inputs: []ABIParam{{Name: "to", Type: "address"}, {Name: "amount", Type: "uint256"}, {Name: "data", Type: "bytes"}}, StateMutability: "nonpayable"},
		{Type: "function", Name: "transfer", Inputs: []ABIParam{{Name: "to", Type: "address"}, {Name: "amount", Type: "uint256"}, {Name: "data", Type: "bytes"}, {Name: "extra", Type: "bytes32"}}, StateMutability: "nonpayable"},
	}

	results := findFunctionInABI(entries, "transfer", "mypkg", evmTypeMap)

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
	cfg := evmContractConfig{ABIFile: "contract.abi"}
	_, _, err := readABIAndBytecode(cfg, "contract", "v1_0_0", t.TempDir())
	if err == nil {
		t.Fatal("expected error for abi_file without .json suffix, got nil")
	}
}

func TestFindFunctionInABINotFound(t *testing.T) {
	t.Parallel()
	entries := []ABIEntry{
		{Type: "function", Name: "transfer"},
	}
	if got := findFunctionInABI(entries, "mint", "pkg", evmTypeMap); got != nil {
		t.Errorf("expected nil for missing function, got %v", got)
	}
}

func TestCheckNeedsBigInt(t *testing.T) {
	t.Parallel()
	makeFuncInfo := func(goType string) *functionInfo {
		return &functionInfo{
			Name:       "Foo",
			Parameters: []parameterInfo{{GoType: goType}},
		}
	}

	t.Run("parameter needs big.Int", func(t *testing.T) {
		t.Parallel()
		info := &contractInfo{
			Functions:     map[string]*functionInfo{"Foo": makeFuncInfo("*big.Int")},
			FunctionOrder: []string{"Foo"},
		}
		if !checkNeedsBigInt(info) {
			t.Error("expected true")
		}
	})

	t.Run("return param needs big.Int", func(t *testing.T) {
		t.Parallel()
		info := &contractInfo{
			Functions: map[string]*functionInfo{
				"Foo": {Name: "Foo", ReturnParams: []parameterInfo{{GoType: "*big.Int"}}},
			},
			FunctionOrder: []string{"Foo"},
		}
		if !checkNeedsBigInt(info) {
			t.Error("expected true")
		}
	})

	t.Run("constructor param needs big.Int", func(t *testing.T) {
		t.Parallel()
		entry := ABIEntry{
			Type:   "constructor",
			Inputs: []ABIParam{{Name: "supply", Type: "uint256"}},
		}
		fi := parseABIFunction(entry, "pkg", evmTypeMap)
		info := &contractInfo{
			Constructor:   fi,
			Functions:     map[string]*functionInfo{},
			FunctionOrder: []string{},
		}
		if !checkNeedsBigInt(info) {
			t.Error("expected true for constructor uint256 param")
		}
	})

	t.Run("no big.Int", func(t *testing.T) {
		t.Parallel()
		info := &contractInfo{
			Functions:     map[string]*functionInfo{"Foo": makeFuncInfo("common.Address")},
			FunctionOrder: []string{"Foo"},
		}
		if checkNeedsBigInt(info) {
			t.Error("expected false")
		}
	})
}
