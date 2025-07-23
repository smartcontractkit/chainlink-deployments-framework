package noop

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

// NoopContractMetadataStore is a no-op implementation of the ContractMetadataStore interface that returns errors for all operations.
var _ datastore.ContractMetadataStore = &NoopContractMetadataStore{}

type NoopContractMetadataStore struct{}

// NewNoopContractMetadataStore creates a new instance of NoopContractMetadataStore.
func NewNoopContractMetadataStore() *NoopContractMetadataStore {
	return &NoopContractMetadataStore{}
}

// Get returns an error as this is a no-op implementation.
func (s *NoopContractMetadataStore) Get(key datastore.ContractMetadataKey) (datastore.ContractMetadata, error) {
	return datastore.ContractMetadata{}, ErrNoopNotSupported
}

// Fetch returns an error as this is a no-op implementation.
func (s *NoopContractMetadataStore) Fetch() ([]datastore.ContractMetadata, error) {
	return nil, ErrNoopNotSupported
}

// Filter returns an empty slice as this is a no-op implementation.
func (s *NoopContractMetadataStore) Filter(filters ...datastore.FilterFunc[datastore.ContractMetadataKey, datastore.ContractMetadata]) []datastore.ContractMetadata {
	return []datastore.ContractMetadata{}
}
