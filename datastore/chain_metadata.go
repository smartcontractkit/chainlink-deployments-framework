package datastore

import "errors"

var ErrChainMetadataNotFound = errors.New("no chain metadata record can be found for the provided key")
var ErrChainMetadataExists = errors.New("a chain metadata record with the supplied key already exists")

// ChainMetadata implements the UniqueRecord interface
var _ UniqueRecord[ChainMetadataKey, ChainMetadata] = ChainMetadata{}

// ChainMetadata is a struct that holds the metadata for a specific chain.
// It implements the UniqueRecord interface and is used to store chain metadata in the datastore.
// NOTE: Metadata can be of any type. To convert from any to a specific type, use the utility method As.
type ChainMetadata struct {
	// ChainSelector refers to the chain associated with the metadata.
	ChainSelector uint64 `json:"chainSelector"`
	// Metadata is the metadata associated with the chain.
	Metadata any `json:"metadata"`
}

// Clone creates a copy of the ChainMetadata.
func (r ChainMetadata) Clone() (ChainMetadata, error) {
	metaClone, err := clone(r.Metadata)
	if err != nil {
		return ChainMetadata{}, err
	}

	return ChainMetadata{
		ChainSelector: r.ChainSelector,
		Metadata:      metaClone,
	}, nil
}

// Key returns the ChainMetadataKey for the ChainMetadata.
// It is used to uniquely identify the chain metadata in the datastore.
func (r ChainMetadata) Key() ChainMetadataKey {
	return NewChainMetadataKey(r.ChainSelector)
}
