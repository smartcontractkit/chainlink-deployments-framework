package datastore

import (
	"context"
	"errors"
	"fmt"
)

// MergeDataStoreToCatalog merges data from a source DataStore (either full local state or changeset-specific)
// into a remote CatalogStore within a transaction. This ensures atomic updates - either all data is
// successfully merged or the entire operation is rolled back on failure.
//
// This function serves two purposes:
// 1. Initial sync: sync entire local datastore to catalog (full state push)
// 2. Ongoing operations: merge changeset artifacts into catalog (incremental updates)
//
// The operation is transactional to prevent partial failures that could lead to data inconsistencies.
func MergeDataStoreToCatalog(ctx context.Context, sourceDS DataStore, catalog CatalogStore) error {
	return catalog.WithTransaction(ctx, func(ctx context.Context, txCatalog BaseCatalogStore) error {
		// Merge all address references to the catalog
		addressRefs, err := sourceDS.Addresses().Fetch()
		if err != nil {
			return fmt.Errorf("failed to fetch address references from source store: %w", err)
		}

		for _, ref := range addressRefs {
			if upsertErr := txCatalog.Addresses().Upsert(ctx, ref); upsertErr != nil {
				return fmt.Errorf("failed to upsert address reference to catalog: %w", upsertErr)
			}
		}

		// Merge all chain metadata to the catalog
		chainMetadata, err := sourceDS.ChainMetadata().Fetch()
		if err != nil {
			return fmt.Errorf("failed to fetch chain metadata from source store: %w", err)
		}

		for _, metadata := range chainMetadata {
			key := NewChainMetadataKey(metadata.ChainSelector)
			if upsertErr := txCatalog.ChainMetadata().Upsert(ctx, key, metadata.Metadata); upsertErr != nil {
				return fmt.Errorf("failed to upsert chain metadata to catalog: %w", upsertErr)
			}
		}

		// Merge all contract metadata to the catalog
		contractMetadata, err := sourceDS.ContractMetadata().Fetch()
		if err != nil {
			return fmt.Errorf("failed to fetch contract metadata from source store: %w", err)
		}

		for _, metadata := range contractMetadata {
			key := NewContractMetadataKey(metadata.ChainSelector, metadata.Address)
			if upsertErr := txCatalog.ContractMetadata().Upsert(ctx, key, metadata.Metadata); upsertErr != nil {
				return fmt.Errorf("failed to upsert contract metadata to catalog: %w", upsertErr)
			}
		}

		// Merge environment metadata to the catalog
		envMetadata, err := sourceDS.EnvMetadata().Get()
		if err != nil {
			if !errors.Is(err, ErrEnvMetadataNotSet) {
				return fmt.Errorf("failed to fetch environment metadata from source store: %w", err)
			}

			return nil
		}

		// Merge the environment metadata (upsert semantics)
		if setErr := txCatalog.EnvMetadata().Set(ctx, envMetadata.Metadata); setErr != nil {
			return fmt.Errorf("failed to set environment metadata in catalog: %w", setErr)
		}

		return nil
	})
}
