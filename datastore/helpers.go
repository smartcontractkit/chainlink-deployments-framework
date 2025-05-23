package datastore

import (
	"encoding/json"
	"fmt"
)

func As[T CustomMetadata](m CustomMetadata) (T, error) {
	var zero T
	rw, ok := m.(RawMetadata)
	if !ok {
		return zero, fmt.Errorf("metadata is not RawMetadata, got %T", m)
	}
	var t T
	if err := json.Unmarshal(rw.Raw(), &t); err != nil {
		return zero, fmt.Errorf("failed to unmarshal to target type: %w", err)
	}
	return t, nil
}
