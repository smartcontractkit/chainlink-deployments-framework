package datastore

import "errors"

var ErrEnvMetadataNotSet = errors.New("no environment metadata set")

// EnvMetadata implements the Record interface
//var _ Record[EnvMetadata] = EnvMetadata{}

type EnvMetadata struct {
	// Metadata is the metadata associated with the domain and environment.
	// It is a generic type that can be of any type that implements the Cloneable interface.
	Metadata any `json:"metadata"`
}

// Clone creates a copy of the EnvMetadata.
// The Metadata field is cloned using the Clone method of the Cloneable interface.
func (r EnvMetadata) Clone() (EnvMetadata, error) {
	return AnyClone(r)
}
