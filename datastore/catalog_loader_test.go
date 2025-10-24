package datastore

import (
	"context"
	"errors"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDataStoreFromCatalog(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("successfully loads all data from catalog", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockAddressStore := NewMockMutableRefStoreV2[AddressRefKey, AddressRef](t)
		mockChainStore := NewMockMutableStoreV2[ChainMetadataKey, ChainMetadata](t)
		mockContractStore := NewMockMutableStoreV2[ContractMetadataKey, ContractMetadata](t)
		mockEnvStore := NewMockMutableUnaryStoreV2[EnvMetadata](t)

		// Setup test data
		testAddressRefs := []AddressRef{
			{
				Address:   "0x123",
				Type:      "contract",
				Version:   semver.MustParse("1.0.0"),
				Qualifier: "test",
			},
			{
				Address:   "0x456",
				Type:      "registry",
				Version:   semver.MustParse("2.0.0"),
				Qualifier: "prod",
			},
		}

		testChainMetadata := []ChainMetadata{
			{
				ChainSelector: 1,
				Metadata: map[string]interface{}{
					"field": "value1",
				},
			},
			{
				ChainSelector: 2,
				Metadata: map[string]interface{}{
					"field": "value2",
				},
			},
		}

		testContractMetadata := []ContractMetadata{
			{
				Address:       "0x789",
				ChainSelector: 1,
				Metadata: map[string]interface{}{
					"name": "TestContract",
				},
			},
		}

		testEnvMetadata := EnvMetadata{
			Metadata: map[string]interface{}{
				"environment": "staging",
			},
		}

		// Setup mock expectations - catalog should only be called once per store
		mockCatalog.EXPECT().Addresses().Return(mockAddressStore).Once()
		mockAddressStore.EXPECT().Fetch(ctx).Return(testAddressRefs, nil).Once()

		mockCatalog.EXPECT().ChainMetadata().Return(mockChainStore).Once()
		mockChainStore.EXPECT().Fetch(ctx).Return(testChainMetadata, nil).Once()

		mockCatalog.EXPECT().ContractMetadata().Return(mockContractStore).Once()
		mockContractStore.EXPECT().Fetch(ctx).Return(testContractMetadata, nil).Once()

		mockCatalog.EXPECT().EnvMetadata().Return(mockEnvStore).Once()
		mockEnvStore.EXPECT().Get(ctx).Return(testEnvMetadata, nil).Once()

		// Execute
		ds, err := LoadDataStoreFromCatalog(ctx, mockCatalog)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, ds)

		// Verify all address refs were loaded - can read multiple times from local store
		loadedAddrs, err := ds.Addresses().Fetch()
		require.NoError(t, err)
		assert.Len(t, loadedAddrs, 2)
		assert.Contains(t, loadedAddrs, testAddressRefs[0])
		assert.Contains(t, loadedAddrs, testAddressRefs[1])

		// Verify we can read multiple times without hitting catalog again
		loadedAddrs2, err := ds.Addresses().Fetch()
		require.NoError(t, err)
		assert.Equal(t, loadedAddrs, loadedAddrs2)

		// Verify all chain metadata was loaded
		loadedChains, err := ds.ChainMetadata().Fetch()
		require.NoError(t, err)
		assert.Len(t, loadedChains, 2)

		// Verify all contract metadata was loaded
		loadedContracts, err := ds.ContractMetadata().Fetch()
		require.NoError(t, err)
		assert.Len(t, loadedContracts, 1)

		// Verify env metadata was loaded
		loadedEnv, err := ds.EnvMetadata().Get()
		require.NoError(t, err)
		assert.Equal(t, testEnvMetadata, loadedEnv)

		// Mock expectations will be verified on test cleanup - they should only be called once
	})

	t.Run("handles empty catalog data", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockAddressStore := NewMockMutableRefStoreV2[AddressRefKey, AddressRef](t)
		mockChainStore := NewMockMutableStoreV2[ChainMetadataKey, ChainMetadata](t)
		mockContractStore := NewMockMutableStoreV2[ContractMetadataKey, ContractMetadata](t)
		mockEnvStore := NewMockMutableUnaryStoreV2[EnvMetadata](t)

		// Setup mock expectations for empty data - catalog should only be called once per store
		mockCatalog.EXPECT().Addresses().Return(mockAddressStore).Once()
		mockAddressStore.EXPECT().Fetch(ctx).Return([]AddressRef{}, nil).Once()

		mockCatalog.EXPECT().ChainMetadata().Return(mockChainStore).Once()
		mockChainStore.EXPECT().Fetch(ctx).Return([]ChainMetadata{}, nil).Once()

		mockCatalog.EXPECT().ContractMetadata().Return(mockContractStore).Once()
		mockContractStore.EXPECT().Fetch(ctx).Return([]ContractMetadata{}, nil).Once()

		mockCatalog.EXPECT().EnvMetadata().Return(mockEnvStore).Once()
		mockEnvStore.EXPECT().Get(ctx).Return(EnvMetadata{}, nil).Once()

		// Execute
		ds, err := LoadDataStoreFromCatalog(ctx, mockCatalog)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, ds)

		// Verify stores are empty but functional
		loadedAddrs, err := ds.Addresses().Fetch()
		require.NoError(t, err)
		assert.Empty(t, loadedAddrs)

		loadedChains, err := ds.ChainMetadata().Fetch()
		require.NoError(t, err)
		assert.Empty(t, loadedChains)

		loadedContracts, err := ds.ContractMetadata().Fetch()
		require.NoError(t, err)
		assert.Empty(t, loadedContracts)
	})

	t.Run("continues when env metadata is not set (ErrEnvMetadataNotSet)", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockAddressStore := NewMockMutableRefStoreV2[AddressRefKey, AddressRef](t)
		mockChainStore := NewMockMutableStoreV2[ChainMetadataKey, ChainMetadata](t)
		mockContractStore := NewMockMutableStoreV2[ContractMetadataKey, ContractMetadata](t)
		mockEnvStore := NewMockMutableUnaryStoreV2[EnvMetadata](t)

		// Setup mock expectations - env metadata returns ErrEnvMetadataNotSet
		mockCatalog.EXPECT().Addresses().Return(mockAddressStore).Once()
		mockAddressStore.EXPECT().Fetch(ctx).Return([]AddressRef{}, nil).Once()

		mockCatalog.EXPECT().ChainMetadata().Return(mockChainStore).Once()
		mockChainStore.EXPECT().Fetch(ctx).Return([]ChainMetadata{}, nil).Once()

		mockCatalog.EXPECT().ContractMetadata().Return(mockContractStore).Once()
		mockContractStore.EXPECT().Fetch(ctx).Return([]ContractMetadata{}, nil).Once()

		mockCatalog.EXPECT().EnvMetadata().Return(mockEnvStore).Once()
		mockEnvStore.EXPECT().Get(ctx).Return(EnvMetadata{}, ErrEnvMetadataNotSet).Once()

		// Execute - should succeed because ErrEnvMetadataNotSet is acceptable
		ds, err := LoadDataStoreFromCatalog(ctx, mockCatalog)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, ds)
	})

	t.Run("returns error when env metadata fetch fails with non-ErrEnvMetadataNotSet error", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockAddressStore := NewMockMutableRefStoreV2[AddressRefKey, AddressRef](t)
		mockChainStore := NewMockMutableStoreV2[ChainMetadataKey, ChainMetadata](t)
		mockContractStore := NewMockMutableStoreV2[ContractMetadataKey, ContractMetadata](t)
		mockEnvStore := NewMockMutableUnaryStoreV2[EnvMetadata](t)

		// Setup mock expectations - env metadata returns a different error
		mockCatalog.EXPECT().Addresses().Return(mockAddressStore)
		mockAddressStore.EXPECT().Fetch(ctx).Return([]AddressRef{}, nil)

		mockCatalog.EXPECT().ChainMetadata().Return(mockChainStore)
		mockChainStore.EXPECT().Fetch(ctx).Return([]ChainMetadata{}, nil)

		mockCatalog.EXPECT().ContractMetadata().Return(mockContractStore)
		mockContractStore.EXPECT().Fetch(ctx).Return([]ContractMetadata{}, nil)

		mockCatalog.EXPECT().EnvMetadata().Return(mockEnvStore)
		mockEnvStore.EXPECT().Get(ctx).Return(EnvMetadata{}, errors.New("connection timeout"))

		// Execute
		ds, err := LoadDataStoreFromCatalog(ctx, mockCatalog)

		// Assert
		require.Error(t, err)
		assert.Nil(t, ds)
		require.ErrorContains(t, err, "failed to fetch environment metadata from catalog")
		require.ErrorContains(t, err, "connection timeout")
	})

	t.Run("continues when address refs not found (ErrAddressRefNotFound)", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockAddressStore := NewMockMutableRefStoreV2[AddressRefKey, AddressRef](t)
		mockChainStore := NewMockMutableStoreV2[ChainMetadataKey, ChainMetadata](t)
		mockContractStore := NewMockMutableStoreV2[ContractMetadataKey, ContractMetadata](t)
		mockEnvStore := NewMockMutableUnaryStoreV2[EnvMetadata](t)

		// Setup mock expectations - address refs returns ErrAddressRefNotFound
		mockCatalog.EXPECT().Addresses().Return(mockAddressStore).Once()
		mockAddressStore.EXPECT().Fetch(ctx).Return(nil, ErrAddressRefNotFound).Once()

		mockCatalog.EXPECT().ChainMetadata().Return(mockChainStore).Once()
		mockChainStore.EXPECT().Fetch(ctx).Return([]ChainMetadata{}, nil).Once()

		mockCatalog.EXPECT().ContractMetadata().Return(mockContractStore).Once()
		mockContractStore.EXPECT().Fetch(ctx).Return([]ContractMetadata{}, nil).Once()

		mockCatalog.EXPECT().EnvMetadata().Return(mockEnvStore).Once()
		mockEnvStore.EXPECT().Get(ctx).Return(EnvMetadata{}, ErrEnvMetadataNotSet).Once()

		// Execute - should succeed because ErrAddressRefNotFound is acceptable
		ds, err := LoadDataStoreFromCatalog(ctx, mockCatalog)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, ds)

		// Verify store is empty but functional
		loadedAddrs, err := ds.Addresses().Fetch()
		require.NoError(t, err)
		assert.Empty(t, loadedAddrs)
	})

	t.Run("continues when chain metadata not found (ErrChainMetadataNotFound)", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockAddressStore := NewMockMutableRefStoreV2[AddressRefKey, AddressRef](t)
		mockChainStore := NewMockMutableStoreV2[ChainMetadataKey, ChainMetadata](t)
		mockContractStore := NewMockMutableStoreV2[ContractMetadataKey, ContractMetadata](t)
		mockEnvStore := NewMockMutableUnaryStoreV2[EnvMetadata](t)

		// Setup mock expectations - chain metadata returns ErrChainMetadataNotFound
		mockCatalog.EXPECT().Addresses().Return(mockAddressStore).Once()
		mockAddressStore.EXPECT().Fetch(ctx).Return([]AddressRef{}, nil).Once()

		mockCatalog.EXPECT().ChainMetadata().Return(mockChainStore).Once()
		mockChainStore.EXPECT().Fetch(ctx).Return(nil, ErrChainMetadataNotFound).Once()

		mockCatalog.EXPECT().ContractMetadata().Return(mockContractStore).Once()
		mockContractStore.EXPECT().Fetch(ctx).Return([]ContractMetadata{}, nil).Once()

		mockCatalog.EXPECT().EnvMetadata().Return(mockEnvStore).Once()
		mockEnvStore.EXPECT().Get(ctx).Return(EnvMetadata{}, ErrEnvMetadataNotSet).Once()

		// Execute - should succeed because ErrChainMetadataNotFound is acceptable
		ds, err := LoadDataStoreFromCatalog(ctx, mockCatalog)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, ds)

		// Verify store is empty but functional
		loadedChains, err := ds.ChainMetadata().Fetch()
		require.NoError(t, err)
		assert.Empty(t, loadedChains)
	})

	t.Run("continues when contract metadata not found (ErrContractMetadataNotFound)", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockAddressStore := NewMockMutableRefStoreV2[AddressRefKey, AddressRef](t)
		mockChainStore := NewMockMutableStoreV2[ChainMetadataKey, ChainMetadata](t)
		mockContractStore := NewMockMutableStoreV2[ContractMetadataKey, ContractMetadata](t)
		mockEnvStore := NewMockMutableUnaryStoreV2[EnvMetadata](t)

		// Setup mock expectations - contract metadata returns ErrContractMetadataNotFound
		mockCatalog.EXPECT().Addresses().Return(mockAddressStore).Once()
		mockAddressStore.EXPECT().Fetch(ctx).Return([]AddressRef{}, nil).Once()

		mockCatalog.EXPECT().ChainMetadata().Return(mockChainStore).Once()
		mockChainStore.EXPECT().Fetch(ctx).Return([]ChainMetadata{}, nil).Once()

		mockCatalog.EXPECT().ContractMetadata().Return(mockContractStore).Once()
		mockContractStore.EXPECT().Fetch(ctx).Return(nil, ErrContractMetadataNotFound).Once()

		mockCatalog.EXPECT().EnvMetadata().Return(mockEnvStore).Once()
		mockEnvStore.EXPECT().Get(ctx).Return(EnvMetadata{}, ErrEnvMetadataNotSet).Once()

		// Execute - should succeed because ErrContractMetadataNotFound is acceptable
		ds, err := LoadDataStoreFromCatalog(ctx, mockCatalog)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, ds)

		// Verify store is empty but functional
		loadedContracts, err := ds.ContractMetadata().Fetch()
		require.NoError(t, err)
		assert.Empty(t, loadedContracts)
	})

	t.Run("returns error when address fetch fails with non-ErrAddressRefNotFound error", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockAddressStore := NewMockMutableRefStoreV2[AddressRefKey, AddressRef](t)

		// Setup mock expectations
		mockCatalog.EXPECT().Addresses().Return(mockAddressStore)
		mockAddressStore.EXPECT().Fetch(ctx).Return(nil, errors.New("connection error"))

		// Execute
		ds, err := LoadDataStoreFromCatalog(ctx, mockCatalog)

		// Assert
		require.Error(t, err)
		assert.Nil(t, ds)
		require.ErrorContains(t, err, "failed to fetch address references from catalog")
		require.ErrorContains(t, err, "connection error")
	})

	t.Run("returns error when chain metadata fetch fails with non-ErrChainMetadataNotFound error", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockAddressStore := NewMockMutableRefStoreV2[AddressRefKey, AddressRef](t)
		mockChainStore := NewMockMutableStoreV2[ChainMetadataKey, ChainMetadata](t)

		// Setup mock expectations
		mockCatalog.EXPECT().Addresses().Return(mockAddressStore)
		mockAddressStore.EXPECT().Fetch(ctx).Return([]AddressRef{}, nil)

		mockCatalog.EXPECT().ChainMetadata().Return(mockChainStore)
		mockChainStore.EXPECT().Fetch(ctx).Return(nil, errors.New("database error"))

		// Execute
		ds, err := LoadDataStoreFromCatalog(ctx, mockCatalog)

		// Assert
		require.Error(t, err)
		assert.Nil(t, ds)
		require.ErrorContains(t, err, "failed to fetch chain metadata from catalog")
		require.ErrorContains(t, err, "database error")
	})

	t.Run("returns error when contract metadata fetch fails with non-ErrContractMetadataNotFound error", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockAddressStore := NewMockMutableRefStoreV2[AddressRefKey, AddressRef](t)
		mockChainStore := NewMockMutableStoreV2[ChainMetadataKey, ChainMetadata](t)
		mockContractStore := NewMockMutableStoreV2[ContractMetadataKey, ContractMetadata](t)

		// Setup mock expectations
		mockCatalog.EXPECT().Addresses().Return(mockAddressStore)
		mockAddressStore.EXPECT().Fetch(ctx).Return([]AddressRef{}, nil)

		mockCatalog.EXPECT().ChainMetadata().Return(mockChainStore)
		mockChainStore.EXPECT().Fetch(ctx).Return([]ChainMetadata{}, nil)

		mockCatalog.EXPECT().ContractMetadata().Return(mockContractStore)
		mockContractStore.EXPECT().Fetch(ctx).Return(nil, errors.New("network timeout"))

		// Execute
		ds, err := LoadDataStoreFromCatalog(ctx, mockCatalog)

		// Assert
		require.Error(t, err)
		assert.Nil(t, ds)
		require.ErrorContains(t, err, "failed to fetch contract metadata from catalog")
		require.ErrorContains(t, err, "network timeout")
	})

	t.Run("returns sealed (read-only) datastore", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockAddressStore := NewMockMutableRefStoreV2[AddressRefKey, AddressRef](t)
		mockChainStore := NewMockMutableStoreV2[ChainMetadataKey, ChainMetadata](t)
		mockContractStore := NewMockMutableStoreV2[ContractMetadataKey, ContractMetadata](t)
		mockEnvStore := NewMockMutableUnaryStoreV2[EnvMetadata](t)

		// Setup mock expectations
		mockCatalog.EXPECT().Addresses().Return(mockAddressStore).Once()
		mockAddressStore.EXPECT().Fetch(ctx).Return([]AddressRef{}, nil).Once()

		mockCatalog.EXPECT().ChainMetadata().Return(mockChainStore).Once()
		mockChainStore.EXPECT().Fetch(ctx).Return([]ChainMetadata{}, nil).Once()

		mockCatalog.EXPECT().ContractMetadata().Return(mockContractStore).Once()
		mockContractStore.EXPECT().Fetch(ctx).Return([]ContractMetadata{}, nil).Once()

		mockCatalog.EXPECT().EnvMetadata().Return(mockEnvStore).Once()
		mockEnvStore.EXPECT().Get(ctx).Return(EnvMetadata{}, nil).Once()

		// Execute
		ds, err := LoadDataStoreFromCatalog(ctx, mockCatalog)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, ds)

		// Verify it's a sealed datastore (should be type *sealedMemoryDataStore)
		_, isSealed := ds.(*sealedMemoryDataStore)
		assert.True(t, isSealed, "returned datastore should be sealed (read-only)")
	})
}
