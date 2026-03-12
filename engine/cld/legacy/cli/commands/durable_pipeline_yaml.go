package commands

import (
	"fmt"
	"os"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/durablepipeline"
)

// setDurablePipelineInputFromYAML reads a YAML file, extracts the payload for the specified changeset,
// and sets it as the DURABLE_PIPELINE_INPUT environment variable in JSON format.
// If inputFileName is just a filename (no path separators), it will be resolved relative to the
// appropriate durable_pipelines/inputs directory based on the domain and environment.
func setDurablePipelineInputFromYAML(inputFileName, changesetName string, domain domain.Domain, envKey string) error {
	dpYAML, err := parseDurablePipelineYAML(inputFileName, domain, envKey)
	if err != nil {
		return err
	}

	changesetData, err := durablepipeline.FindChangesetInData(dpYAML.Changesets, changesetName)
	if err != nil {
		return fmt.Errorf("input file %s: %w", inputFileName, err)
	}

	return durablepipeline.SetChangesetEnvironmentVariable(changesetName, changesetData, inputFileName)
}

// setDurablePipelineInputFromYAMLByIndex sets the DURABLE_PIPELINE_INPUT environment variable
// by selecting the changeset at the specified index position in the input file.
// This function only works with array format YAML files, not object format.
func setDurablePipelineInputFromYAMLByIndex(inputFileName string, index int, domain domain.Domain, envKey string) (string, error) {
	dpYAML, err := parseDurablePipelineYAML(inputFileName, domain, envKey)
	if err != nil {
		return "", err
	}

	// Validate that the changesets are in array format (required for index-based access)
	if _, isArray := dpYAML.Changesets.([]any); !isArray {
		return "", fmt.Errorf("--changeset-index can only be used with array format YAML files. Input file %s uses object format. Use --changeset instead", inputFileName)
	}

	// Get all changesets in order
	changesets, err := durablepipeline.GetAllChangesetsInOrder(dpYAML.Changesets)
	if err != nil {
		return "", fmt.Errorf("input file %s: %w", inputFileName, err)
	}

	if index < 0 || index >= len(changesets) {
		return "", fmt.Errorf("changeset index %d is out of range (found %d changesets in %s)", index, len(changesets), inputFileName)
	}

	selectedChangeset := changesets[index]

	// Use the existing logic to set the environment variable
	if err := durablepipeline.SetChangesetEnvironmentVariable(selectedChangeset.Name, selectedChangeset.Data, inputFileName); err != nil {
		return "", err
	}

	return selectedChangeset.Name, nil
}

// parseDurablePipelineYAML parses and validates a durable pipeline YAML file
func parseDurablePipelineYAML(inputFileName string, domain domain.Domain, envKey string) (*durablepipeline.ParsedYAML, error) {
	resolvedPath, err := resolveDurablePipelineYamlPath(inputFileName, domain, envKey)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve input file path: %w", err)
	}

	yamlData, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read input file %s: %w", resolvedPath, err)
	}

	parsed, err := durablepipeline.ParseYAMLBytes(yamlData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse input file %s: %w", inputFileName, err)
	}

	return parsed, nil
}
