package memory

import (
	"slices"

	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
	"github.com/smartcontractkit/chainlink-protos/job-distributor/v1/shared/ptypes"
)

// applyNodeFilter applies the filter to the list of nodes and returns the filtered results.
func applyNodeFilter(
	nodes []*nodev1.Node, filter *nodev1.ListNodesRequest_Filter,
) []*nodev1.Node {
	var filtered []*nodev1.Node

	for _, node := range nodes {
		if nodeMatchesFilter(node, filter) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// nodeMatchesFilter checks if a node matches the given filter criteria.
func nodeMatchesFilter(node *nodev1.Node, filter *nodev1.ListNodesRequest_Filter) bool {
	// Check ids
	if len(filter.Ids) > 0 {
		if !nodeMatchesIds(node, filter.Ids) {
			return false
		}
	}

	// Check enabled state
	if !nodeMatchesEnabledState(node, filter.Enabled) {
		return false
	}

	// Check selectors
	if len(filter.Selectors) > 0 {
		for _, selector := range filter.Selectors {
			if !nodeMatchesSelector(node, selector) {
				return false
			}
		}
	}

	return true
}

// nodeMatchesIds checks if a node's ID is in the provided list of IDs.
func nodeMatchesIds(node *nodev1.Node, ids []string) bool {
	return slices.Contains(ids, node.Id)
}

// nodeMatchesEnabledState checks if a node matches the enabled state filter.
// ENABLE_STATE_ENABLED: filter for enabled nodes only
// ENABLE_STATE_DISABLED: filter for disabled nodes only
// ENABLE_STATE_UNSPECIFIED: no filtering (default behavior when not set)
func nodeMatchesEnabledState(node *nodev1.Node, enabled nodev1.EnableState) bool {
	switch enabled {
	case nodev1.EnableState_ENABLE_STATE_ENABLED:
		return node.IsEnabled
	case nodev1.EnableState_ENABLE_STATE_DISABLED:
		return !node.IsEnabled
	case nodev1.EnableState_ENABLE_STATE_UNSPECIFIED:
		// No filtering by enabled status
		return true
	default:
		// Unknown state, default to true (no filtering)
		return true
	}
}

// nodeMatchesSelector checks if a node matches a specific selector.
func nodeMatchesSelector(node *nodev1.Node, selector *ptypes.Selector) bool {
	// Get the node's labels as a map for easier lookup
	nodeLabels := make(map[string]*string)
	for _, label := range node.Labels {
		nodeLabels[label.Key] = label.Value
	}

	return matchesSelector(nodeLabels, selector)
}
