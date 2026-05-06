package evm

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"golang.org/x/tools/go/packages"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/core"
)

const (
	anyType    = "any"
	uint64Type = "uint64"
	int64Type  = "int64"
)

var compactStructInternalTypePattern = regexp.MustCompile(`struct([A-Za-z_][A-Za-z0-9_]*\.)`)

// AbiToGoType converts a go-ethereum abi.Type to its Go type string.
//
// Primitive types map to their Go equivalents. Tuple types return
// "gobindings.<TupleRawName>" — the geth-generated binding struct for the
// same Solidity struct. The operations file always imports the gobindings
// package as "gobindings", so referencing these types directly avoids
// declaring a second, incompatible copy in the generated file.
// Slice and array types are handled recursively and therefore compose
// naturally (e.g. []gobindings.Foo, [3]gobindings.Foo).
//
// Integer widths follow abigen's canonical mapping
// (accounts/abi/type.go:reflectIntType): only 8, 16, 32 and 64 map to native
// Go integer types — every other width (24, 40, 48, 56, 72 … 256) becomes
// *big.Int so the generated signatures exactly match the gobindings emitted
// by geth's abigen.
func AbiToGoType(t abi.Type) string {
	switch t.T {
	case abi.UintTy:
		switch t.Size {
		case 8:
			return "uint8"
		case 16:
			return "uint16"
		case 32:
			return "uint32"
		case 64:
			return uint64Type
		default:
			return "*big.Int"
		}
	case abi.IntTy:
		switch t.Size {
		case 8:
			return "int8"
		case 16:
			return "int16"
		case 32:
			return "int32"
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
		// Anonymous tuples (e.g. an inline `(uint256,address)` return) have no
		// TupleRawName and therefore no corresponding gobindings struct. Emitting
		// "gobindings." would produce an invalid identifier and break compilation;
		// fall back to `any` so generated code at least compiles. Slices/arrays of
		// anonymous tuples degrade naturally via the recursive SliceTy/ArrayTy
		// cases above (e.g. "[]any").
		if t.TupleRawName == "" {
			return anyType
		}

		return "gobindings." + t.TupleRawName
	}

	return anyType
}

// ReadABI reads the ABI from the generated gobindings package by extracting the exported <ContractName>MetaData.ABI.
func ReadABI(
	cfg EvmContractConfig,
) (*abi.ABI, error) {
	if cfg.GobindingsPackage == "" {
		return nil, fmt.Errorf("gobindings_package is required for contract %q", cfg.Name)
	}

	abiStr, err := readABIFromGobindings(cfg.GobindingsPackage, cfg.Name, cfg.ConfigDir)
	if err != nil {
		return nil, err
	}

	parsedABI, err := abi.JSON(strings.NewReader(NormalizeStructInternalTypes(*abiStr)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI from %sMetaData in gobindings package %q: %w", cfg.Name, cfg.GobindingsPackage, err)
	}

	return &parsedABI, nil
}

// Some abigen-produced bindings encode struct internal types without the
// space go-ethereum expects ("structContract.Type" instead of
// "struct Contract.Type"). Normalize that shape so abi.JSON can populate
// TupleRawName and the generated ops can reuse the gobindings struct types.
func NormalizeStructInternalTypes(abiString string) string {
	return compactStructInternalTypePattern.ReplaceAllString(abiString, "struct $1")
}

func readABIFromGobindings(pkgPath string, contractName string, loadDir string) (*string, error) {
	loadCfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles,
	}
	if loadDir != "" {
		loadCfg.Dir = loadDir
	}

	pkgs, err := packages.Load(loadCfg, pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load gobindings package %q: %w", pkgPath, err)
	}

	if len(pkgs) != 1 {
		return nil, fmt.Errorf("expected one package for %q, got %d", pkgPath, len(pkgs))
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("failed to load gobindings package %q: %v", pkgPath, pkg.Errors)
	}
	if len(pkg.GoFiles) == 0 {
		return nil, fmt.Errorf("gobindings package %q has no Go files", pkgPath)
	}

	metadataVar := contractName + "MetaData"
	fset := token.NewFileSet()
	for _, filePath := range pkg.GoFiles {
		file, err := parser.ParseFile(fset, filePath, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse gobindings file %q: %w", filePath, err)
		}

		abiStr, err := extractAbiFromFile(file, metadataVar)
		if err != nil {
			return nil, fmt.Errorf("extract %s from gobindings package %q: %w", metadataVar, pkgPath, err)
		}
		if abiStr != nil {
			return abiStr, nil
		}
	}

	return nil, fmt.Errorf("metadata %q not found in gobindings package %q", metadataVar, pkgPath)
}

func extractAbiFromFile(file *ast.File, metadataVar string) (abiStr *string, err error) {
	ast.Inspect(file, func(node ast.Node) bool {
		if err != nil || abiStr != nil {
			return false
		}

		valueSpec, ok := node.(*ast.ValueSpec)
		if !ok {
			return true
		}

		for i, name := range valueSpec.Names {
			if name.Name != metadataVar || i >= len(valueSpec.Values) {
				continue
			}

			var value string
			value, err = metadataABIValue(valueSpec.Values[i])
			if err != nil {
				err = fmt.Errorf("read %s ABI: %w", metadataVar, err)
			} else {
				abiStr = &value
			}

			return false
		}

		return true
	})

	return abiStr, nil
}

func metadataABIValue(expr ast.Expr) (string, error) {
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		expr = unary.X
	}

	comp, ok := expr.(*ast.CompositeLit)
	if !ok {
		return "", errors.New("metadata value is not a composite literal")
	}

	for _, elt := range comp.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}

		if key.Name != "ABI" {
			continue
		}

		return stringLiteralValue(kv.Value)
	}

	return "", errors.New("metadata does not contain ABI")
}

func stringLiteralValue(expr ast.Expr) (string, error) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", errors.New("expected string literal")
	}

	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", fmt.Errorf("unquote string literal: %w", err)
	}

	return value, nil
}

// FindFunctionInABI returns all methods in parsedABI whose RawName matches
// funcName (case-insensitive), sorted by their disambiguated Name for
// deterministic output.
func FindFunctionInABI(parsedABI *abi.ABI, funcName string) []abi.Method {
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
		Name:                 core.Capitalize(m.Name),
		StateMutability:      m.StateMutability,
		CallMethod:           m.Name,
		AllReturnParamsNamed: len(m.Outputs) > 0,
		IsWrite:              m.StateMutability != "view" && m.StateMutability != "pure",
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
			fi.AllReturnParamsNamed = false
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
