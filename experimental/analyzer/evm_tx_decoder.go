package analyzer

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// FieldAnalyzer is an extension point of proposal decoding.
// You can implement your own FieldAnalyzer which returns your own FieldValue instance.
type FieldAnalyzer func(argName string, argAbi *abi.Type, argVal any, analyzers []FieldAnalyzer) FieldValue

type EVMTxCallDecoder struct {
	Analyzers []FieldAnalyzer
}

func NewTxCallDecoder(extraAnalyzers []FieldAnalyzer) *EVMTxCallDecoder {
	analyzers := make([]FieldAnalyzer, 0, len(extraAnalyzers)+DefaultAnalyzersCount)
	analyzers = append(analyzers, extraAnalyzers...)
	analyzers = append(analyzers, BytesAndAddressAnalyzer)
	analyzers = append(analyzers, ChainSelectorAnalyzer)

	return &EVMTxCallDecoder{Analyzers: analyzers}
}

// Decode decodes calldata into a DecodedCall.
// NamedField.Value holds the display-oriented FieldValue tree.
// NamedField.RawValue holds the original ABI-decoded Go value.
func (p *EVMTxCallDecoder) Decode(address string, contractABI *abi.ABI, data []byte) (*DecodedCall, error) {
	if len(data) < MinDataLengthForMethodID {
		return nil, fmt.Errorf("data with value %s is too short", hexutil.Encode(data))
	}

	methodID, methodData := data[:4], data[4:]

	method, err := contractABI.MethodById(methodID)
	if err != nil {
		return nil, err
	}

	args := make(map[string]any)

	err = method.Inputs.UnpackIntoMap(args, methodData)
	if err != nil {
		return nil, err
	}

	inputs := make([]NamedField, len(method.Inputs))
	for i, input := range method.Inputs {
		arg, ok := args[input.Name]
		if !ok {
			return nil, fmt.Errorf("missing argument '%s'", input.Name)
		}

		inputs[i] = NamedField{
			Name:     input.Name,
			RawValue: arg,
			Value:    p.decodeArg(input.Name, &input.Type, arg),
		}
	}

	return &DecodedCall{
		Address: address,
		Method:  method.String(),
		Inputs:  inputs,
		Outputs: []NamedField{},
	}, nil
}

// decodeArg decodes a single argument using the provided ABI type and value.
func (p *EVMTxCallDecoder) decodeArg(argName string, argAbi *abi.Type, argVal any) FieldValue {
	if len(p.Analyzers) > 0 {
		for _, analyzer := range p.Analyzers {
			arg := analyzer(argName, argAbi, argVal, p.Analyzers)
			if arg != nil {
				return arg
			}
		}
	}
	// Struct analyzer
	if argAbi.T == abi.TupleTy {
		return p.decodeStruct(argAbi, argVal)
	}
	// Array analyzer
	if argAbi.T == abi.SliceTy || argAbi.T == abi.ArrayTy {
		return p.decodeArray(argName, argAbi, argVal)
	}
	// Fallback
	return SimpleField{Value: fmt.Sprintf("%v", argVal)}
}

// decodeStruct decodes a struct argument using the provided ABI type and value.
func (p *EVMTxCallDecoder) decodeStruct(argAbi *abi.Type, argVal any) StructField {
	argTyp := argAbi.GetType()
	fields := make([]NamedField, argTyp.NumField())
	for i := range argTyp.NumField() {
		if !argTyp.Field(i).IsExported() {
			continue
		}
		argFieldName := argTyp.Field(i).Name
		argFieldAbi := argAbi.TupleElems[i]
		argFieldTyp := reflect.ValueOf(argVal).FieldByName(argFieldName)
		rawVal := argFieldTyp.Interface()
		argument := p.decodeArg(argFieldName, argFieldAbi, rawVal)
		fields[i] = NamedField{
			Name:  argFieldName,
			Value: argument,
		}
	}

	return StructField{
		Fields: fields,
	}
}

// decodeArray decodes an array argument using the provided ABI type and value.
func (p *EVMTxCallDecoder) decodeArray(argName string, argAbi *abi.Type, argVal any) ArrayField {
	argTyp := reflect.ValueOf(argVal)
	elements := make([]FieldValue, argTyp.Len())
	for i := range argTyp.Len() {
		argElemTyp := argTyp.Index(i)
		argument := p.decodeArg(argName, argAbi.Elem, argElemTyp.Interface())
		elements[i] = argument
	}

	return ArrayField{
		Elements: elements,
	}
}

var chainSelectorRegex = regexp.MustCompile(`[cC]hain([sS]el)?.*$`)

// BytesAndAddressAnalyzer is an EVM-specific analyzer that handles bytes and address types.
func BytesAndAddressAnalyzer(_ string, argAbi *abi.Type, argVal any, _ []FieldAnalyzer) FieldValue {
	if argAbi.T == abi.FixedBytesTy || argAbi.T == abi.BytesTy || argAbi.T == abi.AddressTy {
		argArrTyp := reflect.ValueOf(argVal)
		argArr := make([]byte, argArrTyp.Len())
		for i := range argArrTyp.Len() {
			argArr[i] = byte(argArrTyp.Index(i).Uint())
		}
		if argAbi.T == abi.AddressTy {
			return AddressField{Value: common.BytesToAddress(argArr).Hex()}
		}

		return BytesField{Value: argArr}
	}

	return nil
}

// ChainSelectorAnalyzer is an EVM-specific analyzer that handles chain selector parameters.
func ChainSelectorAnalyzer(argName string, argAbi *abi.Type, argVal any, _ []FieldAnalyzer) FieldValue {
	if argAbi.GetType().Kind() == reflect.Uint64 && chainSelectorRegex.MatchString(argName) {
		return ChainSelectorField{Value: argVal.(uint64)}
	}

	return nil
}
