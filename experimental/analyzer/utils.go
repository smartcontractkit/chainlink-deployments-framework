package analyzer

import (
	"fmt"
	"reflect"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/block-vision/sui-go-sdk/models"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"

	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/proposalutils"
)

func toNamedArguments(decodedOp mcmssdk.DecodedOperation) ([]proposalutils.NamedArgument, error) {
	args := decodedOp.Args()
	keys := decodedOp.Keys()
	if len(keys) != len(args) {
		return nil, fmt.Errorf("mismatched keys and arguments length: %d keys, %d arguments", len(keys), len(args))
	}
	namedArgs := make([]proposalutils.NamedArgument, len(args))
	for i := range args {
		namedArgs[i] = proposalutils.NamedArgument{
			Name:  keys[i],
			Value: getArgument(args[i]),
		}
	}

	return namedArgs, nil
}

func getArgument(argument any) proposalutils.Argument {
	var value proposalutils.Argument

	switch arg := argument.(type) {
	// Pretty-print byte arrays and addresses
	case []byte:
		value = proposalutils.BytesArgument{Value: arg}
	case aptos.AccountAddress:
		value = proposalutils.AddressArgument{Value: arg.StringLong()}
	case *aptos.AccountAddress:
		value = proposalutils.AddressArgument{Value: arg.StringLong()}
	case models.SuiAddress:
		value = proposalutils.AddressArgument{Value: string(arg)}
	default:
		//nolint:exhaustive // default case covers everything else
		switch reflect.TypeOf(arg).Kind() {
		// If the argument is a slice or array, iterate over every element individually
		case reflect.Array, reflect.Slice:
			array := proposalutils.ArrayArgument{}
			v := reflect.ValueOf(arg)
			for i := range v.Len() {
				array.Elements = append(array.Elements, getArgument(v.Index(i).Interface()))
			}
			value = array
		default:
			// Simply print the argument as-is
			value = proposalutils.SimpleArgument{Value: fmt.Sprintf("%v", arg)}
		}
	}

	return value
}
