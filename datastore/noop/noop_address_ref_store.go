package noop

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

// NoopAddressRefStore is a no-op implementation of the AddressRefStore interface that returns errors for all operations.
var _ datastore.AddressRefStore = &NoopAddressRefStore{}

type NoopAddressRefStore struct{}

// NewNoopAddressRefStore creates a new instance of NoopAddressRefStore.
func NewNoopAddressRefStore() *NoopAddressRefStore {
	return &NoopAddressRefStore{}
}

// Get returns an error as this is a no-op implementation.
func (s *NoopAddressRefStore) Get(key datastore.AddressRefKey) (datastore.AddressRef, error) {
	return datastore.AddressRef{}, ErrNoopNotSupported
}

// Fetch returns an error as this is a no-op implementation.
func (s *NoopAddressRefStore) Fetch() ([]datastore.AddressRef, error) {
	return nil, ErrNoopNotSupported
}

// Filter returns an empty slice as this is a no-op implementation.
func (s *NoopAddressRefStore) Filter(filters ...datastore.FilterFunc[datastore.AddressRefKey, datastore.AddressRef]) []datastore.AddressRef {
	return []datastore.AddressRef{}
}
