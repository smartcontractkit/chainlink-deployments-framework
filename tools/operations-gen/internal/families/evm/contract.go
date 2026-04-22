package evm

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const (
	accessOwner = "owner"
	accessRole  = "role"
)

// ContractInfo is the fully-resolved representation of a contract after reading
// its ABI and applying all config. It is the input to the code generator.
type ContractInfo struct {
	Name              string
	Version           string
	PackageName       string
	OutputPath        string
	ABI               string
	Bytecode          string
	NoDeployment      bool
	GobindingsPackage string
	Constructor       *FunctionInfo
	// Functions holds the selected functions in insertion order via FunctionOrder.
	Functions     map[string]*FunctionInfo
	FunctionOrder []string
	// StructDefs holds struct types referenced by selected functions that need
	// to be defined in the generated file. Gobindings-sourced structs are removed
	// by remapGobindingsTypes and referenced directly as geth_bindings.XxxType.
	StructDefs map[string]*StructDef
}

// StructDef describes a Solidity struct type that appears in a function signature.
type StructDef struct {
	Name   string
	Fields []ParameterInfo
	// InternalType is non-empty for structs that originate from the ABI (i.e. have
	// a TupleRawName). It equals the geth binding struct name (e.g.
	// "BaseAuctionAssetParams"). When set, remapGobindingsTypes will replace any
	// GoType references to this struct with "geth_bindings.<InternalType>".
	InternalType string
}

// FunctionInfo is the parsed, config-enriched representation of one ABI function.
type FunctionInfo struct {
	Name            string
	StateMutability string
	Parameters      []ParameterInfo
	ReturnParams    []ParameterInfo
	IsWrite         bool
	// CallMethod is the exact method name used in contract.Transact / contract.Call.
	// For overloaded functions this includes the numeric suffix (e.g. "curse0").
	CallMethod    string
	AccessControl string
	Role          [32]byte
}

// ParameterInfo describes one parameter (input or output) of a function.
type ParameterInfo struct {
	Name       string
	GoType     string // resolved Go type string, e.g. "*big.Int", "common.Address"
	IsStruct   bool
	StructName string // TupleRawName when IsStruct is true, e.g. "BaseAuctionAssetParams"
	Components []ParameterInfo
}

