package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
	"github.com/smartcontractkit/chainlink-protos/job-distributor/v1/shared/ptypes"

	"github.com/smartcontractkit/chainlink-deployments-framework/internal/pointer"
)

func TestApplyNodeFilter(t *testing.T) {
	t.Parallel()

	// Create test nodes
	nodes := []*nodev1.Node{
		{
			Id:        "node-1",
			Name:      "Node 1",
			IsEnabled: true,
			Labels: []*ptypes.Label{
				{Key: "environment", Value: pointer.To("prod")},
			},
		},
		{
			Id:        "node-2",
			Name:      "Node 2",
			IsEnabled: false,
			Labels: []*ptypes.Label{
				{Key: "environment", Value: pointer.To("test")},
			},
		},
		{
			Id:        "node-3",
			Name:      "Node 3",
			IsEnabled: true,
			Labels: []*ptypes.Label{
				{Key: "environment", Value: pointer.To("prod")},
			},
		},
	}

	tests := []struct {
		name     string
		nodes    []*nodev1.Node
		filter   *nodev1.ListNodesRequest_Filter
		expected []string // Expected node IDs
	}{
		{
			name:     "no filter - return all nodes",
			nodes:    nodes,
			filter:   &nodev1.ListNodesRequest_Filter{},
			expected: []string{"node-1", "node-2", "node-3"},
		},
		{
			name:  "filter by id",
			nodes: nodes,
			filter: &nodev1.ListNodesRequest_Filter{
				Ids: []string{"node-1", "node-2"},
			},
			expected: []string{"node-1", "node-2"},
		},
		{
			name:  "filter by enabled state",
			nodes: nodes,
			filter: &nodev1.ListNodesRequest_Filter{
				Enabled: nodev1.EnableState_ENABLE_STATE_ENABLED,
			},
			expected: []string{"node-1", "node-3"},
		},
		{
			name:  "filter by label",
			nodes: nodes,
			filter: &nodev1.ListNodesRequest_Filter{
				Selectors: []*ptypes.Selector{
					{
						Key:   "environment",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("prod"),
					},
				},
			},
			expected: []string{"node-1", "node-3"},
		},
		{
			name:  "combined filters",
			nodes: nodes,
			filter: &nodev1.ListNodesRequest_Filter{
				Ids:     []string{"node-1", "node-2", "node-3"},
				Enabled: nodev1.EnableState_ENABLE_STATE_ENABLED,
				Selectors: []*ptypes.Selector{
					{
						Key:   "environment",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("prod"),
					},
				},
			},
			expected: []string{"node-1", "node-3"},
		},
		{
			name:  "empty nodes list",
			nodes: []*nodev1.Node{},
			filter: &nodev1.ListNodesRequest_Filter{
				Enabled: nodev1.EnableState_ENABLE_STATE_ENABLED,
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := applyNodeFilter(tt.nodes, tt.filter)

			// Extract node IDs for comparison
			resultIds := make([]string, len(result))
			for i, node := range result {
				resultIds[i] = node.Id
			}

			require.ElementsMatch(t, tt.expected, resultIds)
		})
	}
}

