package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"
	"github.com/smartcontractkit/chainlink-protos/job-distributor/v1/shared/ptypes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smartcontractkit/chainlink-deployments-framework/internal/pointer"
)

func TestApplyJobFilter(t *testing.T) {
	t.Parallel()

	// Create test jobs
	jobs := []*jobv1.Job{
		{
			Id:     "job-1",
			Uuid:   "uuid-1",
			NodeId: "node-1",
			Labels: []*ptypes.Label{
				{Key: "environment", Value: pointer.To("prod")},
				{Key: "type", Value: pointer.To("ocr")},
			},
		},
		{
			Id:     "job-2",
			Uuid:   "uuid-2",
			NodeId: "node-2",
			Labels: []*ptypes.Label{
				{Key: "environment", Value: pointer.To("test")},
				{Key: "type", Value: pointer.To("flux")},
			},
		},
		{
			Id:     "job-3",
			Uuid:   "uuid-3",
			NodeId: "node-1",
			Labels: []*ptypes.Label{
				{Key: "environment", Value: pointer.To("prod")},
				{Key: "type", Value: pointer.To("flux")},
			},
		},
	}

	tests := []struct {
		name     string
		jobs     []*jobv1.Job
		filter   *jobv1.ListJobsRequest_Filter
		expected []string // Expected job IDs
	}{
		{
			name:     "no filter - return all jobs",
			jobs:     jobs,
			filter:   &jobv1.ListJobsRequest_Filter{},
			expected: []string{"job-1", "job-2", "job-3"},
		},
		{
			name: "filter by job IDs",
			jobs: jobs,
			filter: &jobv1.ListJobsRequest_Filter{
				Ids: []string{"job-1", "job-2"},
			},
			expected: []string{"job-1", "job-2"},
		},
		{
			name: "filter by node IDs",
			jobs: jobs,
			filter: &jobv1.ListJobsRequest_Filter{
				NodeIds: []string{"node-1"},
			},
			expected: []string{"job-1", "job-3"},
		},
		{
			name: "filter by UUIDs",
			jobs: jobs,
			filter: &jobv1.ListJobsRequest_Filter{
				Uuids: []string{"uuid-1", "uuid-2"},
			},
			expected: []string{"job-1", "job-2"},
		},
		{
			name: "filter by label",
			jobs: jobs,
			filter: &jobv1.ListJobsRequest_Filter{
				Selectors: []*ptypes.Selector{
					{
						Key:   "environment",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("prod"),
					},
				},
			},
			expected: []string{"job-1", "job-3"},
		},
		{
			name: "combined filters",
			jobs: jobs,
			filter: &jobv1.ListJobsRequest_Filter{
				Ids:     []string{"job-1", "job-2", "job-3"},
				NodeIds: []string{"node-1"},
				Selectors: []*ptypes.Selector{
					{
						Key:   "environment",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("prod"),
					},
				},
			},
			expected: []string{"job-1", "job-3"},
		},
		{
			name: "empty jobs list",
			jobs: []*jobv1.Job{},
			filter: &jobv1.ListJobsRequest_Filter{
				NodeIds: []string{"node-1"},
			},
			expected: []string{},
		},
		{
			name: "filter excludes soft-deleted jobs by default",
			jobs: []*jobv1.Job{
				{
					Id:     "job-1",
					Uuid:   "uuid-1",
					NodeId: "node-1",
				},
				{
					Id:        "job-2",
					Uuid:      "uuid-2",
					NodeId:    "node-2",
					DeletedAt: &timestamppb.Timestamp{Seconds: 1234567890},
				},
			},
			filter:   &jobv1.ListJobsRequest_Filter{},
			expected: []string{"job-1"},
		},
		{
			name: "filter includes soft-deleted jobs when IncludeDeleted is true",
			jobs: []*jobv1.Job{
				{
					Id:     "job-1",
					Uuid:   "uuid-1",
					NodeId: "node-1",
				},
				{
					Id:        "job-2",
					Uuid:      "uuid-2",
					NodeId:    "node-2",
					DeletedAt: &timestamppb.Timestamp{Seconds: 1234567890},
				},
			},
			filter: &jobv1.ListJobsRequest_Filter{
				IncludeDeleted: true,
			},
			expected: []string{"job-1", "job-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := applyJobFilter(tt.jobs, tt.filter)

			// Extract job IDs for comparison
			resultIds := make([]string, len(result))
			for i, job := range result {
				resultIds[i] = job.Id
			}

			require.ElementsMatch(t, tt.expected, resultIds)
		})
	}
}

