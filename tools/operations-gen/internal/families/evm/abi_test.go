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
		{abi.Type{T: abi.UintTy, Size: 256}, "*big.Int"},
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
		// intermediate uint sizes
		{abi.Type{T: abi.UintTy, Size: 40}, "uint64"},
		{abi.Type{T: abi.UintTy, Size: 48}, "uint64"},
		{abi.Type{T: abi.UintTy, Size: 56}, "uint64"},
		// tuple → any
		{abi.Type{T: abi.TupleTy, TupleRawName: "TestStruct"}, "TestStruct"},
		{abi.Type{T: abi.ArrayTy, Elem: &abi.Type{T: abi.TupleTy, TupleRawName: "TestStruct"}}, "[]TestStruct"},
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
