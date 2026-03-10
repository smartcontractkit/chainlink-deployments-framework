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
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
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

// saveReports saves any new operations reports generated during changeset execution
// to the artifacts directory. It compares the current report count against the original
// count to determine which reports are new and need to be persisted.
func saveReports(
	reporter operations.Reporter, originalReportsLen int, lggr logger.Logger, artdir *domain.ArtifactsDir, changesetStr string,
) error {
	latestReports, err := reporter.GetReports()
	if err != nil {
		return err
	}
	newReports := len(latestReports) - originalReportsLen
	if newReports > 0 {
		lggr.Infof("Saving %d new operations reports...", newReports)
		if err := artdir.SaveOperationsReports(changesetStr, latestReports); err != nil {
			return err
		}
	}

	return nil
}

// configureEnvironmentOptions builds the environment loading options for a changeset execution.
// It configures the logger, chain overrides, JD settings, operation registry, and dry-run mode
// based on the changeset's registered options and the provided flags.
func configureEnvironmentOptions(
	cs *changeset.ChangesetsRegistry, changesetStr string, dryRun bool, lggr logger.Logger,
) ([]environment.LoadEnvironmentOption, error) {
	var envOptions []environment.LoadEnvironmentOption

	envOptions = append(envOptions, environment.WithLogger(lggr))

	changesetOptions, err := cs.GetChangesetOptions(changesetStr)
	if err != nil {
		return nil, err
	}

	chainOverrides, err := getChainOverrides(cs, changesetStr)
	if err != nil {
		return nil, err
	}
	if chainOverrides != nil {
		envOptions = append(envOptions, environment.OnlyLoadChainsFor(chainOverrides))
	}

	if changesetOptions.WithoutJD {
		envOptions = append(envOptions, environment.WithoutJD())
	}

	if changesetOptions.OperationRegistry != nil {
		envOptions = append(envOptions, environment.WithOperationRegistry(changesetOptions.OperationRegistry))
	}

	if dryRun {
		envOptions = append(envOptions, environment.WithDryRunJobDistributor())
	}

	return envOptions, nil
}

// getChainOverrides retrieves the chain overrides for a given changeset.
// It first checks for changeset options, and if not found, it retrieves input chain overrides.
func getChainOverrides(cs *changeset.ChangesetsRegistry, changesetStr string) ([]uint64, error) {
	changesetOptions, err := cs.GetChangesetOptions(changesetStr)
	if err != nil {
		return nil, err
	}

	if changesetOptions.ChainsToLoad != nil {
		return changesetOptions.ChainsToLoad, nil
	}

	// this is only applicable to durable pipelines
	configs, err := cs.GetConfigurations(changesetStr)
	if err != nil {
		return nil, err
	}

	return configs.InputChainOverrides, nil
}