func TestNodeMatchesIds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		node *nodev1.Node
		ids  []string
		want bool
	}{
		{
			name: "node id matches",
			node: &nodev1.Node{Id: "node-1"},
			ids:  []string{"node-1", "node-2"},
			want: true,
		},
		{
			name: "node id does not match",
			node: &nodev1.Node{Id: "node-3"},
			ids:  []string{"node-1", "node-2"},
			want: false,
		},
		{
			name: "empty ids list",
			node: &nodev1.Node{Id: "node-1"},
			ids:  []string{},
			want: false,
		},
		{
			name: "single id match",
			node: &nodev1.Node{Id: "node-1"},
			ids:  []string{"node-1"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := nodeMatchesIds(tt.node, tt.ids)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestNodeMatchesEnabledState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		node    *nodev1.Node
		enabled nodev1.EnableState
		want    bool
	}{
		{
			name:    "enabled node with ENABLE_STATE_ENABLED",
			node:    &nodev1.Node{IsEnabled: true},
			enabled: nodev1.EnableState_ENABLE_STATE_ENABLED,
			want:    true,
		},
		{
			name:    "disabled node with ENABLE_STATE_ENABLED",
			node:    &nodev1.Node{IsEnabled: false},
			enabled: nodev1.EnableState_ENABLE_STATE_ENABLED,
			want:    false,
		},
		{
			name:    "enabled node with ENABLE_STATE_DISABLED",
			node:    &nodev1.Node{IsEnabled: true},
			enabled: nodev1.EnableState_ENABLE_STATE_DISABLED,
			want:    false,
		},
		{
			name:    "disabled node with ENABLE_STATE_DISABLED",
			node:    &nodev1.Node{IsEnabled: false},
			enabled: nodev1.EnableState_ENABLE_STATE_DISABLED,
			want:    true,
		},
		{
			name:    "enabled node with ENABLE_STATE_UNSPECIFIED",
			node:    &nodev1.Node{IsEnabled: true},
			enabled: nodev1.EnableState_ENABLE_STATE_UNSPECIFIED,
			want:    true,
		},
		{
			name:    "disabled node with ENABLE_STATE_UNSPECIFIED",
			node:    &nodev1.Node{IsEnabled: false},
			enabled: nodev1.EnableState_ENABLE_STATE_UNSPECIFIED,
			want:    true,
		},
		{
			name:    "enabled node with unknown state",
			node:    &nodev1.Node{IsEnabled: true},
			enabled: nodev1.EnableState(999), // Unknown state
			want:    true,
		},
		{
			name:    "disabled node with unknown state",
			node:    &nodev1.Node{IsEnabled: false},
			enabled: nodev1.EnableState(999), // Unknown state
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := nodeMatchesEnabledState(tt.node, tt.enabled)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestNodeMatchesSelector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		node     *nodev1.Node
		selector *ptypes.Selector
		want     bool
	}{
		{
			name: "basic selector matching",
			node: &nodev1.Node{
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("prod")},
					{Key: "region", Value: pointer.To("us-east-1")},
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
			name: "node with nil label value",
			node: &nodev1.Node{
				Labels: []*ptypes.Label{
					{Key: "environment", Value: nil},
					{Key: "region", Value: pointer.To("us-east-1")},
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
			name: "node with empty labels",
			node: &nodev1.Node{
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
			name: "node with nil labels",
			node: &nodev1.Node{
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
			result := nodeMatchesSelector(tt.node, tt.selector)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestNodeMatchesFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		node   *nodev1.Node
		filter *nodev1.ListNodesRequest_Filter
		want   bool
	}{
		{
			name: "no filter - should match",
			node: &nodev1.Node{
				Id:        "node-1",
				IsEnabled: true,
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("prod")},
				},
			},
			filter: &nodev1.ListNodesRequest_Filter{},
			want:   true,
		},
		{
			name: "id filter - matching id",
			node: &nodev1.Node{
				Id:        "node-1",
				IsEnabled: true,
			},
			filter: &nodev1.ListNodesRequest_Filter{
				Ids: []string{"node-1", "node-2"},
			},
			want: true,
		},
		{
			name: "id filter - non-matching id",
			node: &nodev1.Node{
				Id:        "node-3",
				IsEnabled: true,
			},
			filter: &nodev1.ListNodesRequest_Filter{
				Ids: []string{"node-1", "node-2"},
			},
			want: false,
		},
		{
			name: "enabled filter - enabled node",
			node: &nodev1.Node{
				Id:        "node-1",
				IsEnabled: true,
			},
			filter: &nodev1.ListNodesRequest_Filter{
				Enabled: nodev1.EnableState_ENABLE_STATE_ENABLED,
			},
			want: true,
		},
		{
			name: "enabled filter - disabled node",
			node: &nodev1.Node{
				Id:        "node-1",
				IsEnabled: false,
			},
			filter: &nodev1.ListNodesRequest_Filter{
				Enabled: nodev1.EnableState_ENABLE_STATE_ENABLED,
			},
			want: false,
		},
		{
			name: "disabled filter - disabled node",
			node: &nodev1.Node{
				Id:        "node-1",
				IsEnabled: false,
			},
			filter: &nodev1.ListNodesRequest_Filter{
				Enabled: nodev1.EnableState_ENABLE_STATE_DISABLED,
			},
			want: true,
		},
		{
			name: "disabled filter - enabled node",
			node: &nodev1.Node{
				Id:        "node-1",
				IsEnabled: true,
			},
			filter: &nodev1.ListNodesRequest_Filter{
				Enabled: nodev1.EnableState_ENABLE_STATE_DISABLED,
			},
			want: false,
		},
		{
			name: "selector filter - matching selector",
			node: &nodev1.Node{
				Id:        "node-1",
				IsEnabled: true,
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("prod")},
				},
			},
			filter: &nodev1.ListNodesRequest_Filter{
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
			node: &nodev1.Node{
				Id:        "node-1",
				IsEnabled: true,
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("test")},
				},
			},
			filter: &nodev1.ListNodesRequest_Filter{
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
			node: &nodev1.Node{
				Id:        "node-1",
				IsEnabled: true,
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("prod")},
					{Key: "region", Value: pointer.To("us-east-1")},
				},
			},
			filter: &nodev1.ListNodesRequest_Filter{
				Selectors: []*ptypes.Selector{
					{
						Key:   "environment",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("prod"),
					},
					{
						Key:   "region",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("us-east-1"),
					},
				},
			},
			want: true,
		},
		{
			name: "multiple selectors - one does not match",
			node: &nodev1.Node{
				Id:        "node-1",
				IsEnabled: true,
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("prod")},
					{Key: "region", Value: pointer.To("us-west-2")},
				},
			},
			filter: &nodev1.ListNodesRequest_Filter{
				Selectors: []*ptypes.Selector{
					{
						Key:   "environment",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("prod"),
					},
					{
						Key:   "region",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("us-east-1"),
					},
				},
			},
			want: false,
		},
		{
			name: "combined filters - all match",
			node: &nodev1.Node{
				Id:        "node-1",
				IsEnabled: true,
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("prod")},
				},
			},
			filter: &nodev1.ListNodesRequest_Filter{
				Ids:     []string{"node-1", "node-2"},
				Enabled: nodev1.EnableState_ENABLE_STATE_ENABLED,
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
			name: "combined filters - id does not match",
			node: &nodev1.Node{
				Id:        "node-3",
				IsEnabled: true,
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("prod")},
				},
			},
			filter: &nodev1.ListNodesRequest_Filter{
				Ids:     []string{"node-1", "node-2"},
				Enabled: nodev1.EnableState_ENABLE_STATE_ENABLED,
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
			name: "combined filters - enabled state does not match",
			node: &nodev1.Node{
				Id:        "node-1",
				IsEnabled: false,
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("prod")},
				},
			},
			filter: &nodev1.ListNodesRequest_Filter{
				Ids:     []string{"node-1", "node-2"},
				Enabled: nodev1.EnableState_ENABLE_STATE_ENABLED,
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
			node: &nodev1.Node{
				Id:        "node-1",
				IsEnabled: true,
				Labels: []*ptypes.Label{
					{Key: "environment", Value: pointer.To("test")},
				},
			},
			filter: &nodev1.ListNodesRequest_Filter{
				Ids:     []string{"node-1", "node-2"},
				Enabled: nodev1.EnableState_ENABLE_STATE_ENABLED,
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
			result := nodeMatchesFilter(tt.node, tt.filter)
			assert.Equal(t, tt.want, result)
		})
	}
}
