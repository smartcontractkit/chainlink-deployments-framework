package datastore

import (
	"encoding/json"
	"errors"
)

var ErrEnvMetadataNotSet = errors.New("no environment metadata set")

type EnvMetadata struct {
	Metadata CustomMetadata `json:"metadata"`
}

// Clone creates a copy of the EnvMetadata.
// The Metadata field is cloned using the Clone method of the Cloneable interface.
func (r EnvMetadata) Clone() EnvMetadata {
	return EnvMetadata{
		Metadata: r.Metadata.Clone(),
	}
}

// MarshalJSON outputs Metadata as raw JSON
func (e EnvMetadata) MarshalJSON() ([]byte, error) {
	type alias EnvMetadata
	tmp := struct {
		Metadata json.RawMessage `json:"metadata"`
		alias
	}{
		alias: (alias)(e),
	}
	if e.Metadata != nil {
		b, err := json.Marshal(e.Metadata)
		if err != nil {
			return nil, err
		}
		tmp.Metadata = b
	}
	return json.Marshal(tmp)
}

// UnmarshalJSON sets Metadata from raw JSON
func (e *EnvMetadata) UnmarshalJSON(data []byte) error {
	type alias EnvMetadata
	tmp := struct {
		Metadata json.RawMessage `json:"metadata"`
		*alias
	}{
		alias: (*alias)(e),
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	e.Metadata = RawMetadata{raw: tmp.Metadata}
	return nil
}
