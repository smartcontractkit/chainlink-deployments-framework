package datastore

import (
	"encoding/json"
	"slices"
)

// RawMetadata wraps raw JSON and implements CustomMetadata.
type RawMetadata struct {
	raw json.RawMessage
}

func (d RawMetadata) Clone() CustomMetadata {
	return RawMetadata{raw: slices.Clone(d.raw)}
}
func (d RawMetadata) MarshalJSON() ([]byte, error) {
	return d.raw, nil
}

func (d RawMetadata) Raw() json.RawMessage {
	return d.raw
}
