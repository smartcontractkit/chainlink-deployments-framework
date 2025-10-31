package datastore

import (
	"context"
	"errors"
	"fmt"
)

// SyncDataStoreToCatalog pushes all data from a local DataStore to a remote CatalogStore within
// a transaction. This ensures atomic updates - either all data is successfully synced or the
// entire operation is rolled back on failure.
//
// This function is the inverse of LoadDataStoreFromCatalog - while that function reads from
// catalog to local, this function writes from local to catalog.
func SyncDataStoreToCatalog(ctx context.Context, localDS DataStore, catalog CatalogStore) error {
	return catalog.WithTransaction(ctx, func(ctx context.Context, txCatalog BaseCatalogStore) error {
		// Sync all address references to the catalog
		addressRefs, err := localDS.Addresses().Fetch()
		if err != nil {
			return fmt.Errorf("failed to fetch address references from local store: %w", err)
		}

		for _, ref := range addressRefs {
			if upsertErr := txCatalog.Addresses().Upsert(ctx, ref); upsertErr != nil {
				return fmt.Errorf("failed to upsert address reference to catalog: %w", upsertErr)
			}
		}

		// Sync all chain metadata to the catalog
		chainMetadata, err := localDS.ChainMetadata().Fetch()
		if err != nil {
			return fmt.Errorf("failed to fetch chain metadata from local store: %w", err)
		}

		for _, metadata := range chainMetadata {
			key := NewChainMetadataKey(metadata.ChainSelector)
			if upsertErr := txCatalog.ChainMetadata().Upsert(ctx, key, metadata.Metadata); upsertErr != nil {
				return fmt.Errorf("failed to upsert chain metadata to catalog: %w", upsertErr)
			}
		}

		// Sync all contract metadata to the catalog
		contractMetadata, err := localDS.ContractMetadata().Fetch()
		if err != nil {
			return fmt.Errorf("failed to fetch contract metadata from local store: %w", err)
		}

		for _, metadata := range contractMetadata {
			key := NewContractMetadataKey(metadata.ChainSelector, metadata.Address)
			if upsertErr := txCatalog.ContractMetadata().Upsert(ctx, key, metadata.Metadata); upsertErr != nil {
				return fmt.Errorf("failed to upsert contract metadata to catalog: %w", upsertErr)
			}
		}

		// Sync environment metadata to the catalog
		envMetadata, err := localDS.EnvMetadata().Get()
		if err != nil {
			// EnvMetadata might not be set, which is okay
			if !errors.Is(err, ErrEnvMetadataNotSet) {
				return fmt.Errorf("failed to fetch environment metadata from local store: %w", err)
			}
			// If it's ErrEnvMetadataNotSet, skip syncing env metadata
			return nil
		}

		// Sync the environment metadata
		if setErr := txCatalog.EnvMetadata().Set(ctx, envMetadata.Metadata); setErr != nil {
			return fmt.Errorf("failed to set environment metadata in catalog: %w", setErr)
		}

		return nil
	})
}

// MergeDataStoreToCatalog merges data from a migration/changeset DataStore into the catalog
// within a transaction. This is used after a migration/changeset execution to persist new
// contract deployments and metadata to the catalog.
func MergeDataStoreToCatalog(ctx context.Context, migrationDS DataStore, catalog CatalogStore) error {
	return catalog.WithTransaction(ctx, func(ctx context.Context, txCatalog BaseCatalogStore) error {
		// Merge all address references to the catalog
		addressRefs, err := migrationDS.Addresses().Fetch()
		if err != nil {
			return fmt.Errorf("failed to fetch address references from migration store: %w", err)
		}

		for _, ref := range addressRefs {
			if upsertErr := txCatalog.Addresses().Upsert(ctx, ref); upsertErr != nil {
				return fmt.Errorf("failed to upsert address reference to catalog: %w", upsertErr)
			}
		}

		// Merge all chain metadata to the catalog
		chainMetadata, err := migrationDS.ChainMetadata().Fetch()
		if err != nil {
			return fmt.Errorf("failed to fetch chain metadata from migration store: %w", err)
		}

		for _, metadata := range chainMetadata {
			key := NewChainMetadataKey(metadata.ChainSelector)
			if upsertErr := txCatalog.ChainMetadata().Upsert(ctx, key, metadata.Metadata); upsertErr != nil {
				return fmt.Errorf("failed to upsert chain metadata to catalog: %w", upsertErr)
			}
		}

		// Merge all contract metadata to the catalog
		contractMetadata, err := migrationDS.ContractMetadata().Fetch()
		if err != nil {
			return fmt.Errorf("failed to fetch contract metadata from migration store: %w", err)
		}

		for _, metadata := range contractMetadata {
			key := NewContractMetadataKey(metadata.ChainSelector, metadata.Address)
			if upsertErr := txCatalog.ContractMetadata().Upsert(ctx, key, metadata.Metadata); upsertErr != nil {
				return fmt.Errorf("failed to upsert contract metadata to catalog: %w", upsertErr)
			}
		}

		// Merge environment metadata to the catalog
		envMetadata, err := migrationDS.EnvMetadata().Get()
		if err != nil {
			if !errors.Is(err, ErrEnvMetadataNotSet) {
				return fmt.Errorf("failed to fetch environment metadata from migration store: %w", err)
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
