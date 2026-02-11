package evm

import (
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"
)

// ConvertParam converts a decoded parameter value into a typed Go binding struct.
func ConvertParam[T any](param types.DecodedParameter) (T, error) {
	converted := abi.ConvertType(param.Value(), new(T))

	result, ok := converted.(*T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("ConvertParam[%T]: abi.ConvertType returned %T", zero, converted)
	}

	return *result, nil
}

// ConvertParamSlice converts a decoded parameter value into a typed slice.
func ConvertParamSlice[T any](param types.DecodedParameter) ([]T, error) {
	rv := reflect.ValueOf(param.Value())
	if rv.Kind() == reflect.Interface || rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Slice {
		return nil, fmt.Errorf("ConvertParamSlice: expected slice, got %T", param.Value())
	}

	result := make([]T, rv.Len())
	for i := range rv.Len() {
		converted := abi.ConvertType(rv.Index(i).Interface(), new(T))

		typed, ok := converted.(*T)
		if !ok {
			return nil, fmt.Errorf("ConvertParamSlice: element %d: abi.ConvertType returned %T", i, converted)
		}

		result[i] = *typed
	}

	return result, nil
}
