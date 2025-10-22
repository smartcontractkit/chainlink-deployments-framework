package analyzer

import (
	"fmt"
	"reflect"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/block-vision/sui-go-sdk/models"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
)

func toNamedDescriptors(decodedOp mcmssdk.DecodedOperation) ([]NamedDescriptor, error) {
	args := decodedOp.Args()
	keys := decodedOp.Keys()
	if len(keys) != len(args) {
		return nil, fmt.Errorf("mismatched keys and arguments length: %d keys, %d arguments", len(keys), len(args))
	}
	namedArgs := make([]NamedDescriptor, len(args))
	for i := range args {
		namedArgs[i] = NamedDescriptor{
			Name:  keys[i],
			Value: getDescriptor(args[i]),
		}
	}

	return namedArgs, nil
}

func getDescriptor(argument any) Descriptor {
	var value Descriptor

	switch arg := argument.(type) {
	// Pretty-print byte arrays and addresses
	case []byte:
		value = BytesDescriptor{Value: arg}
	case aptos.AccountAddress:
		value = AddressDescriptor{Value: arg.StringLong()}
	case *aptos.AccountAddress:
		value = AddressDescriptor{Value: arg.StringLong()}
	case models.SuiAddress:
		value = AddressDescriptor{Value: string(arg)}
	default:
		//nolint:exhaustive // default case covers everything else
		switch reflect.TypeOf(arg).Kind() {
		// If the descriptor is a slice or array, iterate over every element individually
		case reflect.Array, reflect.Slice:
			array := ArrayDescriptor{}
			v := reflect.ValueOf(arg)
			for i := range v.Len() {
				array.Elements = append(array.Elements, getDescriptor(v.Index(i).Interface()))
			}
			value = array
		default:
			// Simply print the descriptor as-is
			value = SimpleDescriptor{Value: fmt.Sprintf("%v", arg)}
		}
	}

	return value
}
