package memory

import (
	"strings"

	"github.com/smartcontractkit/chainlink-protos/job-distributor/v1/shared/ptypes"
)

// matchesSelector checks if a set of labels matches a specific selector.
// This is a generic function that can be used for any object with labels.
func matchesSelector(labels map[string]*string, selector *ptypes.Selector) bool {
	// Check if the selector key exists in the labels
	labelValuePtr, hasKey := labels[selector.Key]

	// If key doesn't exist or label value is nil, we need to handle it appropriately
	if !hasKey || labelValuePtr == nil {
		// For EXIST/NOT_EXIST operations, we only care about key existence
		switch selector.Op {
		case ptypes.SelectorOp_EXIST:
			return hasKey
		case ptypes.SelectorOp_NOT_EXIST:
			return !hasKey
		case ptypes.SelectorOp_EQ, ptypes.SelectorOp_NOT_EQ, ptypes.SelectorOp_IN, ptypes.SelectorOp_NOT_IN:
			// For other operations, if key doesn't exist or value is nil, return false
			return false
		default:
			// Unknown operation, default to false
			return false
		}
	}

	labelValue := *labelValuePtr

	switch selector.Op {
	case ptypes.SelectorOp_EQ:
		// Equality check
		if selector.Value == nil {
			return false
		}

		return labelValue == *selector.Value

	case ptypes.SelectorOp_NOT_EQ:
		// Not equal check
		if selector.Value == nil {
			return false
		}

		return labelValue != *selector.Value

	case ptypes.SelectorOp_IN:
		// IN operation - check if label value is in the selector values
		if selector.Value == nil {
			return false
		}

		// Parse comma-separated values
		values := strings.Split(*selector.Value, ",")
		for _, value := range values {
			if strings.TrimSpace(value) == labelValue {
				return true
			}
		}

		return false

	case ptypes.SelectorOp_NOT_IN:
		// NOT IN operation - check if label value is not in the selector values
		if selector.Value == nil {
			return false
		}

		// Parse comma-separated values
		values := strings.Split(*selector.Value, ",")
		for _, value := range values {
			if strings.TrimSpace(value) == labelValue {
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
