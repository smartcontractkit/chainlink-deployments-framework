package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

const (
	// anyType is the fallback Go type for unknown source types.
	anyType = "any"
)

// nameOverrides provides special-case naming for specific EVM contracts
// where the default snake_case conversion produces unexpected results.
var nameOverrides = map[string]string{
	"OnRamp":  "onramp",
	"OffRamp": "offramp",
}

// evmTypeMap maps Solidity types to their Go equivalents.
var evmTypeMap = map[string]string{
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

// ---- EVM contract config (YAML schema owned by evmHandler) ----

// evmContractConfig is the EVM-specific contract configuration decoded from YAML.
type evmContractConfig struct {
	Name         string              `yaml:"contract_name"`
	Version      string              `yaml:"version"`
	VersionPath  string              `yaml:"version_path,omitempty"`  // Optional: override folder path derived from version
	PackageName  string              `yaml:"package_name,omitempty"`  // Optional: override package name
	ABIFile      string              `yaml:"abi_file,omitempty"`      // Optional: override ABI file name
	NoDeployment bool                `yaml:"no_deployment,omitempty"` // Optional: skip bytecode and deploy operation
	Functions    []evmFunctionConfig `yaml:"functions"`
}

// evmFunctionConfig selects a contract function and assigns its access control.
type evmFunctionConfig struct {
	Name   string `yaml:"name"`
	Access string `yaml:"access,omitempty"` // "owner" or "public"
}

// ---- Intermediate representation ----

// contractInfo holds all parsed information about a contract needed for code generation.
type contractInfo struct {
	Name          string
	Version       string
	PackageName   string
	OutputPath    string
	ABI           string
	Bytecode      string
	NoDeployment  bool
	Constructor   *functionInfo
	Functions     map[string]*functionInfo
	FunctionOrder []string
	StructDefs    map[string]*structDef
}

type structDef struct {
	Name   string
	Fields []parameterInfo
}

type functionInfo struct {
	Name            string
	StateMutability string
	Parameters      []parameterInfo
	ReturnParams    []parameterInfo
	IsWrite         bool
	CallMethod      string // Method name, with numeric suffix for overloaded functions
	HasOnlyOwner    bool
}

type parameterInfo struct {
	Name         string
	SolidityType string
	GoType       string
	IsStruct     bool
	StructName   string
	Components   []parameterInfo
}

// ---- Template data (EVM-specific) ----

type templateData struct {
	PackageName       string
	PackageNameHyphen string
	ContractType      string
	Version           string
	ABI               string
	Bytecode          string
	NeedsBigInt       bool
	HasWriteOps       bool
	NoDeployment      bool
	Constructor       *constructorData
	StructDefs        []structDefData
	ArgStructs        []argStructData
	Operations        []operationData
	ContractMethods   []contractMethodData
}

type constructorData struct {
	Parameters []parameterData
}

type structDefData struct {
	Name   string
	Fields []parameterData
}

type argStructData struct {
	Name   string
	Fields []parameterData
}

type parameterData struct {
	GoName string
	GoType string
}

type operationData struct {
	Name          string
	MethodName    string
	OpName        string
	ArgsType      string
	CallArgs      string
	IsWrite       bool
	AccessControl string // Only for writes
	ReturnType    string // Only for reads
}

type writeOpData struct {
	Name          string
	MethodName    string
	OpName        string
	ArgsType      string
	CallArgs      string
	AccessControl string
}

type readOpData struct {
	Name       string
	MethodName string
	OpName     string
	ArgsType   string
	ReturnType string
	CallArgs   string
}

type contractMethodData struct {
	Name       string
	MethodName string
	Params     string
	Returns    string
	MethodBody string
}

// ---- evmHandler ----

// evmHandler implements ChainFamilyHandler for EVM (Solidity/go-ethereum) chains.
type evmHandler struct{}

// Generate decodes each YAML node as an evmContractConfig, extracts contract info,
// and writes a generated operations file for each contract.
func (h evmHandler) Generate(config Config, tmpl *template.Template) error {
	for _, node := range config.Contracts.Content {
		if node == nil {
			continue
		}
		var cfg evmContractConfig
		if err := node.Decode(&cfg); err != nil {
			return fmt.Errorf("failed to decode EVM contract config: %w", err)
		}

		info, err := extractContractInfo(cfg, config.Input, config.Output)
		if err != nil {
			return fmt.Errorf("error extracting info for %s: %w", cfg.Name, err)
		}

		if err := generateOperationsFile(info, tmpl); err != nil {
			return fmt.Errorf("error generating file for %s: %w", cfg.Name, err)
		}

		fmt.Printf("✓ Generated operations for %s at %s\n", info.Name, info.OutputPath)
	}

	return nil
}

// ---- Extraction ----

func extractContractInfo(cfg evmContractConfig, input InputConfig, output OutputConfig) (*contractInfo, error) {
	if cfg.Name == "" || cfg.Version == "" {
		return nil, errors.New("contract_name and version are required")
	}

	packageName := cfg.PackageName
	if packageName == "" {
		packageName = toSnakeCase(cfg.Name)
	}
	versionPath := versionToPath(cfg.Version)
	if cfg.VersionPath != "" {
		versionPath = cfg.VersionPath
	}

	abiString, bytecode, err := readABIAndBytecode(cfg, packageName, versionPath, input.BasePath)
	if err != nil {
		return nil, err
	}

	abiEntries, err := parseABIEntries(abiString)
	if err != nil {
		return nil, err
	}

	info := &contractInfo{
		Name:         cfg.Name,
		Version:      cfg.Version,
		PackageName:  packageName,
		OutputPath:   filepath.Join(output.BasePath, versionPath, "operations", packageName, packageName+".go"),
		ABI:          abiString,
		Bytecode:     bytecode,
		NoDeployment: cfg.NoDeployment,
		Functions:    make(map[string]*functionInfo),
		StructDefs:   make(map[string]*structDef),
	}

	extractConstructor(info, abiEntries, evmTypeMap)

	if err := extractFunctions(info, cfg.Functions, abiEntries, evmTypeMap); err != nil {
		return nil, err
	}

	collectAllStructDefs(info)

	return info, nil
}

func collectAllStructDefs(info *contractInfo) {
	if info.Constructor != nil {
		collectStructDefs(info.Constructor.Parameters, info.StructDefs)
	}
	for _, fi := range info.Functions {
		collectStructDefs(fi.Parameters, info.StructDefs)
		collectStructDefs(fi.ReturnParams, info.StructDefs)

		if !fi.IsWrite && len(fi.ReturnParams) > 1 {
			structName := multiReturnStructName(fi.Name)
			if _, exists := info.StructDefs[structName]; !exists {
				info.StructDefs[structName] = &structDef{
					Name:   structName,
					Fields: fi.ReturnParams,
				}
			}
		}
	}
}

func collectStructDefs(params []parameterInfo, structDefs map[string]*structDef) {
	for _, param := range params {
		if param.IsStruct && param.StructName != "" {
			if _, exists := structDefs[param.StructName]; !exists {
				structDefs[param.StructName] = &structDef{
					Name:   param.StructName,
					Fields: param.Components,
				}
			}
			collectStructDefs(param.Components, structDefs)
		}
	}
}

// ---- Code generation ----

func generateOperationsFile(info *contractInfo, tmpl *template.Template) error {
	data := prepareTemplateData(info)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("template execution error: %w", err)
	}

	return writeGoFile(info.OutputPath, buf.Bytes())
}

