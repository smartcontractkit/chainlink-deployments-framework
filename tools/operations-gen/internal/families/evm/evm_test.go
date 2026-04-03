package evm

import (
	"testing"
)

// TestToSnakeCase covers the algorithmic EVM name-normalisation helper
// across representative contract names and mixed casing patterns.
func TestToSnakeCase(t *testing.T) {
	t.Parallel()
	cases := []struct{ input, want string }{
		{"OnRamp", "on_ramp"},
		{"OffRamp", "off_ramp"},
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
			if got := extractStructName(tc.internalType); got != tc.want {
				t.Errorf("extractStructName(%q) = %q, want %q", tc.internalType, got, tc.want)
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
			if got := sanitizeFieldName(tc.input); got != tc.want {
				t.Errorf("sanitizeFieldName(%q) = %q, want %q", tc.input, got, tc.want)
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
			if got := sanitizeParamName(tc.input); got != tc.want {
				t.Errorf("sanitizeParamName(%q) = %q, want %q", tc.input, got, tc.want)
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
	_, _, err := readABIAndBytecode(cfg, "contract", "v1_0_0", evmInputConfig{
		ABIBasePath:      t.TempDir(),
		BytecodeBasePath: t.TempDir(),
	})
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

	t.Run("nested tuple component needs big.Int", func(t *testing.T) {
		t.Parallel()
		info := &contractInfo{
			Functions: map[string]*functionInfo{
				"Foo": {
					Name: "Foo",
					Parameters: []parameterInfo{
						{
							Name:       "metadata",
							GoType:     "RootMetadata",
							IsStruct:   true,
							StructName: "RootMetadata",
							Components: []parameterInfo{
								{Name: "chainId", GoType: "*big.Int"},
							},
						},
					},
				},
			},
			FunctionOrder: []string{"Foo"},
		}
		if !checkNeedsBigInt(info) {
			t.Error("expected true for nested tuple component using *big.Int")
		}
	})
}
