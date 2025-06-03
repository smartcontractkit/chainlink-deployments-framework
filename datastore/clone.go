package datastore

import (
	"bytes"
	"encoding/json"
)

// clone creates a copy of the given value using JSON serialization.
// It returns the cloned value or an error if the cloning process fails.
func clone[T any](v T) (T, error) {
	var zero T
	b, err := json.Marshal(v)
	if err != nil {
		return zero, err
	}

	// Use a JSON decoder to handle numbers as json.Number
	// This allows us to preserve the original types of numbers during unmarshaling.
	// This is particularly useful for large integers that might not fit into standard int types.
	var clone T
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()
	if err = decoder.Decode(&clone); err != nil {
		return zero, err
	}

	return clone, err
}
