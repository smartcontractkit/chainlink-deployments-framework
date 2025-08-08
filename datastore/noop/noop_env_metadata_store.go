package noop

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

// NoopEnvMetadataStore is a no-op implementation of the EnvMetadataStore interface that returns errors for all operations.
var _ datastore.EnvMetadataStore = &NoopEnvMetadataStore{}

type NoopEnvMetadataStore struct{}

// NewNoopEnvMetadataStore creates a new instance of NoopEnvMetadataStore.
func NewNoopEnvMetadataStore() *NoopEnvMetadataStore {
	return &NoopEnvMetadataStore{}
}

// Get returns an error as this is a no-op implementation.
func (s *NoopEnvMetadataStore) Get() (datastore.EnvMetadata, error) {
	return datastore.EnvMetadata{}, ErrNoopNotSupported
}
