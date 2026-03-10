package commands

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/pipeline/input"
)

// These functions delegate to the pipeline/input package.
// Kept for backward compatibility with tests.

func setDurablePipelineInputFromYAML(inputFileName, changesetName string, dom domain.Domain, envKey string) error {
	return input.PrepareInputForRunByName(inputFileName, changesetName, dom, envKey)
}

func setDurablePipelineInputFromYAMLByIndex(inputFileName string, index int, dom domain.Domain, envKey string) (string, error) {
	return input.PrepareInputForRunByIndex(inputFileName, index, dom, envKey)
}

func findChangesetInData(changesets any, changesetName, inputFileName string) (any, error) {
	return input.FindChangesetInData(changesets, changesetName, inputFileName)
}

func getAllChangesetsInOrder(changesets any, inputFileName string) ([]struct {
	name string
	data any
}, error) {
	items, err := input.GetAllChangesetsInOrder(changesets, inputFileName)
	if err != nil {
		return nil, err
	}
	result := make([]struct {
		name string
		data any
	}, len(items))
	for i, item := range items {
		result[i] = struct {
			name string
			data any
		}{name: item.Name, data: item.Data}
	}

	return result, nil
}

func parseDurablePipelineYAML(inputFileName string, dom domain.Domain, envKey string) (*durablePipelineYAML, error) {
	dp, err := input.ParseDurablePipelineYAML(inputFileName, dom, envKey)
	if err != nil {
		return nil, err
	}

	return &durablePipelineYAML{
		Environment: dp.Environment,
		Domain:      dp.Domain,
		Changesets:  dp.Changesets,
	}, nil
}

type durablePipelineYAML struct {
	Environment string
	Domain      string
	Changesets  any
}

func convertToJSONSafe(data any) (any, error) {
	return input.ConvertToJSONSafe(data)
}