func prepareTemplateData(info *contractInfo) templateData {
	data := templateData{
		PackageName:       info.PackageName,
		PackageNameHyphen: toKebabCase(info.PackageName),
		ContractType:      info.Name,
		Version:           info.Version,
		ABI:               info.ABI,
		Bytecode:          info.Bytecode,
		NeedsBigInt:       checkNeedsBigInt(info),
		NoDeployment:      info.NoDeployment,
	}

	if info.Constructor != nil {
		data.Constructor = &constructorData{
			Parameters: prepareParameters(info.Constructor.Parameters),
		}
	}

	for _, name := range info.FunctionOrder {
		fi := info.Functions[name]
		data.ContractMethods = append(data.ContractMethods, prepareContractMethod(fi, fi.IsWrite))

		if fi.IsWrite {
			data.HasWriteOps = true
			wo := prepareWriteOp(fi)
			data.Operations = append(data.Operations, operationData{
				Name:          wo.Name,
				MethodName:    wo.MethodName,
				OpName:        wo.OpName,
				ArgsType:      wo.ArgsType,
				CallArgs:      wo.CallArgs,
				IsWrite:       true,
				AccessControl: wo.AccessControl,
			})
			if len(fi.Parameters) > 1 {
				data.ArgStructs = append(data.ArgStructs, argStructData{
					Name:   fi.Name + "Args",
					Fields: prepareParameters(fi.Parameters),
				})
			}
		} else {
			ro := prepareReadOp(fi)
			data.Operations = append(data.Operations, operationData{
				Name:       ro.Name,
				MethodName: ro.MethodName,
				OpName:     ro.OpName,
				ArgsType:   ro.ArgsType,
				CallArgs:   ro.CallArgs,
				IsWrite:    false,
				ReturnType: ro.ReturnType,
			})
			if len(fi.Parameters) > 1 {
				data.ArgStructs = append(data.ArgStructs, argStructData{
					Name:   fi.Name + "Args",
					Fields: prepareParameters(fi.Parameters),
				})
			}
		}
	}

	structNames := make([]string, 0, len(info.StructDefs))
	for name := range info.StructDefs {
		structNames = append(structNames, name)
	}
	sort.Strings(structNames)
	for _, name := range structNames {
		sd := info.StructDefs[name]
		data.StructDefs = append(data.StructDefs, structDefData{
			Name:   sd.Name,
			Fields: prepareParameters(sd.Fields),
		})
	}

	return data
}

