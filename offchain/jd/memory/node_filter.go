package memory

import (
	"slices"
	"strings"

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
	nodeLabels := make(map[string]string)
	for _, label := range node.Labels {
		if label.Value != nil {
			nodeLabels[label.Key] = *label.Value
		}
	}

	// Check if the selector key exists in the node's labels
	nodeValue, hasKey := nodeLabels[selector.Key]

	switch selector.Op {
	case ptypes.SelectorOp_EQ:
		// Equality check
		if selector.Value == nil {
			return false
		}

		return hasKey && nodeValue == *selector.Value

	case ptypes.SelectorOp_NOT_EQ:
		// Not equal check
		if selector.Value == nil {
			return false
		}

		return hasKey && nodeValue != *selector.Value

	case ptypes.SelectorOp_IN:
		// IN operation - check if node value is in the selector values
		if selector.Value == nil {
			return false
		}
		if !hasKey {
			return false
		}

		// Parse comma-separated values
		values := strings.Split(*selector.Value, ",")
		for _, value := range values {
			if strings.TrimSpace(value) == nodeValue {
				return true
			}
		}

		return false

	case ptypes.SelectorOp_NOT_IN:
		// NOT IN operation - check if node value is not in the selector values
		if selector.Value == nil {
			return false
		}
		if !hasKey {
			return true // Key doesn't exist, so it's not in the list
		}

		// Parse comma-separated values
		values := strings.Split(*selector.Value, ",")
		for _, value := range values {
			if strings.TrimSpace(value) == nodeValue {
				return false // Found in the list, so NOT_IN is false
			}
		}

		return true // Not found in the list, so NOT_IN is true

	case ptypes.SelectorOp_EXIST:
		// Check if the key exists (regardless of value)
		return hasKey

	case ptypes.SelectorOp_NOT_EXIST:
		// Check if the key does not exist
		return !hasKey

	default:
		// Unknown operation, default to false
		return false
	}
}