// extractContractInfo reads the ABI from the gobindings source, parses it with
// the go-ethereum ABI package, and returns a fully-resolved ContractInfo ready
// for code generation.
//
// moduleSearchDir is the directory from which findModuleRoot walks upward to
// locate go.mod (e.g. core.Config.ConfigDir). When empty, output.BasePath is used.
func extractContractInfo(
	cfg ContractConfig,
	output OutputConfig,
	moduleSearchDir string,
) (*ContractInfo, error) {
	if cfg.Name == "" || cfg.Version == "" {
		return nil, errors.New("contract_name and version are required")
	}
	if cfg.GobindingsPackage == "" {
		return nil, fmt.Errorf("contract %s: gobindings_package is required", cfg.Name)
	}

	packageName := cfg.PackageName
	if packageName == "" {
		packageName = ToSnakeCase(cfg.Name)
	}
	outputVersionPath := versionToPath(cfg.Version)
	if cfg.OutputVersionPath != "" {
		outputVersionPath = cfg.OutputVersionPath
	}

	searchDir := moduleSearchDir
	if searchDir == "" {
		searchDir = output.BasePath
	}

	abiString, bytecode, err := readABIFromGobindingsSource(
		cfg.GobindingsPackage,
		searchDir,
		cfg.NoDeployment,
	)
	if err != nil {
		return nil, err
	}

	parsedABI, err := abi.JSON(strings.NewReader(normalizeABIString(abiString)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI for %s: %w", cfg.Name, err)
	}

	info := &ContractInfo{
		Name:        cfg.Name,
		Version:     cfg.Version,
		PackageName: packageName,
		OutputPath: filepath.Join(
			output.BasePath,
			outputVersionPath,
			"operations",
			packageName,
			packageName+".go",
		),
		ABI:               abiString,
		Bytecode:          bytecode,
		NoDeployment:      cfg.NoDeployment,
		GobindingsPackage: cfg.GobindingsPackage,
		Functions:         make(map[string]*FunctionInfo),
		StructDefs:        make(map[string]*StructDef),
	}

	extractConstructor(info, parsedABI)

	if err := extractFunctions(info, cfg.Functions, parsedABI); err != nil {
		return nil, err
	}

	collectAllStructDefs(info)
	if info.GobindingsPackage != "" {
		remapGobindingsTypes(info)
	}

	return info, nil
}

// extractConstructor populates info.Constructor when the ABI defines a
// constructor with one or more inputs.
func extractConstructor(info *ContractInfo, parsedABI abi.ABI) {
	if len(parsedABI.Constructor.Inputs) == 0 {
		return
	}
	fi := &FunctionInfo{
		Name:            "constructor",
		StateMutability: parsedABI.Constructor.StateMutability,
		IsWrite:         true,
	}
	for i, arg := range parsedABI.Constructor.Inputs {
		p := paramInfoFromType(arg.Name, arg.Type)
		if p.Name == "" {
			p.Name = fmt.Sprintf("arg%d", i)
		}
		fi.Parameters = append(fi.Parameters, p)
	}
	info.Constructor = fi
}

// extractFunctions looks up each configured function in the parsed ABI, applies
// access control settings, and stores the results on info in config order.
func extractFunctions(info *ContractInfo, funcConfigs []FunctionConfig, parsedABI abi.ABI) error {
	for _, funcCfg := range funcConfigs {
		matches := findMatchingMethods(parsedABI, funcCfg.Name)
		if len(matches) == 0 {
			return fmt.Errorf("function %s not found in ABI", funcCfg.Name)
		}

		for _, m := range matches {
			funcInfo := methodToFunctionInfo(m)

			switch funcCfg.Access {
			case "public":
			case "role":
				if !funcInfo.IsWrite {
					return fmt.Errorf(
						"function %s: access: role is only valid for non-view/non-pure functions",
						funcCfg.Name,
					)
				}
				if strings.TrimSpace(funcCfg.Role) == "" {
					return fmt.Errorf(
						"function %s: access: role requires role: (e.g. PAUSER_ROLE or 64-char hex)",
						funcCfg.Name,
					)
				}
				role, err := resolveRoleField(funcCfg.Role)
				if err != nil {
					return fmt.Errorf("function %s: %w", funcCfg.Name, err)
				}
				funcInfo.AccessControl = accessRole
				funcInfo.Role = role
			case "owner":
				if !funcInfo.IsWrite {
					return fmt.Errorf(
						"function %s: access: role is only valid for non-view/non-pure functions",
						funcCfg.Name,
					)
				}
				funcInfo.AccessControl = accessOwner
			default:
				return fmt.Errorf(
					"unknown access %q for function %s (use \"public\" or \"role\")",
					funcCfg.Access, funcCfg.Name,
				)
			}

			info.Functions[funcInfo.Name] = funcInfo
			info.FunctionOrder = append(info.FunctionOrder, funcInfo.Name)
		}
	}

	return nil
}

// collectAllStructDefs walks every selected function's parameters to discover
// all struct types that need to be defined in the generated file.
// Multi-return read functions also get a synthetic result struct.
func collectAllStructDefs(info *ContractInfo) {
	if info.Constructor != nil {
		collectStructDefs(info.Constructor.Parameters, info.StructDefs)
	}
	for _, funcInfo := range info.Functions {
		collectStructDefs(funcInfo.Parameters, info.StructDefs)
		collectStructDefs(funcInfo.ReturnParams, info.StructDefs)

		if !funcInfo.IsWrite && len(funcInfo.ReturnParams) > 1 {
			structName := multiReturnStructName(funcInfo.Name)
			if _, exists := info.StructDefs[structName]; !exists {
				info.StructDefs[structName] = &StructDef{
					Name:   structName,
					Fields: funcInfo.ReturnParams,
					// InternalType is intentionally empty: this is a synthetic struct,
					// not an ABI-sourced type, so it must be declared in the generated file.
				}
			}
		}
	}
}

// collectStructDefs recursively registers struct types found in a parameter list.
func collectStructDefs(params []ParameterInfo, structDefs map[string]*StructDef) {
	for _, param := range params {
		if param.IsStruct && param.StructName != "" {
			if _, exists := structDefs[param.StructName]; !exists {
				structDefs[param.StructName] = &StructDef{
					Name:         param.StructName,
					Fields:       param.Components,
					InternalType: param.StructName, // TupleRawName == geth binding name
				}
			}
			collectStructDefs(param.Components, structDefs)
		}
	}
}

// remapGobindingsTypes replaces GoType references to ABI struct names
// (e.g. "BaseAuctionAssetParams") with the fully-qualified geth_bindings name
// (e.g. "geth_bindings.BaseAuctionAssetParams") in all parameter lists, then
// removes those struct defs so the generator does not re-declare types that are
// already defined in the gobindings package.
func remapGobindingsTypes(info *ContractInfo) {
	remap := make(map[string]string, len(info.StructDefs))
	for name, sd := range info.StructDefs {
		if sd.InternalType != "" {
			// sd.InternalType == name == TupleRawName, which is already the geth type name.
			remap[name] = "geth_bindings." + name
		}
	}
	if len(remap) == 0 {
		return
	}

	remapParams := func(params []ParameterInfo) {
		for i := range params {
			p := &params[i]
			if p.IsStruct && p.StructName != "" {
				if mapped, ok := remap[p.StructName]; ok {
					// Replace the base struct name wherever it appears in GoType,
					// preserving any wrapping ([], [][], [N], etc.).
					p.GoType = strings.Replace(p.GoType, p.StructName, mapped, 1)
				}
			}
		}
	}

	if info.Constructor != nil {
		remapParams(info.Constructor.Parameters)
	}
	for _, fn := range info.Functions {
		remapParams(fn.Parameters)
		remapParams(fn.ReturnParams)
	}

	for name := range remap {
		delete(info.StructDefs, name)
	}
}
