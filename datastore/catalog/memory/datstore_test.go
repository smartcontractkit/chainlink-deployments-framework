package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

func TestMemoryDatastore(t *testing.T) {
	t.Parallel()

	store, err := NewMemoryCatalogDataStore()
	require.NoError(t, err)
	assert.NotNil(t, store)
}

func TestMemoryDatastore_Accessors(t *testing.T) {
	t.Parallel()

	store, err := NewMemoryCatalogDataStore()
	require.NoError(t, err)

	t.Run("Addresses returns non-nil store", func(t *testing.T) {
		t.Parallel()
		addressStore := store.Addresses()
		require.NotNil(t, addressStore)
	})

	t.Run("ChainMetadata returns non-nil store", func(t *testing.T) {
		t.Parallel()
		chainStore := store.ChainMetadata()
		require.NotNil(t, chainStore)
	})

	t.Run("ContractMetadata returns non-nil store", func(t *testing.T) {
		t.Parallel()
		contractStore := store.ContractMetadata()
		require.NotNil(t, contractStore)
	})

	t.Run("EnvMetadata returns non-nil store", func(t *testing.T) {
		t.Parallel()
		envStore := store.EnvMetadata()
		require.NotNil(t, envStore)
	})
}

func TestMemoryDatastore_WithTransaction_Commit(t *testing.T) {
	t.Parallel()

	store, err := NewMemoryCatalogDataStore()
	require.NoError(t, err)

	ctx := t.Context()

	// Add data within a transaction
	err = store.WithTransaction(ctx, func(txCtx context.Context, catalog datastore.BaseCatalogStore) error {
		// Add an address reference
		version := semver.MustParse("1.0.0")
		addressRef := datastore.AddressRef{
			ChainSelector: 1,
			Address:       "0x123",
			Type:          "TestContract",
			Version:       version,
		}
		if addErr := catalog.Addresses().Add(txCtx, addressRef); addErr != nil {
			return addErr
		}

		// Add chain metadata
		chainMetadata := datastore.ChainMetadata{
			ChainSelector: 1,
			Metadata:      map[string]any{"name": "TestChain"},
		}
		if addErr := catalog.ChainMetadata().Add(txCtx, chainMetadata); addErr != nil {
			return addErr
		}

		return nil
	})
	require.NoError(t, err)

	// Verify data was committed
	version := semver.MustParse("1.0.0")
	addressKey := datastore.NewAddressRefKey(1, "TestContract", version, "")
	addressRef, err := store.Addresses().Get(ctx, addressKey)
	require.NoError(t, err)
	assert.Equal(t, "0x123", addressRef.Address)

	chainKey := datastore.NewChainMetadataKey(1)
	chainMetadata, err := store.ChainMetadata().Get(ctx, chainKey)
	require.NoError(t, err)
	metadataMap, ok := chainMetadata.Metadata.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "TestChain", metadataMap["name"])
}

func TestMemoryDatastore_WithTransaction_Rollback(t *testing.T) {
	t.Parallel()

	store, err := NewMemoryCatalogDataStore()
	require.NoError(t, err)

	ctx := t.Context()

	// Transaction that returns an error should rollback
	err = store.WithTransaction(ctx, func(txCtx context.Context, catalog datastore.BaseCatalogStore) error {
		// Add an address reference
		version := semver.MustParse("1.0.0")
		addressRef := datastore.AddressRef{
			ChainSelector: 1,
			Address:       "0x123",
			Type:          "TestContract",
			Version:       version,
		}
		if addErr := catalog.Addresses().Add(txCtx, addressRef); addErr != nil {
			return addErr
		}

		// Return an error to trigger rollback
		return errors.New("intentional error")
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "intentional error")

	// Verify data was NOT committed
	version := semver.MustParse("1.0.0")
	addressKey := datastore.NewAddressRefKey(1, "TestContract", version, "")
	_, err = store.Addresses().Get(ctx, addressKey)
	assert.ErrorIs(t, err, datastore.ErrAddressRefNotFound)
}

func TestMemoryDatastore_WithTransaction_Panic(t *testing.T) {
	t.Parallel()

	store, err := NewMemoryCatalogDataStore()
	require.NoError(t, err)

	ctx := t.Context()

	// Transaction that panics should rollback
	assert.Panics(t, func() {
		_ = store.WithTransaction(ctx, func(txCtx context.Context, catalog datastore.BaseCatalogStore) error {
			// Add an address reference
			version := semver.MustParse("1.0.0")
			addressRef := datastore.AddressRef{
				ChainSelector: 1,
				Address:       "0x123",
				Type:          "TestContract",
				Version:       version,
			}
			if addErr := catalog.Addresses().Add(txCtx, addressRef); addErr != nil {
				return addErr
			}

			// Panic to trigger rollback
			panic("intentional panic")
		})
	})

	// Verify data was NOT committed
	version := semver.MustParse("1.0.0")
	addressKey := datastore.NewAddressRefKey(1, "TestContract", version, "")
	_, err = store.Addresses().Get(ctx, addressKey)
	assert.ErrorIs(t, err, datastore.ErrAddressRefNotFound)
}
