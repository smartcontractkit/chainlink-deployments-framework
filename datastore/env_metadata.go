package datastore

import "errors"

var ErrEnvMetadataNotSet = errors.New("no environment metadata set")

// EnvMetadata implements the Record interface
var _ Record[EnvMetadata[DefaultMetadata]] = EnvMetadata[DefaultMetadata]{}

type EnvMetadata[M Cloneable[M]] struct {
	// Metadata is the metadata associated with the domain and environment.
	// It is a generic type that can be of any type that implements the Cloneable interface.
	Metadata M `json:"metadata"`
}

// Clone creates a copy of the EnvMetadata.
// The Metadata field is cloned using the Clone method of the Cloneable interface.
func (r EnvMetadata[M]) Clone() EnvMetadata[M] {
	return EnvMetadata[M]{
		Metadata: r.Metadata.Clone(),
	}
}
