package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suzuki-shunsuke/go-convmap/convmap"
	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// resolveChangesetConfig resolves the configuration for a changeset using either a registered resolver or keeping the original payload
func resolveChangesetConfig(valueNode *yaml.Node, csName string, resolver resolvers.ConfigResolver) (any, error) {
	var resolvedCfg any

	var changesetData struct {
		Payload any `yaml:"payload"`
	}

	err := valueNode.Decode(&changesetData)
	if err != nil {
		return nil, fmt.Errorf("decode changeset data for %s: %w", csName, err)
	}

	if resolver != nil {
		// Convert YAML-decoded payload to JSON-safe format
		jsonSafePayload, err := convmap.Convert(changesetData.Payload, nil)
		if err != nil {
			return nil, fmt.Errorf("convert payload for %s: %w", csName, err)
		}
		raw, err := json.Marshal(jsonSafePayload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload for %s: %w", csName, err)
		}

		// Call the resolver with the JSON payload
		resolvedCfg, err = resolvers.CallResolver[any](resolver, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve config for changeset %q: %w", csName, err)
		}
	} else {
		resolvedCfg = changesetData.Payload
	}

	return resolvedCfg, nil
}

// Helper Functions

// findWorkspaceRoot finds the root of the workspace by looking for the go.mod file and the domains directory
func findWorkspaceRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up directories looking for the root go.mod
	for {
		// Check if this looks like the workspace root by looking for domains/
		if _, err := os.Stat(filepath.Join(dir, "domains")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root directory
		}
		dir = parent
	}

	return "", errors.New("could not find workspace root (directory with go.mod and domains/)")
}

// resolveDurablePipelineYamlPath resolves a YAML file path for durable pipelines.
// It only accepts filenames (not full paths) and resolves them to the appropriate
// durable_pipelines/inputs directory based on the domain and environment.
func resolveDurablePipelineYamlPath(inputFileName string, domain domain.Domain, envKey string) (string, error) {
	// Only support filenames, not full paths
	if filepath.Dir(inputFileName) != "." {
		return "", fmt.Errorf("only filenames are supported, not full paths: %s", inputFileName)
	}

	// It's just a filename, resolve it to the appropriate inputs directory
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return "", fmt.Errorf("find workspace root: %w", err)
	}

	resolvedPath := filepath.Join(
		workspaceRoot, "domains", domain.String(),
		envKey, "durable_pipelines", "inputs", inputFileName,
	)

	return resolvedPath, nil
}
