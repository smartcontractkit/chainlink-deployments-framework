package evm

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/core"
)

const (
	anyType     = "any"
	uint64Typee = "uint64"
	int64Type   = "int64"
)

// AbiToGoType converts a go-ethereum abi.Type to its Go type string.
//
// Primitive types map to their Go equivalents. Tuple types return
// TupleRawName — the geth binding struct name (e.g. "BaseAuctionAssetParams"),
// which is later prefixed with "geth_bindings." by remapGobindingsTypes.
// Slice and array types are handled recursively.
func AbiToGoType(t abi.Type) string {
	switch t.T {
	case abi.UintTy:
		switch t.Size {
		case 8:
			return "uint8"
		case 16:
			return "uint16"
		case 24:
			return "uint32"
		case 32:
			return "uint32"
		case 40:
			return uint64Typee
		case 48:
			return uint64Typee
		case 56:
			return uint64Typee
		case 64:
			return uint64Typee
		default:
			return "*big.Int"
		}
	case abi.IntTy:
		switch t.Size {
		case 8:
			return "int8"
		case 16:
			return "int16"
		case 24:
			return "int32"
		case 32:
			return "int32"
		case 40:
			return int64Type
		case 48:
			return int64Type
		case 56:
			return int64Type
		case 64:
			return int64Type
		default:
			return "*big.Int"
		}
	case abi.BoolTy:
		return "bool"
	case abi.StringTy:
		return "string"
	case abi.AddressTy:
		return "common.Address"
	case abi.BytesTy:
		return "[]byte"
	case abi.FixedBytesTy:
		return fmt.Sprintf("[%d]byte", t.Size)
	case abi.SliceTy:
		return "[]" + AbiToGoType(*t.Elem)
	case abi.ArrayTy:
		switch t.Size {
		case 0:
			return "[]" + AbiToGoType(*t.Elem)
		default:
			return fmt.Sprintf("[%d]%s", t.Size, AbiToGoType(*t.Elem))
		}
	case abi.TupleTy:
		return t.TupleRawName
	}

	return anyType
}

// ReadABIAndBytecode reads the ABI JSON and (optionally) bytecode for a contract
// from the configured input roots:
//
//	{input.ABIBasePath}/{versionPath}/{name}.json
//	{input.BytecodeBasePath}/{versionPath}/{name}.bin
func ReadABIAndBytecode(
	cfg EvmContractConfig,
	packageName,
	versionPath string,
	input EvmInputConfig) (abiString string, bytecode string, err error) {
	var abiFileName string
	if cfg.ABIFile != "" {
		if !strings.HasSuffix(cfg.ABIFile, ".json") {
			return "", "", fmt.Errorf("abi_file %q must end with .json", cfg.ABIFile)
		}
		abiFileName = cfg.ABIFile
	} else {
		abiFileName = packageName + ".json"
	}

	abiPath := filepath.Join(input.ABIBasePath, versionPath, abiFileName)
	abiBytes, err := os.ReadFile(abiPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read ABI from %s: %w", abiPath, err)
	}

	if cfg.OmitDeploy {
		return string(abiBytes), "", nil
	}

	bytecodeName := strings.TrimSuffix(abiFileName, ".json") + ".bin"
	bytecodePath := filepath.Join(input.BytecodeBasePath, versionPath, bytecodeName)
	bytecodeBytes, err := os.ReadFile(bytecodePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read bytecode from %s: %w", bytecodePath, err)
	}

	return string(abiBytes), strings.TrimSpace(string(bytecodeBytes)), nil
}

// FindFunctionInABI returns all methods in parsedABI whose RawName matches
// funcName (case-insensitive), sorted by their disambiguated Name for
// deterministic output.
func FindFunctionInABI(parsedABI abi.ABI, funcName string) []abi.Method {
	var matches []abi.Method
	for _, m := range parsedABI.Methods {
		if strings.EqualFold(m.RawName, funcName) {
			matches = append(matches, m)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Name < matches[j].Name
	})

	return matches
}

// paramInfoFromType converts a go-ethereum abi.Type into a ParameterInfo,
// recursively populating Components for tuple types so that StructDefs can
// be collected later.
func paramInfoFromType(name string, t abi.Type) ParameterInfo {
	info := ParameterInfo{
		Name:   name,
		GoType: AbiToGoType(t),
	}

	// Walk through slice/array wrappers to find the base type.
	base := &t
	for base.T == abi.SliceTy || base.T == abi.ArrayTy {
		base = base.Elem
	}

	if base.T == abi.TupleTy && base.TupleRawName != "" {
		info.IsStruct = true
		info.StructName = base.TupleRawName
		for i, elem := range base.TupleElems {
			fieldName := ""
			if i < len(base.TupleRawNames) {
				fieldName = base.TupleRawNames[i]
			}
			info.Components = append(info.Components, paramInfoFromType(fieldName, *elem))
		}
	}

	return info
}

// methodToFunctionInfo converts a go-ethereum abi.Method into a FunctionInfo.
// m.Name is the disambiguated method name (handles overloads, e.g. "curse0")
// and is used as both the Go method name key and the CallMethod string.
func methodToFunctionInfo(m abi.Method) *FunctionInfo {
	fi := &FunctionInfo{
		Name:            core.Capitalize(m.Name),
		StateMutability: m.StateMutability,
		CallMethod:      m.Name,
		IsWrite:         m.StateMutability != "view" && m.StateMutability != "pure",
	}
	for i, arg := range m.Inputs {
		p := paramInfoFromType(arg.Name, arg.Type)
		if p.Name == "" {
			p.Name = fmt.Sprintf("arg%d", i)
		}
		fi.Parameters = append(fi.Parameters, p)
	}
	for i, arg := range m.Outputs {
		p := paramInfoFromType(arg.Name, arg.Type)
		if p.Name == "" {
			p.Name = fmt.Sprintf("ret%d", i)
		}
		fi.ReturnParams = append(fi.ReturnParams, p)
	}

	return fi
}

// SanitizeFieldName strips leading underscores and capitalizes the result,
// producing a valid exported Go identifier for struct fields.
// Returns "" when the result would start with a digit (e.g. "_1" → ""); callers fall back to "Field%d".
// e.g. "_to" → "To", "_value" → "Value", "balance" → "Balance"
func SanitizeFieldName(name string) string {
	trimmed := strings.TrimLeft(name, "_")
	if len(trimmed) == 0 || (trimmed[0] >= '0' && trimmed[0] <= '9') {
		return ""
	}

	return core.Capitalize(trimmed)
}
