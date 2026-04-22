package evm

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// findModuleRoot walks up the directory tree from startDir until it finds a
// go.mod file, then returns the directory containing it and the module path
// declared inside (e.g. "github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen").
func findModuleRoot(startDir string) (moduleDir, moduleName string, err error) {
	dir := startDir
	for {
		gomod := filepath.Join(dir, "go.mod")
		data, readErr := os.ReadFile(gomod)
		if readErr == nil {
			scanner := bufio.NewScanner(bytes.NewReader(data))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "module ") {
					return dir, strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
				}
			}

			return "", "", fmt.Errorf("no module directive found in %s", gomod)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", fmt.Errorf("go.mod not found starting from %s", startDir)
		}
		dir = parent
	}
}

// readABIFromGobindingsSource reads the ABI JSON string and (unless noDeployment)
// the deployment bytecode hex string directly from the gobindings Go source file.
//
// The gobindings package is located by stripping the module prefix from
// gobindingsPackage and resolving the path relative to the module root.
// moduleSearchDir must be a directory inside the module tree (findModuleRoot walks
// upward from it); use core.Config.ConfigDir when output.BasePath is outside the repo.
// The main source file is expected at <pkgDir>/<pkgName>.go and must contain
// a MetaData struct literal with "ABI:" and "Bin:" string fields.
func readABIFromGobindingsSource(
	gobindingsPackage, moduleSearchDir string,
	noDeployment bool,
) (abiStr, bin string, err error) {
	modRoot, modName, err := findModuleRoot(moduleSearchDir)
	if err != nil {
		return "", "", err
	}

	relPkg := strings.TrimPrefix(gobindingsPackage, modName+"/")
	if relPkg == gobindingsPackage {
		return "", "", fmt.Errorf(
			"gobindings_package %q does not start with module %q",
			gobindingsPackage, modName,
		)
	}

	pkgName := filepath.Base(relPkg)
	goFile := filepath.Join(modRoot, filepath.FromSlash(relPkg), pkgName+".go")

	content, err := os.ReadFile(goFile)
	if err != nil {
		return "", "", fmt.Errorf("failed to read gobindings source %s: %w", goFile, err)
	}

	abiStr, err = extractGoStringField(content, "ABI")
	if err != nil {
		return "", "", fmt.Errorf("failed to extract ABI from %s: %w", goFile, err)
	}

	if !noDeployment {
		bin, err = extractGoStringField(content, "Bin")
		if err != nil {
			return "", "", fmt.Errorf("failed to extract Bin from %s: %w", goFile, err)
		}
	}

	return abiStr, bin, nil
}

// goStringFieldRe matches lines of the form `FieldName: "..."` inside a Go
// struct literal and captures the field name and the quoted string value.
var goStringFieldRe = regexp.MustCompile(`(?m)^\s*(\w+):\s*("(?:[^"\\]|\\.)*")`)

// extractGoStringField finds the first occurrence of `fieldName: "..."` in
// content and returns the unquoted string value.
func extractGoStringField(content []byte, fieldName string) (string, error) {
	for _, m := range goStringFieldRe.FindAllSubmatch(content, -1) {
		if string(m[1]) == fieldName {
			return strconv.Unquote(string(m[2]))
		}
	}

	return "", fmt.Errorf("field %q not found", fieldName)
}