func TestJobMatchesIds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		job    *jobv1.Job
		jobIds []string
		want   bool
	}{
		{
			name:   "job id matches",
			job:    &jobv1.Job{Id: "job-1"},
			jobIds: []string{"job-1", "job-2"},
			want:   true,
		},
		{
			name:   "job id does not match",
			job:    &jobv1.Job{Id: "job-3"},
			jobIds: []string{"job-1", "job-2"},
			want:   false,
		},
		{
			name:   "empty job ids list",
			job:    &jobv1.Job{Id: "job-1"},
			jobIds: []string{},
			want:   false,
		},
		{
			name:   "single job id match",
			job:    &jobv1.Job{Id: "job-1"},
			jobIds: []string{"job-1"},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := jobMatchesJobIds(tt.job, tt.jobIds)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestJobMatchesUuids(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		job   *jobv1.Job
		uuids []string
		want  bool
	}{
		{
			name:  "uuid matches",
			job:   &jobv1.Job{Id: "job-1", Uuid: "uuid-1"},
			uuids: []string{"uuid-1", "uuid-2"},
			want:  true,
		},
		{
			name:  "uuid does not match",
			job:   &jobv1.Job{Id: "job-3", Uuid: "uuid-3"},
			uuids: []string{"uuid-1", "uuid-2"},
			want:  false,
		},
		{
			name:  "empty uuids list",
			job:   &jobv1.Job{Id: "job-1", Uuid: "uuid-1"},
			uuids: []string{},
			want:  false,
		},
		{
			name:  "single uuid match",
			job:   &jobv1.Job{Id: "job-1", Uuid: "uuid-1"},
			uuids: []string{"uuid-1"},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := jobMatchesUuids(tt.job, tt.uuids)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestJobMatchesDeletedFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		job    *jobv1.Job
		filter *jobv1.ListJobsRequest_Filter
		want   bool
	}{
		{
			name: "non-deleted job with no filter",
			job: &jobv1.Job{
				Id: "job-1",
			},
			filter: &jobv1.ListJobsRequest_Filter{},
			want:   true,
		},
		{
			name: "non-deleted job with IncludeDeleted false",
			job: &jobv1.Job{
				Id: "job-1",
			},
			filter: &jobv1.ListJobsRequest_Filter{
				IncludeDeleted: false,
			},
			want: true,
		},
		{
			name: "non-deleted job with IncludeDeleted true",
			job: &jobv1.Job{
				Id: "job-1",
			},
			filter: &jobv1.ListJobsRequest_Filter{
				IncludeDeleted: true,
			},
			want: true,
		},
		{
			name: "soft-deleted job with no filter",
			job: &jobv1.Job{
				Id:        "job-1",
				DeletedAt: &timestamppb.Timestamp{Seconds: 1234567890},
			},
			filter: &jobv1.ListJobsRequest_Filter{},
			want:   false,
		},
		{
			name: "soft-deleted job with IncludeDeleted false",
			job: &jobv1.Job{
				Id:        "job-1",
				DeletedAt: &timestamppb.Timestamp{Seconds: 1234567890},
			},
			filter: &jobv1.ListJobsRequest_Filter{
				IncludeDeleted: false,
			},
			want: false,
		},
		{
			name: "soft-deleted job with IncludeDeleted true",
			job: &jobv1.Job{
				Id:        "job-1",
				DeletedAt: &timestamppb.Timestamp{Seconds: 1234567890},
			},
			filter: &jobv1.ListJobsRequest_Filter{
				IncludeDeleted: true,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := jobMatchesDeletedFilter(tt.job, tt.filter)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestJobMatchesNodeIds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		job     *jobv1.Job
		nodeIds []string
		want    bool
	}{
		{
			name:    "node id matches",
			job:     &jobv1.Job{NodeId: "node-1"},
			nodeIds: []string{"node-1", "node-2"},
			want:    true,
		},
		{
			name:    "node id does not match",
			job:     &jobv1.Job{NodeId: "node-3"},
			nodeIds: []string{"node-1", "node-2"},
			want:    false,
		},
		{
			name:    "empty node ids list",
			job:     &jobv1.Job{NodeId: "node-1"},
			nodeIds: []string{},
			want:    false,
		},
		{
			name:    "single node id match",
			job:     &jobv1.Job{NodeId: "node-1"},
			nodeIds: []string{"node-1"},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := jobMatchesNodeIds(tt.job, tt.nodeIds)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestJobMatchesSelector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		job      *jobv1.Job
		selector *ptypes.Selector
		want     bool
	}{
		{
			name: "basic selector matching",
			job: &jobv1.Job{
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("prod")},
					{Key: "type", Value: pointer.To("ocr")},
				},
			},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_EQ,
				Value: pointer.To("prod"),
			},
			want: true,
		},
		{
			name: "job with nil label value",
			job: &jobv1.Job{
				Labels: []*ptypes.Label{
					{Key: "environment", Value: nil},
					{Key: "type", Value: pointer.To("ocr")},
				},
			},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_EQ,
				Value: pointer.To("prod"),
			},
			want: false,
		},
		{
			name: "job with empty labels",
			job: &jobv1.Job{
				Labels: []*ptypes.Label{},
			},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_EQ,
				Value: pointer.To("prod"),
			},
			want: false,
		},
		{
			name: "job with nil labels",
			job: &jobv1.Job{
				Labels: nil,
			},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_EQ,
				Value: pointer.To("prod"),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := jobMatchesSelector(tt.job, tt.selector)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestJobMatchesFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		job    *jobv1.Job
		filter *jobv1.ListJobsRequest_Filter
		want   bool
	}{
		{
			name: "no filter - should match",
			job: &jobv1.Job{
				Id:     "job-1",
				NodeId: "node-1",
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("prod")},
				},
			},
			filter: &jobv1.ListJobsRequest_Filter{},
			want:   true,
		},
		{
			name: "job id filter - matching id",
			job: &jobv1.Job{
				Id:     "job-1",
				NodeId: "node-1",
			},
			filter: &jobv1.ListJobsRequest_Filter{
				Ids: []string{"job-1", "job-2"},
			},
			want: true,
		},
		{
			name: "job id filter - non-matching id",
			job: &jobv1.Job{
				Id:     "job-3",
				NodeId: "node-1",
			},
			filter: &jobv1.ListJobsRequest_Filter{
				Ids: []string{"job-1", "job-2"},
			},
			want: false,
		},
		{
			name: "node id filter - matching node id",
			job: &jobv1.Job{
				Id:     "job-1",
				NodeId: "node-1",
			},
			filter: &jobv1.ListJobsRequest_Filter{
				NodeIds: []string{"node-1", "node-2"},
			},
			want: true,
		},
		{
			name: "node id filter - non-matching node id",
			job: &jobv1.Job{
				Id:     "job-1",
				NodeId: "node-3",
			},
			filter: &jobv1.ListJobsRequest_Filter{
				NodeIds: []string{"node-1", "node-2"},
			},
			want: false,
		},
		{
			name: "uuid filter - matching uuid",
			job: &jobv1.Job{
				Id:     "job-1",
				Uuid:   "uuid-1",
				NodeId: "node-1",
			},
			filter: &jobv1.ListJobsRequest_Filter{
				Uuids: []string{"uuid-1", "uuid-2"},
			},
			want: true,
		},
		{
			name: "uuid filter - non-matching uuid",
			job: &jobv1.Job{
				Id:     "job-3",
				Uuid:   "uuid-3",
				NodeId: "node-1",
			},
			filter: &jobv1.ListJobsRequest_Filter{
				Uuids: []string{"uuid-1", "uuid-2"},
			},
			want: false,
		},
		{
			name: "selector filter - matching selector",
			job: &jobv1.Job{
				Id:     "job-1",
				NodeId: "node-1",
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("prod")},
				},
			},
			filter: &jobv1.ListJobsRequest_Filter{
				Selectors: []*ptypes.Selector{
					{
						Key:   "environment",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("prod"),
					},
				},
			},
			want: true,
		},
		{
			name: "selector filter - non-matching selector",
			job: &jobv1.Job{
				Id:     "job-1",
				NodeId: "node-1",
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("test")},
				},
			},
			filter: &jobv1.ListJobsRequest_Filter{
				Selectors: []*ptypes.Selector{
					{
						Key:   "environment",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("prod"),
					},
				},
			},
			want: false,
		},
		{
			name: "multiple selectors - all match",
			job: &jobv1.Job{
				Id:     "job-1",
				NodeId: "node-1",
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("prod")},
					{Key: "type", Value: pointer.To("ocr")},
				},
			},
			filter: &jobv1.ListJobsRequest_Filter{
				Selectors: []*ptypes.Selector{
					{
						Key:   "environment",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("prod"),
					},
					{
						Key:   "type",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("ocr"),
					},
				},
			},
			want: true,
		},
		{
			name: "multiple selectors - one does not match",
			job: &jobv1.Job{
				Id:     "job-1",
				NodeId: "node-1",
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("prod")},
					{Key: "type", Value: pointer.To("flux")},
				},
			},
			filter: &jobv1.ListJobsRequest_Filter{
				Selectors: []*ptypes.Selector{
					{
						Key:   "environment",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("prod"),
					},
					{
						Key:   "type",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("ocr"),
					},
				},
			},
			want: false,
		},
		{
			name: "combined filters - all match",
			job: &jobv1.Job{
				Id:     "job-1",
				NodeId: "node-1",
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("prod")},
				},
			},
			filter: &jobv1.ListJobsRequest_Filter{
				Ids:     []string{"job-1", "job-2"},
				NodeIds: []string{"node-1", "node-2"},
				Selectors: []*ptypes.Selector{
					{
						Key:   "environment",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("prod"),
					},
				},
			},
			want: true,
		},
		{
			name: "combined filters - job id does not match",
			job: &jobv1.Job{
				Id:     "job-3",
				NodeId: "node-1",
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("prod")},
				},
			},
			filter: &jobv1.ListJobsRequest_Filter{
				Ids:     []string{"job-1", "job-2"},
				NodeIds: []string{"node-1", "node-2"},
				Selectors: []*ptypes.Selector{
					{
						Key:   "environment",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("prod"),
					},
				},
			},
			want: false,
		},
		{
			name: "combined filters - node id does not match",
			job: &jobv1.Job{
				Id:     "job-1",
				NodeId: "node-3",
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("prod")},
				},
			},
			filter: &jobv1.ListJobsRequest_Filter{
				Ids:     []string{"job-1", "job-2"},
				NodeIds: []string{"node-1", "node-2"},
				Selectors: []*ptypes.Selector{
					{
						Key:   "environment",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("prod"),
					},
				},
			},
			want: false,
		},
		{
			name: "combined filters - selector does not match",
			job: &jobv1.Job{
				Id:     "job-1",
				NodeId: "node-1",
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("test")},
				},
			},
			filter: &jobv1.ListJobsRequest_Filter{
				Ids:     []string{"job-1", "job-2"},
				NodeIds: []string{"node-1", "node-2"},
				Selectors: []*ptypes.Selector{
					{
						Key:   "environment",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("prod"),
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := jobMatchesFilter(tt.job, tt.filter)
			assert.Equal(t, tt.want, result)
		})
	}
}