func prepareParameters(params []parameterInfo) []parameterData {
	result := make([]parameterData, 0, len(params))
	for i, param := range params {
		name := capitalize(param.Name)
		if name == "" {
			name = fmt.Sprintf("Field%d", i)
		}
		result = append(result, parameterData{
			GoName: name,
			GoType: param.GoType,
		})
	}

	return result
}

// buildCallArgs builds the argsType and callArgs strings for an operation.
func buildCallArgs(fi *functionInfo, argsPrefix string) (argsType string, callArgs string) {
	if len(fi.Parameters) == 0 {
		return "struct{}", ""
	}

	if len(fi.Parameters) == 1 {
		return fi.Parameters[0].GoType, ", " + argsPrefix
	}

	argsType = fi.Name + "Args"
	var callArgsList []string
	for i, p := range fi.Parameters {
		fieldName := capitalize(p.Name)
		if fieldName == "" {
			fieldName = fmt.Sprintf("Field%d", i)
		}
		callArgsList = append(callArgsList, argsPrefix+"."+fieldName)
	}
	callArgs = ", " + strings.Join(callArgsList, ", ")

	return argsType, callArgs
}

func prepareWriteOp(fi *functionInfo) writeOpData {
	argsType, callArgs := buildCallArgs(fi, "args")

	accessControl := "AllCallersAllowed"
	if fi.HasOnlyOwner {
		accessControl = "OnlyOwner"
	}

	return writeOpData{
		Name:          fi.Name,
		MethodName:    fi.CallMethod,
		OpName:        toKebabCase(fi.Name),
		ArgsType:      argsType,
		CallArgs:      callArgs,
		AccessControl: accessControl,
	}
}

func prepareReadOp(fi *functionInfo) readOpData {
	argsType, callArgs := buildCallArgs(fi, "args")

	returnType := anyType
	if len(fi.ReturnParams) == 1 {
		returnType = fi.ReturnParams[0].GoType
	} else if len(fi.ReturnParams) > 1 {
		returnType = multiReturnStructName(fi.Name)
	}

	return readOpData{
		Name:       fi.Name,
		MethodName: fi.CallMethod,
		OpName:     toKebabCase(fi.Name),
		ArgsType:   argsType,
		ReturnType: returnType,
		CallArgs:   callArgs,
	}
}

func multiReturnStructName(funcName string) string {
	return funcName + "Result"
}

