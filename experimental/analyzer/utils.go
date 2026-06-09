package analyzer

import (
	"fmt"
	"reflect"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/block-vision/sui-go-sdk/models"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
)

func toNamedFields(decodedOp mcmssdk.DecodedOperation) ([]NamedField, error) {
	args := decodedOp.Args()
	keys := decodedOp.Keys()
	if len(keys) != len(args) {
		return nil, fmt.Errorf("mismatched keys and arguments length: %d keys, %d arguments", len(keys), len(args))
	}
	namedArgs := make([]NamedField, len(args))
	for i := range args {
		namedArgs[i] = NamedField{
			Name:     keys[i],
			Value:    getFieldValue(args[i]),
			RawValue: args[i],
		}
	}

	return namedArgs, nil
}

func getFieldValue(argument any) FieldValue {
	var value FieldValue

	switch arg := argument.(type) {
	case nil:
		value = SimpleField{Value: "null"}
	case []byte:
		value = BytesField{Value: arg}
	case aptos.AccountAddress:
		value = AddressField{Value: arg.StringLong()}
	case *aptos.AccountAddress:
		value = AddressField{Value: arg.StringLong()}
	case models.SuiAddress:
		value = AddressField{Value: string(arg)}
	default:
		//nolint:exhaustive // default case covers everything else
		switch reflect.TypeOf(arg).Kind() {
		case reflect.Array, reflect.Slice:
			array := ArrayField{}
			v := reflect.ValueOf(arg)
			for i := range v.Len() {
				array.Elements = append(array.Elements, getFieldValue(v.Index(i).Interface()))
			}
			value = array
		default:
			value = SimpleField{Value: fmt.Sprintf("%v", arg)}
		}
	}

	return value
}
