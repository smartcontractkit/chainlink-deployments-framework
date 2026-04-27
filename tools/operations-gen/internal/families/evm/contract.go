package evm

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/core"
)

const (
	accessPublic = "public"
	accessOwner  = "owner"
	accessRole   = "role"
)

// ---- Intermediate representation ----

// ContractInfo holds all parsed information about a contract needed for code generation.
type ContractInfo struct {
	Name              string
	Version           string
	PackageName       string
	GobindingsPackage string
	OutputPath        string
	OmitDeploy        bool
	Constructor       *FunctionInfo
	Functions         map[string]*FunctionInfo
	FunctionOrder     []string
	StructDefs        map[string]*structDef
}

type structDef struct {
	Name   string
	Fields []ParameterInfo
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

type ParameterInfo struct {
	Name       string
	GoType     string
	IsStruct   bool
	StructName string
	Components []ParameterInfo
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

	cfg.GobindingsPackage = resolveGobindingsPackage(cfg.GobindingsPackage, input.GobindingsPackage, versionPath, packageName)
	if cfg.GobindingsPackage == "" {
		return nil, fmt.Errorf("gobindings_package is required for contract %q; set either contract gobindings_package or input.gobindings_package", cfg.Name)
	}

	parsedAbi, err := ReadABI(cfg)
	if err != nil {
		return nil, err
	}

	info := &ContractInfo{
		Name:              cfg.Name,
		Version:           cfg.Version,
		PackageName:       packageName,
		GobindingsPackage: cfg.GobindingsPackage,
		OutputPath:        core.ContractOutputPath(output.BasePath, versionPath, packageName),
		OmitDeploy:        cfg.OmitDeploy,
		Functions:         make(map[string]*FunctionInfo),
		StructDefs:        make(map[string]*structDef),
	}

	extractConstructor(info, parsedAbi)

	if err := extractFunctions(info, cfg.Functions, parsedAbi); err != nil {
		return nil, err
	}

	collectAllStructDefs(info)

	return info, nil
}

func resolveGobindingsPackage(contractPackage, parentPackage, versionPath, packageName string) string {
	if contractPackage != "" {
		return contractPackage
	}
	if parentPackage == "" {
		return ""
	}

	return strings.TrimSuffix(parentPackage, "/") + "/" + versionPath + "/" + packageName
}

// extractConstructor populates info.Constructor when the ABI defines a
// constructor with one or more inputs.
func extractConstructor(info *ContractInfo, parsedABI *abi.ABI) {
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

func extractFunctions(info *ContractInfo, funcConfigs []EvmFunctionConfig, parsedAbi *abi.ABI) error {
	for _, funcCfg := range funcConfigs {
		methods := FindFunctionInABI(parsedAbi, funcCfg.Name)
		if len(methods) == 0 {
			return fmt.Errorf("function %s not found in ABI", funcCfg.Name)
		}

		for _, m := range methods {
			fi := methodToFunctionInfo(m)

			switch funcCfg.Access {
			case accessOwner:
				fi.AccessControl = accessOwner
			case accessPublic, "":
				fi.AccessControl = accessPublic
			case accessRole:
				if funcCfg.Role == "" {
					return fmt.Errorf("role is required when access is %q for function %s", accessRole, funcCfg.Name)
				}
				role, err := ResolveRoleField(funcCfg.Role)
				if err != nil {
					return fmt.Errorf("failed to resolve role %q for function %s: %w", funcCfg.Role, funcCfg.Name, err)
				}
				fi.AccessControl = accessRole
				fi.Role = role
			default:
				return fmt.Errorf("unknown access control '%s' for function %s (use 'owner', 'public', or 'role')",
					funcCfg.Access, funcCfg.Name)
			}

			info.Functions[fi.Name] = fi
			info.FunctionOrder = append(info.FunctionOrder, fi.Name)
		}
	}

	return nil
}

// collectAllStructDefs registers the struct defs that the operations file
// must declare locally. The only such case today is the synthetic XxxResult
// struct used to pack multi-return reads; Solidity tuple types are referenced
// directly via the "gobindings" import and therefore do not need re-declaring.
func collectAllStructDefs(info *ContractInfo) {
	for _, fi := range info.Functions {
		if fi.IsWrite || len(fi.ReturnParams) <= 1 {
			continue
		}
		structName := multiReturnStructName(fi.Name)
		if _, exists := info.StructDefs[structName]; !exists {
			info.StructDefs[structName] = &structDef{
				Name:   structName,
				Fields: fi.ReturnParams,
			}
		}
	}
}

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
