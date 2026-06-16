package evm

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/core"
)

// ---- Template data (EVM-specific) ----

type templateData struct {
	PackageName                 string
	PackageNameHyphen           string
	ContractType                string
	Version                     string
	GobindingsImport            string
	ZkSyncBytecodeSymbol        string
	ZkSyncBytecodeImport        string
	ZkSyncBytecodeUseGobindings bool
	NeedsBigInt                 bool
	HasWriteOps                 bool
	OmitDeploy                  bool
	Constructor                 *constructorData
	StructDefs                  []structDefData
	ArgStructs                  []argStructData
	Operations                  []OperationData
	ContractMethods             []contractMethodData
}

type constructorData struct {
	Parameters []ParameterData
}

type structDefData struct {
	Name   string
	Fields []ParameterData
}

type argStructData struct {
	Name   string
	Fields []ParameterData
}

type ParameterData struct {
	GoName  string
	GoType  string
	JSONTag string // ABI parameter name; may be a synthesized placeholder (e.g. "ret0") for unnamed outputs
}

type OperationData struct {
	Name          string
	MethodName    string
	OpName        string
	ArgsType      string
	CallArgs      string
	IsWrite       bool
	AccessControl string // Only for writes
	Role          string // Go literal for the role bytes32, e.g. [32]byte{0x12, …}
	ReturnType    string // Only for reads
	// ReturnFields is non-empty for read operations with multiple return values.
	// The template uses it to pack individual return values into a synthetic result struct.
	ReturnFields []ParameterData
}

type contractMethodData struct {
	Name       string
	MethodName string
	Params     string
	Returns    string
	MethodBody string
}

func generateOperationsFile(info *ContractInfo, tmpl *template.Template) error {
	data := prepareTemplateData(info)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("template execution error: %w", err)
	}

	return core.WriteGoFile(info.OutputPath, buf.Bytes())
}

func prepareTemplateData(info *ContractInfo) templateData {
	data := templateData{
		PackageName:       info.PackageName,
		PackageNameHyphen: toKebabCase(info.PackageName),
		ContractType:      info.Name,
		Version:           info.Version,
		GobindingsImport:  info.GobindingsPackage,
		NeedsBigInt:       ChecksNeedsBigInt(info),
		OmitDeploy:        info.OmitDeploy,
	}
	if info.ZkSync != nil {
		data.ZkSyncBytecodeSymbol = info.ZkSync.BytecodeSymbol
		data.ZkSyncBytecodeImport = info.ZkSync.BytecodePackage
		data.ZkSyncBytecodeUseGobindings = info.ZkSync.BytecodePackage == info.GobindingsPackage
	}

	if info.Constructor != nil {
		data.Constructor = &constructorData{
			Parameters: prepareParameters(info.Constructor.Parameters),
		}
	}

	for _, name := range info.FunctionOrder {
		fi := info.Functions[name]

		if fi.IsWrite {
			data.HasWriteOps = true
			data.Operations = append(data.Operations, prepareWriteOp(fi))
		} else {
			data.Operations = append(data.Operations, prepareReadOp(fi))
		}

		if len(fi.Parameters) > 1 {
			data.ArgStructs = append(data.ArgStructs, argStructData{
				Name:   fi.Name + "Args",
				Fields: prepareParameters(fi.Parameters),
			})
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

// ChecksNeedsBigInt reports whether any parameter in the contract uses *big.Int,
// which requires importing "math/big" in the generated file.
//
// The check only inspects the top-level GoType of each parameter: tuple
// (struct) parameters are emitted as references into the gobindings package,
// so their internal fields never surface as *big.Int in the generated file.
func ChecksNeedsBigInt(info *ContractInfo) bool {
	check := func(params []ParameterInfo) bool {
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

	return false
}

// prepareParameters converts a slice of ParameterInfo to ParameterData,
// capitalising names and providing fallback names for anonymous fields.
func prepareParameters(params []ParameterInfo) []ParameterData {
	result := make([]ParameterData, 0, len(params))
	for i, param := range params {
		name := SanitizeFieldName(param.Name)
		if name == "" {
			name = fmt.Sprintf("Field%d", i)
		}
		jsonTag := param.Name
		if jsonTag == "" {
			jsonTag = fmt.Sprintf("ret%d", i)
		}
		result = append(result, ParameterData{GoName: name, GoType: param.GoType, JSONTag: jsonTag})
	}

	return result
}

func prepareWriteOp(fi *FunctionInfo) OperationData {
	argsType, callArgs := buildCallArgs(fi)

	role := ""
	if fi.AccessControl == accessRole {
		role = FormatRoleGoLiteral(fi.Role)
	}

	return OperationData{
		Name:          fi.Name,
		MethodName:    fi.CallMethod,
		OpName:        toKebabCase(fi.Name),
		ArgsType:      argsType,
		CallArgs:      callArgs,
		IsWrite:       true,
		AccessControl: fi.AccessControl,
		Role:          role,
	}
}

// prepareReadOp builds the OperationData for a view/pure function.
// For multi-return functions, ReturnFields is populated so the template can
// pack individual return values into a synthetic result struct.
func prepareReadOp(funcInfo *FunctionInfo) OperationData {
	argsType, callArgs := buildCallArgs(funcInfo)

	returnType := anyType
	var returnFields []ParameterData
	if len(funcInfo.ReturnParams) == 1 {
		returnType = funcInfo.ReturnParams[0].GoType
	} else if len(funcInfo.ReturnParams) > 1 {
		if funcInfo.AllReturnParamsNamed {
			returnType = "gobindings." + funcInfo.Name
		} else {
			returnType = multiReturnStructName(funcInfo.Name)
			returnFields = prepareParameters(funcInfo.ReturnParams)
		}
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

// buildCallArgs builds the argsType and callArgs strings for an operation.
func buildCallArgs(fi *FunctionInfo) (argsType string, callArgs string) {
	if len(fi.Parameters) == 0 {
		return "struct{}", ""
	}
	if len(fi.Parameters) == 1 {
		return fi.Parameters[0].GoType, ", args"
	}

	argsType = fi.Name + "Args"
	var callArgsList []string
	for i, p := range fi.Parameters {
		fieldName := SanitizeFieldName(p.Name)
		if fieldName == "" {
			fieldName = fmt.Sprintf("Field%d", i)
		}
		callArgsList = append(callArgsList, "args."+fieldName)
	}
	callArgs = ", " + strings.Join(callArgsList, ", ")

	return argsType, callArgs
}

func multiReturnStructName(funcName string) string {
	return funcName + "Result"
}

// ---- Naming utilities ----

func ToSnakeCase(s string) string {
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
	return strings.ReplaceAll(ToSnakeCase(s), "_", "-")
}
