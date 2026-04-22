package evm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/core"
)

// EvmTypeMap maps Solidity types to their Go equivalents.
var EvmTypeMap = map[string]string{
	"address": "common.Address",
	"string":  "string",
	"bool":    "bool",
	"bytes":   "[]byte",
	"bytes32": "[32]byte",
	"bytes16": "[16]byte",
	"bytes4":  "[4]byte",
	"uint8":   "uint8",
	"uint16":  "uint16",
	"uint32":  "uint32",
	"uint40":  "uint64",
	"uint48":  "uint64",
	"uint56":  "uint64",
	"uint64":  "uint64",
	"uint96":  "*big.Int",
	"uint128": "*big.Int",
	"uint160": "*big.Int",
	"uint192": "*big.Int",
	"uint224": "*big.Int",
	"uint256": "*big.Int",
	"int8":    "int8",
	"int16":   "int16",
	"int32":   "int32",
	"int64":   "int64",
	"int96":   "*big.Int",
	"int128":  "*big.Int",
	"int160":  "*big.Int",
	"int192":  "*big.Int",
	"int224":  "*big.Int",
	"int256":  "*big.Int",
}

// ABIEntry represents a single entry in a Solidity contract ABI JSON.
type ABIEntry struct {
	Type            string     `json:"type"`
	Name            string     `json:"name"`
	Inputs          []ABIParam `json:"inputs"`
	Outputs         []ABIParam `json:"outputs"`
	StateMutability string     `json:"stateMutability"`
}

