package datastore

import (
	"encoding/json"
)

// As is a utility function that converts a source value of any type to a destination type T.
// It uses JSON marshaling and unmarshaling to perform the conversion.
func As[T any](src any) (T, error) {
	var zero T
	bytes, err := json.Marshal(src)
	if err != nil {
		return zero, err
	}

	var dst T
	err = json.Unmarshal(bytes, &dst)

	return dst, err
}
