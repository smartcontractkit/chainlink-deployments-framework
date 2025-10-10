package memory

import (
	"context"
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

type memoryAddressRefStore struct {
	storage *memoryStorage
}

// Ensure memoryAddressRefStore implements the V2 interface
var _ datastore.MutableRefStoreV2[datastore.AddressRefKey, datastore.AddressRef] = &memoryAddressRefStore{}

func newCatalogAddressRefStore(storage *memoryStorage) *memoryAddressRefStore {
	return &memoryAddressRefStore{
		storage: storage,
	}
}

func (s *memoryAddressRefStore) Get(ctx context.Context, key datastore.AddressRefKey, options ...datastore.GetOption) (datastore.AddressRef, error) {
	ignoreTransactions := false
	for _, option := range options {
		switch option {
		case datastore.IgnoreTransactionsGetOption:
			ignoreTransactions = true
		}
	}

	compositeKey := addressRefKey(key.ChainSelector(), key.Type().String(), key.Version().String(), key.Qualifier())

	return s.storage.getAddressRef(ctx, compositeKey, ignoreTransactions)
}

// Fetch returns a copy of all AddressRefs in the catalog.
func (s *memoryAddressRefStore) Fetch(ctx context.Context) ([]datastore.AddressRef, error) {
	return s.storage.getAllAddressRefs(ctx)
}

// Filter returns a copy of all AddressRef in the catalog that match the provided filter.
// Filters are applied in the order they are provided.
// If no filters are provided, all records are returned.
func (s *memoryAddressRefStore) Filter(
	ctx context.Context,
	filters ...datastore.FilterFunc[datastore.AddressRefKey, datastore.AddressRef],
) ([]datastore.AddressRef, error) {
	// First, fetch all records from the catalog
	records, err := s.Fetch(ctx)
	if err != nil {
		// In case of error, return empty slice
		// In a more robust implementation, you might want to log this error
		return []datastore.AddressRef{}, fmt.Errorf("failed to fetch records: %w", err)
	}

	// Apply each filter in sequence
	for _, filter := range filters {
		records = filter(records)
	}

	return records, nil
}

func (s *memoryAddressRefStore) Add(ctx context.Context, r datastore.AddressRef) error {
	compositeKey := addressRefKey(r.ChainSelector, r.Type.String(), r.Version.String(), r.Qualifier)

	// Check if the record already exists
	_, err := s.storage.getAddressRef(ctx, compositeKey, false)
	if err == nil {
		return errors.New("address reference already exists")
	}
	if !errors.Is(err, datastore.ErrAddressRefNotFound) {
		return err
	}

	return s.storage.setAddressRef(ctx, compositeKey, r)
}

func (s *memoryAddressRefStore) Upsert(ctx context.Context, r datastore.AddressRef) error {
	compositeKey := addressRefKey(r.ChainSelector, r.Type.String(), r.Version.String(), r.Qualifier)
	return s.storage.setAddressRef(ctx, compositeKey, r)
}

func (s *memoryAddressRefStore) Update(ctx context.Context, r datastore.AddressRef) error {
	compositeKey := addressRefKey(r.ChainSelector, r.Type.String(), r.Version.String(), r.Qualifier)

	// Check if the record exists first
	_, err := s.storage.getAddressRef(ctx, compositeKey, false)
	if err != nil {
		return err
	}

	return s.storage.setAddressRef(ctx, compositeKey, r)
}

func (s *memoryAddressRefStore) Delete(_ context.Context, _ datastore.AddressRefKey) error {
	// The catalog API does not support delete operations
	// This is intentional as catalogs are typically immutable reference stores
	return errors.New("delete operation not supported by catalog API")
}
