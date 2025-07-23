package noop

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

// NoopChainMetadataStore is a no-op implementation of the ChainMetadataStore interface that returns errors for all operations.
var _ datastore.ChainMetadataStore = &NoopChainMetadataStore{}

type NoopChainMetadataStore struct{}

// NewNoopChainMetadataStore creates a new instance of NoopChainMetadataStore.
func NewNoopChainMetadataStore() *NoopChainMetadataStore {
	return &NoopChainMetadataStore{}
}

// Get returns an error as this is a no-op implementation.
func (s *NoopChainMetadataStore) Get(key datastore.ChainMetadataKey) (datastore.ChainMetadata, error) {
	return datastore.ChainMetadata{}, ErrNoopNotSupported
}

// Fetch returns an error as this is a no-op implementation.
func (s *NoopChainMetadataStore) Fetch() ([]datastore.ChainMetadata, error) {
	return nil, ErrNoopNotSupported
}

// Filter returns an empty slice as this is a no-op implementation.
func (s *NoopChainMetadataStore) Filter(filters ...datastore.FilterFunc[datastore.ChainMetadataKey, datastore.ChainMetadata]) []datastore.ChainMetadata {
	return []datastore.ChainMetadata{}
}
