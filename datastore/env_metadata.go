package datastore

import "errors"

var ErrEnvMetadataNotSet = errors.New("no environment metadata set")

type EnvMetadata[M any] struct {
	// Metadata is the metadata associated with the domain and environment.
	// It is a generic type that can be of any type that implements the Cloneable interface.
	Metadata M `json:"metadata"`
}

// Clone creates a copy of the EnvMetadata.
// The Metadata field is cloned using the Clone method of the Cloneable interface.
func (r EnvMetadata[M]) Clone() (EnvMetadata[M], error) {
	metaClone, err := clone(r.Metadata)
	if err != nil {
		// If cloning fails, we return an empty EnvMetadata with the error.
		return EnvMetadata[M]{}, err
	}

	return EnvMetadata[M]{
		Metadata: metaClone,
	}, nil
}
