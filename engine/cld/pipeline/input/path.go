package input

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// validateInputFileName ensures the given value is a bare filename (no directory components)
// and is non-empty. This prevents path-traversal via values like "../other.yaml" or "subdir/f.yaml"
// when the filename ultimately comes from a CLI flag.
func validateInputFileName(inputFileName string) error {
	if strings.TrimSpace(inputFileName) == "" {
		return errors.New("input file name must not be empty")
	}

	if filepath.Dir(inputFileName) != "." {
		return fmt.Errorf("only filenames are supported, not full paths: %s", inputFileName)
	}

	return nil
}

// ResolveDurablePipelineYamlPath resolves a YAML file path for durable pipelines.
// It only accepts filenames (not full paths) and resolves them to the appropriate
// durable_pipelines/inputs directory based on the domain and environment.
func ResolveDurablePipelineYamlPath(inputFileName string, dom domain.Domain, envKey string) (string, error) {
	if err := validateInputFileName(inputFileName); err != nil {
		return "", err
	}

	return filepath.Join(
		dom.EnvDir(envKey).DurablePipelinesInputsDirPath(), inputFileName,
	), nil
}
