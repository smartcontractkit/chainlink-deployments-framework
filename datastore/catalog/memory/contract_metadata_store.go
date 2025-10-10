package memory

import (
	"context"
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

type memoryContractMetadataStore struct {
	storage *memoryStorage
}

// Ensure memoryContractMetadataStore implements the V2 interface
var _ datastore.MutableStoreV2[datastore.ContractMetadataKey, datastore.ContractMetadata] = &memoryContractMetadataStore{}

func newCatalogContractMetadataStore(storage *memoryStorage) *memoryContractMetadataStore {
	return &memoryContractMetadataStore{
		storage: storage,
	}
}

func (s *memoryContractMetadataStore) Get(ctx context.Context, key datastore.ContractMetadataKey, options ...datastore.GetOption) (datastore.ContractMetadata, error) {
	ignoreTransactions := false
	for _, option := range options {
		switch option {
		case datastore.IgnoreTransactionsGetOption:
			ignoreTransactions = true
		}
	}

	compositeKey := contractMetadataKey(key.ChainSelector(), key.Address())

	return s.storage.getContractMetadata(ctx, compositeKey, ignoreTransactions)
}

// Fetch returns a copy of all ContractMetadata in the catalog.
func (s *memoryContractMetadataStore) Fetch(ctx context.Context) ([]datastore.ContractMetadata, error) {
	return s.storage.getAllContractMetadata(ctx)
}

// Filter returns a copy of all ContractMetadata in the catalog that match the provided filter.
// Filters are applied in the order they are provided.
// If no filters are provided, all records are returned.
func (s *memoryContractMetadataStore) Filter(
	ctx context.Context,
	filters ...datastore.FilterFunc[datastore.ContractMetadataKey, datastore.ContractMetadata],
) ([]datastore.ContractMetadata, error) {
	// First, fetch all records from the catalog
	records, err := s.Fetch(ctx)
	if err != nil {
		// In case of error, return empty slice
		return []datastore.ContractMetadata{}, fmt.Errorf("failed to fetch records: %w", err)
	}

	// Apply each filter in sequence
	for _, filter := range filters {
		records = filter(records)
	}

	return records, nil
}

func (s *memoryContractMetadataStore) Add(ctx context.Context, r datastore.ContractMetadata) error {
	compositeKey := contractMetadataKey(r.ChainSelector, r.Address)

	// Check if the record already exists
	_, err := s.storage.getContractMetadata(ctx, compositeKey, false)
	if err == nil {
		return errors.New("contract metadata already exists")
	}
	if !errors.Is(err, datastore.ErrContractMetadataNotFound) {
		return err
	}

	return s.storage.setContractMetadata(ctx, compositeKey, r)
}

func (s *memoryContractMetadataStore) Upsert(ctx context.Context, key datastore.ContractMetadataKey, metadata any, opts ...datastore.UpdateOption) error {
	// Build options with defaults
	options := &datastore.UpdateOptions{
		Updater: datastore.IdentityUpdaterF, // default updater
	}

	// Apply user-provided options
	for _, opt := range opts {
		opt(options)
	}

	// Get current record for merging (if it exists)
	currentRecord, err := s.Get(ctx, key)
	if err != nil && !errors.Is(err, datastore.ErrContractMetadataNotFound) {
		return fmt.Errorf("failed to get current record for upsert: %w", err)
	}

	var finalMetadata any
	if errors.Is(err, datastore.ErrContractMetadataNotFound) {
		// Record doesn't exist, use the provided metadata directly
		finalMetadata = metadata
	} else {
		// Record exists, apply the updater to merge with existing metadata
		finalMetadata, err = options.Updater(currentRecord.Metadata, metadata)
		if err != nil {
			return fmt.Errorf("failed to apply metadata updater: %w", err)
		}
	}

	// Create record with final metadata
	record := datastore.ContractMetadata{
		ChainSelector: key.ChainSelector(),
		Address:       key.Address(),
		Metadata:      finalMetadata,
	}

	compositeKey := contractMetadataKey(key.ChainSelector(), key.Address())

	return s.storage.setContractMetadata(ctx, compositeKey, record)
}

func (s *memoryContractMetadataStore) Update(ctx context.Context, key datastore.ContractMetadataKey, metadata any, opts ...datastore.UpdateOption) error {
	// Build options with defaults
	options := &datastore.UpdateOptions{
		Updater: datastore.IdentityUpdaterF, // default updater
	}

	// Apply user-provided options
	for _, opt := range opts {
		opt(options)
	}

	// Get current record - it must exist for update
	currentRecord, err := s.Get(ctx, key)
	if err != nil {
		if errors.Is(err, datastore.ErrContractMetadataNotFound) {
			return datastore.ErrContractMetadataNotFound
		}

		return fmt.Errorf("failed to get current record for update: %w", err)
	}

	// Apply the updater (either default or custom)
	finalMetadata, updateErr := options.Updater(currentRecord.Metadata, metadata)
	if updateErr != nil {
		return fmt.Errorf("failed to apply metadata updater: %w", updateErr)
	}

	// Create record with final metadata
	record := datastore.ContractMetadata{
		ChainSelector: key.ChainSelector(),
		Address:       key.Address(),
		Metadata:      finalMetadata,
	}

	compositeKey := contractMetadataKey(key.ChainSelector(), key.Address())

	return s.storage.setContractMetadata(ctx, compositeKey, record)
}

func (s *memoryContractMetadataStore) Delete(_ context.Context, _ datastore.ContractMetadataKey) error {
	// The catalog API does not support delete operations
	// This is intentional as catalogs are typically immutable reference stores
	return errors.New("delete operation not supported for catalog contract metadata store")
}
