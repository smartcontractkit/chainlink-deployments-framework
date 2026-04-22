package evm

import (
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/core"
)

// ---- Intermediate representation ----

// ContractInfo holds all parsed information about a contract needed for code generation.
type ContractInfo struct {
	Name          string
	Version       string
	PackageName   string
	OutputPath    string
	ABI           string
	Bytecode      string
	OmitDeploy    bool
	Constructor   *FunctionInfo
	Functions     map[string]*FunctionInfo
	FunctionOrder []string
	StructDefs    map[string]*structDef
}

type structDef struct {
	Name   string
	Fields []ParameterInfo
}

type FunctionInfo struct {
	Name         string
	Parameters   []ParameterInfo
	ReturnParams []ParameterInfo
	IsWrite      bool
	CallMethod   string // Method name, with numeric suffix for overloaded functions
	HasOnlyOwner bool
}

type ParameterInfo struct {
	Name         string
	SolidityType string
	GoType       string
	IsStruct     bool
	StructName   string
	Components   []ParameterInfo
}

// ---- Extraction ----

func extractContractInfo(cfg EvmContractConfig, input EvmInputConfig, output EvmOutputConfig) (*ContractInfo, error) {
	if cfg.Name == "" || cfg.Version == "" {
		return nil, errors.New("contract_name and version are required")
	}

	packageName := cfg.PackageName
	if packageName == "" {
		packageName = ToSnakeCase(cfg.Name)
	}
	versionPath := core.VersionToPath(cfg.Version)
	if cfg.VersionPath != "" {
		versionPath = cfg.VersionPath
	}

	if err := validatePathSegment("package_name", packageName); err != nil {
		return nil, err
	}
	if err := validatePathSegment("version_path", versionPath); err != nil {
		return nil, err
	}

	abiString, bytecode, err := ReadABIAndByteCode(cfg, packageName, versionPath, input)
	if err != nil {
		return nil, err
	}

	abiEntries, err := parseABIEntries(abiString)
	if err != nil {
		return nil, err
	}

	info := &ContractInfo{
		Name:        cfg.Name,
		Version:     cfg.Version,
		PackageName: packageName,
		OutputPath:  core.ContractOutputPath(output.BasePath, versionPath, packageName),
		ABI:         abiString,
		Bytecode:    bytecode,
		OmitDeploy:  cfg.OmitDeploy,
		Functions:   make(map[string]*FunctionInfo),
		StructDefs:  make(map[string]*structDef),
	}

	extractConstructor(info, abiEntries, EvmTypeMap)

	if err := extractFunctions(info, cfg.Functions, abiEntries, EvmTypeMap); err != nil {
		return nil, err
	}

	collectAllStructDefs(info)

	return info, nil
}

func extractConstructor(info *ContractInfo, abiEntries []ABIEntry, typeMap map[string]string) {
	for _, entry := range abiEntries {
		if entry.Type == abiTypeConstructor {
			info.Constructor = ParseABIFunction(entry, info.PackageName, typeMap)
			break
		}
	}
}

func extractFunctions(info *ContractInfo, funcConfigs []evmFunctionConfig, abiEntries []ABIEntry, typeMap map[string]string) error {
	for _, funcCfg := range funcConfigs {
		funcInfos := FindFunctionInABI(abiEntries, funcCfg.Name, info.PackageName, typeMap)
		if funcInfos == nil {
			return fmt.Errorf("function %s not found in ABI", funcCfg.Name)
		}

		for _, fi := range funcInfos {
			switch funcCfg.Access {
			case accessOwner:
				fi.HasOnlyOwner = true
			case accessPublic, "":
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

func collectAllStructDefs(info *ContractInfo) {
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

func collectStructDefs(params []ParameterInfo, structDefs map[string]*structDef) {
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
