package evm

import (
	"bytes"
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/core"
)

// ---- Template data types ----------------------------------------------------
// These types are the input to operations.tmpl and mirror the structure of the
// generated Go file.

// TemplateData is the root context passed to operations.tmpl.
type TemplateData struct {
	PackageName       string
	PackageNameHyphen string
	ContractType      string
	// GobindingsImport is the import path of the gobindings package, aliased
	// as "geth_bindings" in the generated file.
	GobindingsImport string
	Version          string
	NeedsBigInt      bool
	HasWriteOps      bool
	NoDeployment     bool
	Constructor      *ConstructorData
	StructDefs       []StructDefData
	ArgStructs       []ArgStructData
	Operations       []OperationData
}

// ConstructorData holds the parsed constructor parameters for the Deploy operation.
type ConstructorData struct {
	Parameters []ParameterData
}

// StructDefData describes one struct type to be declared in the generated file.
type StructDefData struct {
	Name   string
	Fields []ParameterData
}

// ArgStructData describes a multi-parameter args struct (e.g. ApplyAssetParamsUpdatesArgs).
type ArgStructData struct {
	Name   string
	Fields []ParameterData
}

// ParameterData is a single field/parameter ready for template rendering.
type ParameterData struct {
	GoName string
	GoType string
}

// OperationData describes one CLD operation (read or write) in the generated file.
type OperationData struct {
	Name          string
	MethodName    string
	OpName        string // kebab-case operation name used as the CLD operation identifier
	ArgsType      string
	CallArgs      string
	IsWrite       bool
	AccessControl string // "public" | "owner" | role"
	RoleGoLiteral string // Go literal for the role bytes32, e.g. [32]byte{0x12, …}
	ReturnType    string // reads only
	// ReturnFields is non-empty for read operations with multiple return values.
	// The template uses it to pack individual return values into a synthetic result struct.
	ReturnFields []ParameterData
}

// ---- Code generation --------------------------------------------------------

// generateOperationsFile renders the operations template with data derived from
// info and writes the gofmt-formatted result to info.OutputPath.
func generateOperationsFile(info *ContractInfo, tmpl *template.Template) error {
	data := prepareTemplateData(info)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("template execution error: %w", err)
	}

	return core.WriteGoFile(info.OutputPath, buf.Bytes())
}

// prepareTemplateData converts a ContractInfo into the flat TemplateData
// structure consumed by operations.tmpl.
func prepareTemplateData(info *ContractInfo) TemplateData {
	data := TemplateData{
		PackageName:       info.PackageName,
		PackageNameHyphen: toKebabCase(info.PackageName),
		ContractType:      info.Name,
		Version:           info.Version,
		GobindingsImport:  info.GobindingsPackage,
		NeedsBigInt:       CheckNeedsBigInt(info),
		NoDeployment:      info.NoDeployment,
	}

	if info.Constructor != nil {
		data.Constructor = &ConstructorData{
			Parameters: prepareParameters(info.Constructor.Parameters),
		}
	}

	for _, name := range info.FunctionOrder {
		funcInfo := info.Functions[name]

		if funcInfo.IsWrite {
			data.HasWriteOps = true
			data.Operations = append(data.Operations, prepareWriteOp(funcInfo))
		} else {
			data.Operations = append(data.Operations, prepareReadOp(funcInfo))
		}

		if len(funcInfo.Parameters) > 1 {
			data.ArgStructs = append(data.ArgStructs, ArgStructData{
				Name:   funcInfo.Name + "Args",
				Fields: prepareParameters(funcInfo.Parameters),
			})
		}
	}

	// StructDefs: emit remaining defs (gobindings types have already been removed
	// by remapGobindingsTypes and are referenced directly as geth_bindings.XxxType).
	structNames := make([]string, 0, len(info.StructDefs))
	for name := range info.StructDefs {
		structNames = append(structNames, name)
	}
	sort.Strings(structNames)
	for _, name := range structNames {
		structDef := info.StructDefs[name]
		data.StructDefs = append(data.StructDefs, StructDefData{
			Name:   structDef.Name,
			Fields: prepareParameters(structDef.Fields),
		})
	}

	return data
}

// CheckNeedsBigInt returns true if any parameter in the contract uses *big.Int,
// which means the generated file must import "math/big".
func CheckNeedsBigInt(info *ContractInfo) bool {
	check := func(params []ParameterInfo) bool {
		for _, p := range params {
			if strings.Contains(p.GoType, "*big.Int") {
				return true
			}
		}

		return false
	}

	for _, funcInfo := range info.Functions {
		if check(funcInfo.Parameters) || check(funcInfo.ReturnParams) {
			return true
		}
	}
	if info.Constructor != nil && check(info.Constructor.Parameters) {
		return true
	}

	return false
}

