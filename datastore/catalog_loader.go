package datastore

import (
	"context"
	"errors"
	"fmt"
)

// LoadDataStoreFromCatalog fetches all data from a remote CatalogStore and creates a local
// in-memory DataStore populated with that data. After loading, all operations are performed
// on the local DataStore without any remote calls.
func LoadDataStoreFromCatalog(ctx context.Context, catalog CatalogStore) (DataStore, error) {
	// Create a new mutable in-memory datastore
	memoryDS := NewMemoryDataStore()

	// Fetch all address references from the catalog
	addressRefs, err := catalog.Addresses().Fetch(ctx)
	if err != nil {
		// If no address refs found, treat as empty catalog (valid state)
		if !errors.Is(err, ErrAddressRefNotFound) {
			return nil, fmt.Errorf("failed to fetch address references from catalog: %w", err)
		}
		addressRefs = []AddressRef{} // Empty catalog is valid
	}

	// Populate the address ref store
	for _, ref := range addressRefs {
		if addErr := memoryDS.Addresses().Add(ref); addErr != nil {
			return nil, fmt.Errorf("failed to add address reference to local store: %w", addErr)
		}
	}

	// Fetch all chain metadata from the catalog
	chainMetadata, err := catalog.ChainMetadata().Fetch(ctx)
	if err != nil {
		// If no chain metadata found, treat as empty catalog (valid state)
		if !errors.Is(err, ErrChainMetadataNotFound) {
			return nil, fmt.Errorf("failed to fetch chain metadata from catalog: %w", err)
		}
		chainMetadata = []ChainMetadata{} // Empty catalog is valid
	}

	// Populate the chain metadata store
	for _, metadata := range chainMetadata {
		if addErr := memoryDS.ChainMetadata().Add(metadata); addErr != nil {
			return nil, fmt.Errorf("failed to add chain metadata to local store: %w", addErr)
		}
	}

	// Fetch all contract metadata from the catalog
	contractMetadata, err := catalog.ContractMetadata().Fetch(ctx)
	if err != nil {
		// If no contract metadata found, treat as empty catalog (valid state)
		if !errors.Is(err, ErrContractMetadataNotFound) {
			return nil, fmt.Errorf("failed to fetch contract metadata from catalog: %w", err)
		}
		contractMetadata = []ContractMetadata{} // Empty catalog is valid
	}

	// Populate the contract metadata store
	for _, metadata := range contractMetadata {
		if addErr := memoryDS.ContractMetadata().Add(metadata); addErr != nil {
			return nil, fmt.Errorf("failed to add contract metadata to local store: %w", addErr)
		}
	}

	// Fetch environment metadata from the catalog
	envMetadata, err := catalog.EnvMetadata().Get(ctx)
	if err != nil {
		// EnvMetadata might not exist, which is okay - ignore ErrEnvMetadataNotSet
		if !errors.Is(err, ErrEnvMetadataNotSet) {
			return nil, fmt.Errorf("failed to fetch environment metadata from catalog: %w", err)
		}
		// If it's ErrEnvMetadataNotSet, continue without error
	} else {
		// Populate the environment metadata store
		if setErr := memoryDS.EnvMetadata().Set(envMetadata); setErr != nil {
			return nil, fmt.Errorf("failed to set environment metadata in local store: %w", setErr)
		}
	}

	// Seal the datastore to make it read-only and return
	return memoryDS.Seal(), nil
}
