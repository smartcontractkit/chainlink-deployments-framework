package input

import (
	"fmt"
	"path/filepath"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// ResolveDurablePipelineYamlPath resolves a YAML file path for durable pipelines.
// It only accepts filenames (not full paths) and resolves them to the appropriate
// durable_pipelines/inputs directory based on the domain and environment.
func ResolveDurablePipelineYamlPath(inputFileName string, dom domain.Domain, envKey string) (string, error) {
	if filepath.Dir(inputFileName) != "." {
		return "", fmt.Errorf("only filenames are supported, not full paths: %s", inputFileName)
	}

	workspaceRoot, err := FindWorkspaceRoot()
	if err != nil {
		return "", fmt.Errorf("find workspace root: %w", err)
	}

	return filepath.Join(
		workspaceRoot, "domains", dom.String(),
		envKey, "durable_pipelines", "inputs", inputFileName,
	), nil
}
