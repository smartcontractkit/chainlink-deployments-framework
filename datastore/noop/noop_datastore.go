package noop

import (
	"errors"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

var (
	// ErrNoopNotSupported is returned by all noop operations
	ErrNoopNotSupported = errors.New("operation not supported by noop data store")
)

// NoopDataStore is a no-op implementation of the DataStore interface that returns errors for all operations.
var _ datastore.DataStore = &NoopDataStore{}

type NoopDataStore struct {
	addressRefStore       *NoopAddressRefStore
	chainMetadataStore    *NoopChainMetadataStore
	contractMetadataStore *NoopContractMetadataStore
	envMetadataStore      *NoopEnvMetadataStore
}

// NewNoopDataStore creates a new instance of NoopDataStore.
func NewNoopDataStore() *NoopDataStore {
	return &NoopDataStore{
		addressRefStore:       NewNoopAddressRefStore(),
		chainMetadataStore:    NewNoopChainMetadataStore(),
		contractMetadataStore: NewNoopContractMetadataStore(),
		envMetadataStore:      NewNoopEnvMetadataStore(),
	}
}

// Addresses returns the NoopAddressRefStore.
func (s *NoopDataStore) Addresses() datastore.AddressRefStore {
	return s.addressRefStore
}

// ChainMetadata returns the NoopChainMetadataStore.
func (s *NoopDataStore) ChainMetadata() datastore.ChainMetadataStore {
	return s.chainMetadataStore
}

// ContractMetadata returns the NoopContractMetadataStore.
func (s *NoopDataStore) ContractMetadata() datastore.ContractMetadataStore {
	return s.contractMetadataStore
}

// EnvMetadata returns the NoopEnvMetadataStore.
func (s *NoopDataStore) EnvMetadata() datastore.EnvMetadataStore {
	return s.envMetadataStore
}