// ABIParam represents a parameter within an ABI entry.
type ABIParam struct {
	Name         string     `json:"name"`
	Type         string     `json:"type"`
	InternalType string     `json:"internalType"`
	Components   []ABIParam `json:"components"`
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

// FindFunctionInABI finds all overloads of a function by name and returns FunctionInfo
// for each, following Geth's overload naming convention.
func FindFunctionInABI(entries []ABIEntry, funcName string, packageName string, typeMap map[string]string) []*FunctionInfo {
	var candidates []ABIEntry
	for _, entry := range entries {
		if entry.Type == abiTypeFunction && strings.EqualFold(entry.Name, funcName) {
			candidates = append(candidates, entry)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	var funcInfos []*FunctionInfo
	for i, candidate := range candidates {
		fi := ParseABIFunction(candidate, packageName, typeMap)

		// Follow Geth's overload naming convention:
		// First: no suffix, second: "0", third: "1", etc.
		if len(candidates) > 1 && i > 0 {
			suffix := strconv.Itoa(i - 1)
			fi.Name = fi.Name + suffix
			fi.CallMethod = fi.CallMethod + suffix
		}

		funcInfos = append(funcInfos, fi)
	}

	return funcInfos
}

// ParseABIFunction converts a Solidity ABI function entry into a FunctionInfo.
// IsWrite is determined by stateMutability: anything other than "view" or "pure" is a write.
func ParseABIFunction(entry ABIEntry, packageName string, typeMap map[string]string) *FunctionInfo {
	fi := &FunctionInfo{
		Name:       core.Capitalize(entry.Name),
		CallMethod: entry.Name,
		IsWrite:    entry.StateMutability != stateMutabilityView && entry.StateMutability != stateMutabilityPure,
	}

	for i, input := range entry.Inputs {
		p := parseABIParam(input, packageName, typeMap)
		if p.Name == "" {
			p.Name = fmt.Sprintf("arg%d", i)
		}
		fi.Parameters = append(fi.Parameters, p)
	}

	for i, output := range entry.Outputs {
		p := parseABIParam(output, packageName, typeMap)
		if p.Name == "" {
			p.Name = fmt.Sprintf("ret%d", i)
		}
		fi.ReturnParams = append(fi.ReturnParams, p)
	}

	return fi
}

//nolint:unparam
func parseABIParam(param ABIParam, packageName string, typeMap map[string]string) ParameterInfo {
	goType := SolidityToGoType(param.Type, typeMap)

	pi := ParameterInfo{
		Name:         param.Name,
		SolidityType: param.Type,
		GoType:       goType,
	}

	if strings.HasPrefix(param.Type, "tuple") {
		structName := ExtractStructName(param.InternalType)
		if structName != "" {
			pi.IsStruct = true
			pi.StructName = structName

			if strings.HasSuffix(param.Type, "[]") {
				pi.GoType = "[]" + structName
			} else {
				pi.GoType = structName
			}

			for _, comp := range param.Components {
				pi.Components = append(pi.Components, parseABIParam(comp, packageName, typeMap))
			}
		}
	}

	return pi
}

// SolidityToGoType maps a Solidity type string to its Go equivalent using typeMap.
func SolidityToGoType(solidityType string, typeMap map[string]string) string {
	// Array: uint8[] → []uint8, uint8[32] → [32]uint8
	if i := strings.LastIndexByte(solidityType, '['); i != -1 {
		// Guard malformed type strings like "[" or "uint8[" to avoid slicing panics.
		if !strings.HasSuffix(solidityType, "]") || i+1 > len(solidityType)-1 {
			return anyType
		}
		sizeStr := solidityType[i+1 : len(solidityType)-1]
		_, numErr := strconv.Atoi(sizeStr)
		if sizeStr == "" || numErr == nil {
			inner := SolidityToGoType(solidityType[:i], typeMap)
			if inner != anyType {
				return "[" + sizeStr + "]" + inner
			}

			return anyType
		}
	}
	if goType, ok := typeMap[solidityType]; ok {
		return goType
	}

	return anyType
}

// ExtractStructName parses the Go struct name from a Solidity ABI internalType field.
// e.g. "struct IOnRamp.DestChainConfig" → "DestChainConfig"
// e.g. "struct MyStruct" → "MyStruct"  (no module prefix)
// Returns "" for anonymous tuples ("tuple", "tuple[]") so callers fall back to any.
func ExtractStructName(internalType string) string {
	if internalType == "" {
		return ""
	}

	// Bare "tuple" / "tuple[]" have no named struct — callers should fall back to any.
	if strings.HasPrefix(internalType, "tuple") {
		return ""
	}

	normalized := strings.TrimPrefix(internalType, "struct ")
	normalized = strings.TrimSuffix(normalized, "[]")
	parts := strings.Split(normalized, ".")

	return parts[len(parts)-1]
}

// parseABIEntries unmarshals a raw ABI JSON string into a slice of ABIEntry.
func parseABIEntries(abiString string) ([]ABIEntry, error) {
	var entries []ABIEntry
	if err := json.Unmarshal([]byte(abiString), &entries); err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	return entries, nil
}

// trimUnderscores strips all leading underscores from s.
func trimUnderscores(s string) string {
	return strings.TrimLeft(s, "_")
}

// SanitizeFieldName strips leading underscores and capitalizes the result,
// producing a valid exported Go identifier for struct fields.
// Returns "" when the result would start with a digit (e.g. "_1" → ""); callers fall back to "Field%d".
// e.g. "_to" → "To", "_value" → "Value", "balance" → "Balance"
func SanitizeFieldName(name string) string {
	trimmed := trimUnderscores(name)
	if len(trimmed) == 0 || (trimmed[0] >= '0' && trimmed[0] <= '9') {
		return ""
	}

	return core.Capitalize(trimmed)
}

// SanitizeParamName strips leading underscores and lowercases the first rune,
// producing a valid unexported Go identifier for method parameters.
// Returns "" when the result would start with a digit (e.g. "_1" → ""); callers fall back to "arg%d".
// e.g. "_to" → "to", "_value" → "value"
func SanitizeParamName(name string) string {
	name = trimUnderscores(name)
	if len(name) == 0 || (name[0] >= '0' && name[0] <= '9') {
		return ""
	}

	return strings.ToLower(name[:1]) + name[1:]
}

// fieldNameOrIndex returns the sanitized exported field name for a struct field,
// or "Field{i}" when the sanitized result would be empty (e.g. numeric-only names).
func fieldNameOrIndex(name string, i int) string {
	if n := SanitizeFieldName(name); n != "" {
		return n
	}

	return fmt.Sprintf("Field%d", i)
}

// validatePathSegment rejects values that could traverse outside the output base path.
// Absolute paths and any cleaned path containing ".." or a path separator are rejected.
func validatePathSegment(field, value string) error {
	if filepath.IsAbs(value) {
		return fmt.Errorf("%s must not be an absolute path: %q", field, value)
	}
	cleaned := filepath.Clean(value)
	if strings.Contains(cleaned, "..") || strings.ContainsRune(cleaned, filepath.Separator) {
		return fmt.Errorf("%s must not contain path separators or '..': %q", field, value)
	}

	return nil
}

func resolveReturnType(fi *FunctionInfo) string {
	if len(fi.ReturnParams) == 1 {
		return fi.ReturnParams[0].GoType
	} else if len(fi.ReturnParams) > 1 {
		return multiReturnStructName(fi.Name)
	}

	return emptyReturnType
}

// prepareContractMethod builds the contractMethodData for a single contract function,
// generating go-ethereum–specific method signatures and bodies.
func prepareContractMethod(fi *FunctionInfo, isWrite bool) contractMethodData {
	optsType := "*bind.CallOpts"
	if isWrite {
		optsType = "*bind.TransactOpts"
	}

	params := "opts " + optsType
	var methodArgs []string

	if len(fi.Parameters) == 1 {
		params += ", args " + fi.Parameters[0].GoType
		methodArgs = []string{"args"}
	} else if len(fi.Parameters) > 1 {
		var sb strings.Builder
		for _, p := range fi.Parameters {
			paramName := SanitizeParamName(p.Name)
			if paramName == "" {
				paramName = fmt.Sprintf("arg%d", len(methodArgs))
			}
			fmt.Fprintf(&sb, ", %s %s", paramName, p.GoType)
			methodArgs = append(methodArgs, paramName)
		}
		params += sb.String()
	}

	returns := "(*types.Transaction, error)"
	if !isWrite {
		returns = fmt.Sprintf("(%s, error)", resolveReturnType(fi))
	}

	var methodBody string
	if isWrite {
		methodBody = buildWriteMethodBody(fi.CallMethod, methodArgs)
	} else {
		methodBody = buildReadMethodBody(fi, methodArgs, resolveReturnType(fi))
	}

	return contractMethodData{
		Name:       fi.Name,
		MethodName: fi.CallMethod,
		Params:     params,
		Returns:    returns,
		MethodBody: methodBody,
	}
}

// buildWriteMethodBody generates the body of a write (transact) method.
func buildWriteMethodBody(callMethod string, methodArgs []string) string {
	if len(methodArgs) > 0 {
		return fmt.Sprintf("return c.contract.Transact(opts, \"%s\", %s)",
			callMethod, strings.Join(methodArgs, ", "))
	}

	return fmt.Sprintf("return c.contract.Transact(opts, \"%s\")", callMethod)
}

// buildReadMethodBody generates the body of a read (call) method.
func buildReadMethodBody(fi *FunctionInfo, methodArgs []string, returnType string) string {
	callArgsStr := ""
	if len(methodArgs) > 0 {
		callArgsStr = ", " + strings.Join(methodArgs, ", ")
	}
	if len(fi.ReturnParams) == 0 {
		return fmt.Sprintf(
			`err := c.contract.Call(opts, nil, "%s"%s)
	return struct{}{}, err`,
			fi.CallMethod, callArgsStr,
		)
	}
	if len(fi.ReturnParams) > 1 {
		return buildMultiReturnMethodBody(fi, callArgsStr, returnType)
	}

	return fmt.Sprintf(
		`var out []any
	err := c.contract.Call(opts, &out, "%s"%s)
	if err != nil {
		var zero %s
		return zero, err
	}
	return *abi.ConvertType(out[0], new(%s)).(*%s), nil`,
		fi.CallMethod, callArgsStr, returnType, returnType, returnType,
	)
}

// buildMultiReturnMethodBody generates the body for a read method with multiple return values,
// packing them into a result struct.
func buildMultiReturnMethodBody(fi *FunctionInfo, callArgsStr, returnType string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "var out []any\n")
	fmt.Fprintf(&b, "\terr := c.contract.Call(opts, &out, \"%s\"%s)\n", fi.CallMethod, callArgsStr)
	fmt.Fprintf(&b, "\toutstruct := new(%s)\n", returnType)
	fmt.Fprintf(&b, "\tif err != nil {\n")
	fmt.Fprintf(&b, "\t\treturn *outstruct, err\n")
	fmt.Fprintf(&b, "\t}\n\n")
	for i, p := range fi.ReturnParams {
		fmt.Fprintf(&b, "\toutstruct.%s = *abi.ConvertType(out[%d], new(%s)).(*%s)\n",
			fieldNameOrIndex(p.Name, i), i, p.GoType, p.GoType)
	}
	fmt.Fprintf(&b, "\n\treturn *outstruct, nil")

	return b.String()
}