// prepareParameters converts a slice of ParameterInfo to ParameterData,
// capitalising names and providing fallback names for anonymous fields.
func prepareParameters(params []ParameterInfo) []ParameterData {
	result := make([]ParameterData, 0, len(params))
	for i, param := range params {
		name := capitalize(param.Name)
		if name == "" {
			name = fmt.Sprintf("Field%d", i)
		}
		result = append(result, ParameterData{GoName: name, GoType: param.GoType})
	}

	return result
}

// prepareWriteOp builds the OperationData for a state-changing function.
func prepareWriteOp(funcInfo *FunctionInfo) OperationData {
	argsType, callArgs := buildCallArgs(funcInfo, "args")

	role := ""
	if funcInfo.AccessControl == "role" {
		role = formatRoleGoLiteral(funcInfo.Role)
	}

	return OperationData{
		Name:          funcInfo.Name,
		MethodName:    funcInfo.CallMethod,
		OpName:        toKebabCase(funcInfo.Name),
		ArgsType:      argsType,
		CallArgs:      callArgs,
		IsWrite:       true,
		AccessControl: funcInfo.AccessControl,
		RoleGoLiteral: role,
	}
}

// prepareReadOp builds the OperationData for a view/pure function.
// For multi-return functions, ReturnFields is populated so the template can
// pack individual return values into a synthetic result struct.
func prepareReadOp(funcInfo *FunctionInfo) OperationData {
	argsType, callArgs := buildCallArgs(funcInfo, "args")

	returnType := anyType
	var returnFields []ParameterData
	if len(funcInfo.ReturnParams) == 1 {
		returnType = funcInfo.ReturnParams[0].GoType
	} else if len(funcInfo.ReturnParams) > 1 {
		returnType = multiReturnStructName(funcInfo.Name)
		returnFields = prepareParameters(funcInfo.ReturnParams)
	}

	return OperationData{
		Name:         funcInfo.Name,
		MethodName:   funcInfo.CallMethod,
		OpName:       toKebabCase(funcInfo.Name),
		ArgsType:     argsType,
		CallArgs:     callArgs,
		IsWrite:      false,
		ReturnType:   returnType,
		ReturnFields: returnFields,
	}
}

// buildCallArgs determines the argsType and callArgs strings for a function.
// Zero params → struct{}, one param → the param's type passed directly as "args",
// multiple params → a generated XxxArgs struct with individual field accessors.
func buildCallArgs(funcInfo *FunctionInfo, argsPrefix string) (argsType string, callArgs string) {
	if len(funcInfo.Parameters) == 0 {
		return "struct{}", ""
	}
	if len(funcInfo.Parameters) == 1 {
		return funcInfo.Parameters[0].GoType, ", " + argsPrefix
	}

	argsType = funcInfo.Name + "Args"
	var callArgsList []string
	for i, p := range funcInfo.Parameters {
		fieldName := capitalize(p.Name)
		if fieldName == "" {
			fieldName = fmt.Sprintf("Field%d", i)
		}
		callArgsList = append(callArgsList, argsPrefix+"."+fieldName)
	}
	callArgs = ", " + strings.Join(callArgsList, ", ")

	return argsType, callArgs
}

// multiReturnStructName returns the name of the synthetic result struct for a
// function with multiple return values (e.g. "GetFoo" → "GetFooResult").
func multiReturnStructName(funcName string) string {
	return funcName + "Result"
}

// ---- Naming helpers ---------------------------------------------------------

// ToSnakeCase converts a PascalCase contract name to snake_case.
func ToSnakeCase(s string) string {
	var result []rune
	runes := []rune(s)
	for i, r := range runes {
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

// toKebabCase converts a snake_case string to kebab-case
// (used for CLD operation name identifiers).
func toKebabCase(s string) string {
	return strings.ReplaceAll(ToSnakeCase(s), "_", "-")
}

// capitalize upper-cases the first character of s.
func capitalize(s string) string {
	if s == "" {
		return ""
	}

	return strings.ToUpper(s[:1]) + s[1:]
}

// versionToPath converts a semver string to a directory path segment,
// e.g. "1.0.0" → "v1_0_0".
func versionToPath(version string) string {
	return "v" + strings.ReplaceAll(version, ".", "_")
}
