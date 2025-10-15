package memory

import (
	"context"
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

type memoryEnvMetadataStore struct {
	storage *memoryStorage
}

// Ensure memoryEnvMetadataStore implements the V2 interface
var _ datastore.MutableUnaryStoreV2[datastore.EnvMetadata] = &memoryEnvMetadataStore{}

func newCatalogEnvMetadataStore(storage *memoryStorage) *memoryEnvMetadataStore {
	return &memoryEnvMetadataStore{
		storage: storage,
	}
}

func (s *memoryEnvMetadataStore) Get(ctx context.Context, options ...datastore.GetOption) (datastore.EnvMetadata, error) {
	ignoreTransactions := false
	for _, option := range options {
		switch option {
		case datastore.IgnoreTransactionsGetOption:
			ignoreTransactions = true
		}
	}

	return s.storage.getEnvMetadata(ctx, ignoreTransactions)
}

func (s *memoryEnvMetadataStore) Set(ctx context.Context, metadata any, opts ...datastore.UpdateOption) error {
	// Build options with defaults
	options := &datastore.UpdateOptions{
		Updater: datastore.IdentityUpdaterF, // default updater
	}

	// Apply user-provided options
	for _, opt := range opts {
		opt(options)
	}

	// Get current record for merging (if it exists)
	currentRecord, err := s.Get(ctx)
	if err != nil && !errors.Is(err, datastore.ErrEnvMetadataNotSet) {
		return fmt.Errorf("failed to get current record for set: %w", err)
	}

	var finalMetadata any
	if errors.Is(err, datastore.ErrEnvMetadataNotSet) {
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
	record := datastore.EnvMetadata{
		Metadata: finalMetadata,
	}

	return s.storage.setEnvMetadata(ctx, record)
}
