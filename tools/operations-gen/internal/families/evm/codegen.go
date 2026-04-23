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
	PackageName       string
	PackageNameHyphen string
	ContractType      string
	Version           string
	ABI               string
	Bytecode          string
	NeedsBigInt       bool
	HasWriteOps       bool
	OmitDeploy        bool
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
	GoName  string
	GoType  string
	JSONTag string // ABI parameter name; may be a synthesized placeholder (e.g. "ret0") for unnamed outputs
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
		ABI:               info.ABI,
		Bytecode:          info.Bytecode,
		NeedsBigInt:       ChecksNeedsBigInt(info),
		OmitDeploy:        info.OmitDeploy,
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
func ChecksNeedsBigInt(info *ContractInfo) bool {
	var check func(params []ParameterInfo) bool
	check = func(params []ParameterInfo) bool {
		for _, p := range params {
			if strings.Contains(p.GoType, "*big.Int") {
				return true
			}
			if len(p.Components) > 0 && check(p.Components) {
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

func prepareParameters(params []ParameterInfo) []parameterData {
	result := make([]parameterData, 0, len(params))
	for i, param := range params {
		result = append(result, parameterData{
			GoName:  fieldNameOrIndex(param.Name, i),
			GoType:  param.GoType,
			JSONTag: param.Name,
		})
	}

	return result
}

func prepareWriteOp(fi *FunctionInfo) operationData {
	argsType, callArgs := buildCallArgs(fi)

	accessControl := accessControlAllCallers
	if fi.HasOnlyOwner {
		accessControl = accessControlOnlyOwner
	}

	return operationData{
		Name:          fi.Name,
		MethodName:    fi.CallMethod,
		OpName:        toKebabCase(fi.Name),
		ArgsType:      argsType,
		CallArgs:      callArgs,
		IsWrite:       true,
		AccessControl: accessControl,
	}
}

func prepareReadOp(fi *FunctionInfo) operationData {
	argsType, callArgs := buildCallArgs(fi)

	return operationData{
		Name:       fi.Name,
		MethodName: fi.CallMethod,
		OpName:     toKebabCase(fi.Name),
		ArgsType:   argsType,
		ReturnType: resolveReturnType(fi),
		CallArgs:   callArgs,
		IsWrite:    false,
	}
}

// buildCallArgs builds the argsType and callArgs strings for an operation.
func buildCallArgs(fi *FunctionInfo) (argsType string, callArgs string) {
	if len(fi.Parameters) == 0 {
		return emptyReturnType, ""
	}

	if len(fi.Parameters) == 1 {
		return fi.Parameters[0].GoType, ", args"
	}

	argsType = fi.Name + "Args"
	callArgsList := make([]string, 0, len(fi.Parameters))
	for i, p := range fi.Parameters {
		callArgsList = append(callArgsList, "args."+fieldNameOrIndex(p.Name, i))
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
