package memory

import (
	"slices"

	jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"
	"github.com/smartcontractkit/chainlink-protos/job-distributor/v1/shared/ptypes"
)

// applyJobFilter applies the filter to the list of jobs and returns the filtered results.
func applyJobFilter(
	jobs []*jobv1.Job, filter *jobv1.ListJobsRequest_Filter,
) []*jobv1.Job {
	var filtered []*jobv1.Job

	for _, job := range jobs {
		if jobMatchesFilter(job, filter) {
			filtered = append(filtered, job)
		}
	}

	return filtered
}

// jobMatchesFilter checks if a job matches the given filter criteria.
func jobMatchesFilter(job *jobv1.Job, filter *jobv1.ListJobsRequest_Filter) bool {
	// Check if job is soft-deleted and should be excluded
	if !jobMatchesDeletedFilter(job, filter) {
		return false
	}

	// Check job IDs
	if len(filter.Ids) > 0 {
		if !jobMatchesJobIds(job, filter.Ids) {
			return false
		}
	}

	// Check UUIDs
	if len(filter.Uuids) > 0 {
		if !jobMatchesUuids(job, filter.Uuids) {
			return false
		}
	}

	// Check node IDs
	if len(filter.NodeIds) > 0 {
		if !jobMatchesNodeIds(job, filter.NodeIds) {
			return false
		}
	}

	// Check selectors
	if len(filter.Selectors) > 0 {
		for _, selector := range filter.Selectors {
			if !jobMatchesSelector(job, selector) {
				return false
			}
		}
	}

	return true
}

// jobMatchesJobIds checks if a job's ID is in the provided list of job IDs.
func jobMatchesJobIds(job *jobv1.Job, jobIds []string) bool {
	return slices.Contains(jobIds, job.Id)
}

// jobMatchesUuids checks if a job's UUID is in the provided list of UUIDs.
func jobMatchesUuids(job *jobv1.Job, uuids []string) bool {
	return slices.Contains(uuids, job.Uuid)
}

// jobMatchesNodeIds checks if a job's node ID is in the provided list of node IDs.
func jobMatchesNodeIds(job *jobv1.Job, nodeIds []string) bool {
	return slices.Contains(nodeIds, job.NodeId)
}

// jobMatchesDeletedFilter checks if a job should be included based on its deleted status.
// By default, soft-deleted jobs (with DeletedAt set) are excluded unless IncludeDeleted is true.
func jobMatchesDeletedFilter(job *jobv1.Job, filter *jobv1.ListJobsRequest_Filter) bool {
	// If job is soft-deleted (DeletedAt is not nil)
	if job.DeletedAt != nil {
		// Only include if IncludeDeleted is explicitly set to true
		return filter.IncludeDeleted
	}

	// If job is not soft-deleted, always include it
	return true
}

// jobMatchesSelector checks if a job matches a specific selector.
func jobMatchesSelector(job *jobv1.Job, selector *ptypes.Selector) bool {
	// Get the job's labels as a map for easier lookup
	jobLabels := make(map[string]*string)
	for _, label := range job.Labels {
		jobLabels[label.Key] = label.Value
	}

	return matchesSelector(jobLabels, selector)
}