// prepareContractMethod builds the contractMethodData for a single contract function,
// generating go-ethereum–specific method signatures and bodies.
func prepareContractMethod(fi *functionInfo, isWrite bool) contractMethodData {
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
		var paramsSb490 strings.Builder
		for _, p := range fi.Parameters {
			paramName := p.Name
			if len(paramName) > 0 {
				paramName = strings.ToLower(paramName[:1]) + paramName[1:]
			}
			if paramName == "" {
				paramName = fmt.Sprintf("arg%d", len(methodArgs))
			}
			paramsSb490.WriteString(fmt.Sprintf(", %s %s", paramName, p.GoType))
			methodArgs = append(methodArgs, paramName)
		}
		params += paramsSb490.String()
	}

	returns := "(*types.Transaction, error)"
	returnType := anyType
	if !isWrite {
		if len(fi.ReturnParams) == 1 {
			returnType = fi.ReturnParams[0].GoType
		} else if len(fi.ReturnParams) > 1 {
			returnType = multiReturnStructName(fi.Name)
		}
		returns = fmt.Sprintf("(%s, error)", returnType)
	}

	var methodBody string
	if isWrite {
		methodBody = buildWriteMethodBody(fi, methodArgs)
	} else {
		methodBody = buildReadMethodBody(fi, methodArgs, returnType)
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
func buildWriteMethodBody(fi *functionInfo, methodArgs []string) string {
	if len(methodArgs) > 0 {
		return fmt.Sprintf("return c.contract.Transact(opts, \"%s\", %s)",
			fi.CallMethod, strings.Join(methodArgs, ", "))
	}

	return fmt.Sprintf("return c.contract.Transact(opts, \"%s\")", fi.CallMethod)
}

// buildReadMethodBody generates the body of a read (call) method.
func buildReadMethodBody(fi *functionInfo, methodArgs []string, returnType string) string {
	callArgsStr := ""
	if len(methodArgs) > 0 {
		callArgsStr = ", " + strings.Join(methodArgs, ", ")
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
func buildMultiReturnMethodBody(fi *functionInfo, callArgsStr, returnType string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "var out []any\n")
	fmt.Fprintf(&b, "\terr := c.contract.Call(opts, &out, \"%s\"%s)\n", fi.CallMethod, callArgsStr)
	fmt.Fprintf(&b, "\toutstruct := new(%s)\n", returnType)
	fmt.Fprintf(&b, "\tif err != nil {\n")
	fmt.Fprintf(&b, "\t\treturn *outstruct, err\n")
	fmt.Fprintf(&b, "\t}\n\n")
	for i, p := range fi.ReturnParams {
		fieldName := capitalize(p.Name)
		if fieldName == "" {
			fieldName = fmt.Sprintf("Field%d", i)
		}
		fmt.Fprintf(&b, "\toutstruct.%s = *abi.ConvertType(out[%d], new(%s)).(*%s)\n",
			fieldName, i, p.GoType, p.GoType)
	}
	fmt.Fprintf(&b, "\n\treturn *outstruct, nil")

	return b.String()
}

// ---- ABI parsing ----

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

// readABIAndBytecode reads the ABI JSON and (optionally) bytecode for a contract
// from the EVM-conventional directory layout:
//
//	{basePath}/abi/{versionPath}/{name}.json
//	{basePath}/bytecode/{versionPath}/{name}.bin
func readABIAndBytecode(
	cfg evmContractConfig,
	packageName,
	versionPath,
	basePath string) (abiString string, bytecode string, err error) {
	var abiFileName string
	if cfg.ABIFile != "" {
		if !strings.HasSuffix(cfg.ABIFile, ".json") {
			return "", "", fmt.Errorf("abi_file %q must end with .json", cfg.ABIFile)
		}
		abiFileName = cfg.ABIFile
	} else {
		abiFileName = packageName + ".json"
	}

	abiPath := filepath.Join(basePath, "abi", versionPath, abiFileName)
	abiBytes, err := os.ReadFile(abiPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read ABI from %s: %w", abiPath, err)
	}

	if cfg.NoDeployment {
		return string(abiBytes), "", nil
	}

	bytecodeName := strings.TrimSuffix(abiFileName, ".json") + ".bin"
	bytecodePath := filepath.Join(basePath, "bytecode", versionPath, bytecodeName)
	bytecodeBytes, err := os.ReadFile(bytecodePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read bytecode from %s: %w", bytecodePath, err)
	}

	return string(abiBytes), strings.TrimSpace(string(bytecodeBytes)), nil
}

func extractConstructor(info *contractInfo, abiEntries []ABIEntry, typeMap map[string]string) {
	for _, entry := range abiEntries {
		if entry.Type == "constructor" {
			info.Constructor = parseABIFunction(entry, info.PackageName, typeMap)
			break
		}
	}
}

func extractFunctions(info *contractInfo, funcConfigs []evmFunctionConfig, abiEntries []ABIEntry, typeMap map[string]string) error {
	for _, funcCfg := range funcConfigs {
		funcInfos := findFunctionInABI(abiEntries, funcCfg.Name, info.PackageName, typeMap)
		if funcInfos == nil {
			return fmt.Errorf("function %s not found in ABI", funcCfg.Name)
		}

		for _, fi := range funcInfos {
			switch funcCfg.Access {
			case "owner":
				fi.HasOnlyOwner = true
			case "public":
				fi.HasOnlyOwner = false
			default:
				return fmt.Errorf("unknown access control '%s' for function %s (use 'owner' or 'public')",
					funcCfg.Access, funcCfg.Name)
			}

			info.Functions[fi.Name] = fi
			info.FunctionOrder = append(info.FunctionOrder, fi.Name)
		}
	}

	return nil
}

// findFunctionInABI finds all overloads of a function by name and returns functionInfo
// for each, following Geth's overload naming convention.
func findFunctionInABI(entries []ABIEntry, funcName string, packageName string, typeMap map[string]string) []*functionInfo {
	var candidates []ABIEntry
	for _, entry := range entries {
		if entry.Type == "function" && strings.EqualFold(entry.Name, funcName) {
			candidates = append(candidates, entry)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	var funcInfos []*functionInfo
	for i, candidate := range candidates {
		fi := parseABIFunction(candidate, packageName, typeMap)

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

// parseABIFunction converts a Solidity ABI function entry into a functionInfo.
// IsWrite is determined by stateMutability: anything other than "view" or "pure" is a write.
func parseABIFunction(entry ABIEntry, packageName string, typeMap map[string]string) *functionInfo {
	fi := &functionInfo{
		Name:            capitalize(entry.Name),
		StateMutability: entry.StateMutability,
		CallMethod:      entry.Name,
		IsWrite:         entry.StateMutability != "view" && entry.StateMutability != "pure",
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
func parseABIParam(param ABIParam, packageName string, typeMap map[string]string) parameterInfo {
	goType := solidityToGoType(param.Type, typeMap)

	pi := parameterInfo{
		Name:         param.Name,
		SolidityType: param.Type,
		GoType:       goType,
	}

	if strings.HasPrefix(param.Type, "tuple") {
		structName := extractStructName(param.InternalType)
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

// solidityToGoType maps a Solidity type string to its Go equivalent using typeMap.
func solidityToGoType(solidityType string, typeMap map[string]string) string {
	baseType := strings.TrimSuffix(solidityType, "[]")
	if goType, ok := typeMap[baseType]; ok {
		if strings.HasSuffix(solidityType, "[]") {
			return "[]" + goType
		}

		return goType
	}

	if strings.HasPrefix(baseType, "tuple") {
		return anyType
	}

	return anyType
}

// extractStructName parses the Go struct name from a Solidity ABI internalType field.
// e.g. "struct IOnRamp.DestChainConfig" → "DestChainConfig"
func extractStructName(internalType string) string {
	if internalType == "" {
		return ""
	}

	parts := strings.Split(internalType, ".")
	structName := parts[len(parts)-1]

	return strings.TrimSuffix(structName, "[]")
}

// parseABIEntries unmarshals a raw ABI JSON string into a slice of ABIEntry.
func parseABIEntries(abiString string) ([]ABIEntry, error) {
	var entries []ABIEntry
	if err := json.Unmarshal([]byte(abiString), &entries); err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	return entries, nil
}

// checkNeedsBigInt reports whether any parameter in the contract uses *big.Int,
// which requires importing "math/big" in the generated file.
func checkNeedsBigInt(info *contractInfo) bool {
	check := func(params []parameterInfo) bool {
		for _, p := range params {
			if strings.Contains(p.GoType, "*big.Int") {
				return true
			}
		}

		return false
	}

	for _, fi := range info.Functions {
		if check(fi.Parameters) || check(fi.ReturnParams) {
			return true
		}
	}

	if info.Constructor != nil && check(info.Constructor.Parameters) {
		return true
	}

	for _, sd := range info.StructDefs {
		if check(sd.Fields) {
			return true
		}
	}

	return false
}

// ---- Naming utilities (EVM-specific due to nameOverrides) ----

func toSnakeCase(s string) string {
	if override, ok := nameOverrides[s]; ok {
		return override
	}

	var result []rune
	runes := []rune(s)
	for i := range runes {
		r := runes[i]
		if i > 0 && r >= 'A' && r <= 'Z' {
			prevLower := runes[i-1] >= 'a' && runes[i-1] <= 'z'
			nextLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'
			if prevLower || nextLower {
				result = append(result, '_')
			}
		}
		result = append(result, r)
	}

	return strings.ToLower(string(result))
}

func toKebabCase(s string) string {
	return strings.ReplaceAll(toSnakeCase(s), "_", "-")
}
