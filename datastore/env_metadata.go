package datastore

import "errors"

var ErrEnvMetadataNotSet = errors.New("no environment metadata set")

// EnvMetadata is a struct that holds the metadata for a domain and environment.
// NOTE: Metadata can be of any type. To convert from any to a specific type, use the utility method As.
type EnvMetadata struct {
	// Metadata is the metadata associated with the domain and environment.
	Metadata any `json:"metadata"`
}

// Clone creates a copy of the EnvMetadata.
func (r EnvMetadata) Clone() (EnvMetadata, error) {
	metaClone, err := clone(r.Metadata)
	if err != nil {
		// If cloning fails, we return an empty EnvMetadata with the error.
		return EnvMetadata{}, err
	}

	return EnvMetadata{
		Metadata: metaClone,
	}, nil
}
