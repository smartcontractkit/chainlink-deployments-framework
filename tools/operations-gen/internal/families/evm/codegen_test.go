package evm_test

import (
	"testing"

	"github.com/stretchr/testify/require"

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
			require.Equal(t, tc.want, evm.ToSnakeCase(tc.input), "ToSnakeCase(%q)", tc.input)
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
		require.True(t, evm.ChecksNeedsBigInt(info))
	})

	t.Run("return param needs big.Int", func(t *testing.T) {
		t.Parallel()
		info := &evm.ContractInfo{
			Functions: map[string]*evm.FunctionInfo{
				"Foo": {Name: "Foo", ReturnParams: []evm.ParameterInfo{{GoType: "*big.Int"}}},
			},
			FunctionOrder: []string{"Foo"},
		}
		require.True(t, evm.ChecksNeedsBigInt(info))
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
		require.True(t, evm.ChecksNeedsBigInt(info), "expected true for constructor uint256 param")
	})

	t.Run("no big.Int", func(t *testing.T) {
		t.Parallel()
		info := &evm.ContractInfo{
			Functions:     map[string]*evm.FunctionInfo{"Foo": makeFuncInfo("common.Address")},
			FunctionOrder: []string{"Foo"},
		}
		require.False(t, evm.ChecksNeedsBigInt(info))
	})

	t.Run("tuple parameter with *big.Int component does not need local big.Int", func(t *testing.T) {
		t.Parallel()
		// Tuple parameters are emitted as gobindings.<Name> references, so the
		// generated file never declares a local *big.Int field on behalf of a
		// nested tuple component. ChecksNeedsBigInt must therefore only look
		// at the top-level GoType of each parameter.
		info := &evm.ContractInfo{
			Functions: map[string]*evm.FunctionInfo{
				"Foo": {
					Name: "Foo",
					Parameters: []evm.ParameterInfo{
						{
							Name:       "metadata",
							GoType:     "gobindings.RootMetadata",
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
		require.False(t, evm.ChecksNeedsBigInt(info), "tuple components live in gobindings, no local *big.Int field")
	})
}
