package evm_test

import (
	"testing"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/families/evm"
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
			if got := evm.ToSnakeCase(tc.input); got != tc.want {
				t.Errorf("ToSnakeCase(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCheckNeedsBigInt(t *testing.T) {
	t.Parallel()
	makeFuncInfo := func(goType string) *evm.FunctionInfo {
		return &evm.FunctionInfo{
			Name:       "Foo",
			Parameters: []evm.ParameterInfo{{GoType: goType}},
		}
	}

	t.Run("parameter needs big.Int", func(t *testing.T) {
		t.Parallel()
		info := &evm.ContractInfo{
			Functions:     map[string]*evm.FunctionInfo{"Foo": makeFuncInfo("*big.Int")},
			FunctionOrder: []string{"Foo"},
		}
		if !evm.ChecksNeedsBigInt(info) {
			t.Error("expected true")
		}
	})

	t.Run("return param needs big.Int", func(t *testing.T) {
		t.Parallel()
		info := &evm.ContractInfo{
			Functions: map[string]*evm.FunctionInfo{
				"Foo": {Name: "Foo", ReturnParams: []evm.ParameterInfo{{GoType: "*big.Int"}}},
			},
			FunctionOrder: []string{"Foo"},
		}
		if !evm.ChecksNeedsBigInt(info) {
			t.Error("expected true")
		}
	})

	t.Run("constructor param needs big.Int", func(t *testing.T) {
		t.Parallel()
		info := &evm.ContractInfo{
			Constructor: &evm.FunctionInfo{
				Parameters: []evm.ParameterInfo{{GoType: "*big.Int"}},
			},
			Functions:     map[string]*evm.FunctionInfo{},
			FunctionOrder: []string{},
		}
		if !evm.ChecksNeedsBigInt(info) {
			t.Error("expected true for constructor uint256 param")
		}
	})

	t.Run("no big.Int", func(t *testing.T) {
		t.Parallel()
		info := &evm.ContractInfo{
			Functions:     map[string]*evm.FunctionInfo{"Foo": makeFuncInfo("common.Address")},
			FunctionOrder: []string{"Foo"},
		}
		if evm.ChecksNeedsBigInt(info) {
			t.Error("expected false")
		}
	})

	t.Run("nested tuple component needs big.Int", func(t *testing.T) {
		t.Parallel()
		info := &evm.ContractInfo{
			Functions: map[string]*evm.FunctionInfo{
				"Foo": {
					Name: "Foo",
					Parameters: []evm.ParameterInfo{
						{
							Name:       "metadata",
							GoType:     "RootMetadata",
							IsStruct:   true,
							StructName: "RootMetadata",
							Components: []evm.ParameterInfo{
								{Name: "chainId", GoType: "*big.Int"},
							},
						},
					},
				},
			},
			FunctionOrder: []string{"Foo"},
		}
		if !evm.ChecksNeedsBigInt(info) {
			t.Error("expected true for nested tuple component using *big.Int")
		}
	})
}
