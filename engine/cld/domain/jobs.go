package domain

import (
	"encoding/json"
	"fmt"
	"os"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

// LoadJobSpecs loads job specs from a file.

// Deprecated: The map used to represent the job specs that was provided by
// `chainlink/deployments` is no longer used, and has been replaced by a slice
// of `fdeployment.ProposedJob` structs. This has been kept around for backwards
// compatibility with existing migrations, but should not be used in new code.
// Prefer using `LoadJobs` instead.
func LoadJobSpecs(jobSpecsFilePath string) (map[string][]string, error) {
	specs := make(map[string][]string)
	b, err := os.ReadFile(jobSpecsFilePath)
	if err != nil {
		return specs, err
	}

	if err = json.Unmarshal(b, &specs); err != nil {
		return specs, fmt.Errorf("unable to unmarshal data: %w", err)
	}

	return specs, nil
}

// LoadJobs unmarshals a slice of `fdeployment.ProposedJob` structs from a file.
func LoadJobs(jobsFilePath string) ([]fdeployment.ProposedJob, error) {
	jobs := make([]fdeployment.ProposedJob, 0)
	b, err := os.ReadFile(jobsFilePath)
	if err != nil {
		return jobs, err
	}

	if err = json.Unmarshal(b, &jobs); err != nil {
		return jobs, fmt.Errorf("unable to unmarshal data: %w", err)
	}

	return jobs, nil
}
