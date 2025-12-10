package memory

import (
	"context"
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

type memoryChainMetadataStore struct {
	storage *memoryStorage
}

// Ensure memoryChainMetadataStore implements the V2 interface
var _ datastore.MutableStoreV2[datastore.ChainMetadataKey, datastore.ChainMetadata] = &memoryChainMetadataStore{}

func newCatalogChainMetadataStore(storage *memoryStorage) *memoryChainMetadataStore {
	return &memoryChainMetadataStore{
		storage: storage,
	}
}

func (s *memoryChainMetadataStore) Get(ctx context.Context, key datastore.ChainMetadataKey, options ...datastore.GetOption) (datastore.ChainMetadata, error) {
	ignoreTransactions := false
	for _, option := range options {
		switch option {
		case datastore.IgnoreTransactionsGetOption:
			ignoreTransactions = true
		}
	}

	return s.storage.getChainMetadata(ctx, key.ChainSelector(), ignoreTransactions)
}

// Fetch returns a copy of all ChainMetadata in the catalog.
func (s *memoryChainMetadataStore) Fetch(ctx context.Context) ([]datastore.ChainMetadata, error) {
	return s.storage.getAllChainMetadata(ctx)
}

// Filter returns a copy of all ChainMetadata in the catalog that match the provided filter.
// Filters are applied in the order they are provided.
// If no filters are provided, all records are returned.
func (s *memoryChainMetadataStore) Filter(
	ctx context.Context,
	filters ...datastore.FilterFunc[datastore.ChainMetadataKey, datastore.ChainMetadata],
) ([]datastore.ChainMetadata, error) {
	// First, fetch all records from the catalog
	records, err := s.Fetch(ctx)
	if err != nil {
		// In case of error, return empty slice
		return []datastore.ChainMetadata{}, fmt.Errorf("failed to fetch records: %w", err)
	}

	// Apply each filter in sequence
	for _, filter := range filters {
		records = filter(records)
	}

	return records, nil
}

func (s *memoryChainMetadataStore) Add(ctx context.Context, r datastore.ChainMetadata) error {
	// Check if the record already exists
	_, err := s.storage.getChainMetadata(ctx, r.ChainSelector, false)
	if err == nil {
		return errors.New("chain metadata already exists")
	}
	if !errors.Is(err, datastore.ErrChainMetadataNotFound) {
		return err
	}

	return s.storage.setChainMetadata(ctx, r.ChainSelector, r)
}

func (s *memoryChainMetadataStore) Upsert(ctx context.Context, key datastore.ChainMetadataKey, metadata any, opts ...datastore.UpdateOption) error {
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
	if err != nil && !errors.Is(err, datastore.ErrChainMetadataNotFound) {
		return fmt.Errorf("failed to get current record for upsert: %w", err)
	}

	var finalMetadata any
	if errors.Is(err, datastore.ErrChainMetadataNotFound) {
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
	record := datastore.ChainMetadata{
		ChainSelector: key.ChainSelector(),
		Metadata:      finalMetadata,
	}

	return s.storage.setChainMetadata(ctx, key.ChainSelector(), record)
}

func (s *memoryChainMetadataStore) Update(ctx context.Context, key datastore.ChainMetadataKey, metadata any, opts ...datastore.UpdateOption) error {
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
		if errors.Is(err, datastore.ErrChainMetadataNotFound) {
			return datastore.ErrChainMetadataNotFound
		}

		return fmt.Errorf("failed to get current record for update: %w", err)
	}

	// Apply the updater (either default or custom)
	finalMetadata, updateErr := options.Updater(currentRecord.Metadata, metadata)
	if updateErr != nil {
		return fmt.Errorf("failed to apply metadata updater: %w", updateErr)
	}

	// Create record with final metadata
	record := datastore.ChainMetadata{
		ChainSelector: key.ChainSelector(),
		Metadata:      finalMetadata,
	}

	return s.storage.setChainMetadata(ctx, key.ChainSelector(), record)
}

func (s *memoryChainMetadataStore) Delete(_ context.Context, _ datastore.ChainMetadataKey) error {
	// The catalog API does not support delete operations
	// This is intentional as catalogs are typically immutable reference stores
	return errors.New("delete operation not supported for catalog chain metadata store")
}
