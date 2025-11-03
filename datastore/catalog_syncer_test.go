package datastore

import (
	"context"
	"errors"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMergeDataStoreToCatalog(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("successfully merges all data to catalog", func(t *testing.T) {
		t.Parallel()

		// Create mocks for catalog and its stores
		mockCatalog := NewMockCatalogStore(t)
		mockTxCatalog := NewMockCatalogStore(t)
		mockCatalogAddressStore := NewMockMutableRefStoreV2[AddressRefKey, AddressRef](t)
		mockCatalogChainStore := NewMockMutableStoreV2[ChainMetadataKey, ChainMetadata](t)
		mockCatalogContractStore := NewMockMutableStoreV2[ContractMetadataKey, ContractMetadata](t)
		mockCatalogEnvStore := NewMockMutableUnaryStoreV2[EnvMetadata](t)

		// Create mocks for migration datastore and its stores
		mockMigrationDS := NewMockDataStore(t)
		mockMigrationAddressStore := NewMockAddressRefStore(t)
		mockMigrationChainStore := NewMockChainMetadataStore(t)
		mockMigrationContractStore := NewMockContractMetadataStore(t)
		mockMigrationEnvStore := NewMockEnvMetadataStore(t)

		// Setup test data
		testAddressRefs := []AddressRef{
			{
				Address:       "0xabc",
				ChainSelector: 3,
				Type:          "migration",
				Version:       semver.MustParse("3.0.0"),
				Qualifier:     "new",
			},
		}

		testChainMetadata := []ChainMetadata{
			{
				ChainSelector: 3,
				Metadata: map[string]interface{}{
					"field": "value3",
				},
			},
		}

		testContractMetadata := []ContractMetadata{
			{
				Address:       "0xdef",
				ChainSelector: 3,
				Metadata: map[string]interface{}{
					"name": "NewContract",
				},
			},
		}

		testEnvMetadata := EnvMetadata{
			Metadata: map[string]interface{}{
				"environment": "production",
			},
		}

		// Setup WithTransaction to execute the transaction logic
		mockCatalog.EXPECT().WithTransaction(ctx, mock.Anything).RunAndReturn(
			func(ctx context.Context, fn TransactionLogic) error {
				return fn(ctx, mockTxCatalog)
			},
		).Once()

		// Setup migration datastore expectations - fetch from migration
		mockMigrationDS.EXPECT().Addresses().Return(mockMigrationAddressStore).Once()
		mockMigrationAddressStore.EXPECT().Fetch().Return(testAddressRefs, nil).Once()

		mockMigrationDS.EXPECT().ChainMetadata().Return(mockMigrationChainStore).Once()
		mockMigrationChainStore.EXPECT().Fetch().Return(testChainMetadata, nil).Once()

		mockMigrationDS.EXPECT().ContractMetadata().Return(mockMigrationContractStore).Once()
		mockMigrationContractStore.EXPECT().Fetch().Return(testContractMetadata, nil).Once()

		mockMigrationDS.EXPECT().EnvMetadata().Return(mockMigrationEnvStore).Once()
		mockMigrationEnvStore.EXPECT().Get().Return(testEnvMetadata, nil).Once()

		// Setup catalog expectations - upsert to catalog
		mockTxCatalog.EXPECT().Addresses().Return(mockCatalogAddressStore).Times(1)
		for _, ref := range testAddressRefs {
			mockCatalogAddressStore.EXPECT().Upsert(ctx, ref).Return(nil).Once()
		}

		mockTxCatalog.EXPECT().ChainMetadata().Return(mockCatalogChainStore).Times(1)
		for _, metadata := range testChainMetadata {
			key := NewChainMetadataKey(metadata.ChainSelector)
			mockCatalogChainStore.EXPECT().Upsert(ctx, key, metadata.Metadata).Return(nil).Once()
		}

		mockTxCatalog.EXPECT().ContractMetadata().Return(mockCatalogContractStore).Times(1)
		for _, metadata := range testContractMetadata {
			key := NewContractMetadataKey(metadata.ChainSelector, metadata.Address)
			mockCatalogContractStore.EXPECT().Upsert(ctx, key, metadata.Metadata).Return(nil).Once()
		}

		mockTxCatalog.EXPECT().EnvMetadata().Return(mockCatalogEnvStore).Once()
		mockCatalogEnvStore.EXPECT().Set(ctx, testEnvMetadata.Metadata).Return(nil).Once()

		// Execute
		err := MergeDataStoreToCatalog(ctx, mockMigrationDS, mockCatalog)

		// Assert
		require.NoError(t, err)
	})

	t.Run("skips env metadata when not set (ErrEnvMetadataNotSet)", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockTxCatalog := NewMockCatalogStore(t)

		mockMigrationDS := NewMockDataStore(t)
		mockMigrationAddressStore := NewMockAddressRefStore(t)
		mockMigrationChainStore := NewMockChainMetadataStore(t)
		mockMigrationContractStore := NewMockContractMetadataStore(t)
		mockMigrationEnvStore := NewMockEnvMetadataStore(t)

		// Setup WithTransaction
		mockCatalog.EXPECT().WithTransaction(ctx, mock.Anything).RunAndReturn(
			func(ctx context.Context, fn TransactionLogic) error {
				return fn(ctx, mockTxCatalog)
			},
		).Once()

		// Setup migration datastore expectations
		mockMigrationDS.EXPECT().Addresses().Return(mockMigrationAddressStore).Once()
		mockMigrationAddressStore.EXPECT().Fetch().Return([]AddressRef{}, nil).Once()

		mockMigrationDS.EXPECT().ChainMetadata().Return(mockMigrationChainStore).Once()
		mockMigrationChainStore.EXPECT().Fetch().Return([]ChainMetadata{}, nil).Once()

		mockMigrationDS.EXPECT().ContractMetadata().Return(mockMigrationContractStore).Once()
		mockMigrationContractStore.EXPECT().Fetch().Return([]ContractMetadata{}, nil).Once()

		mockMigrationDS.EXPECT().EnvMetadata().Return(mockMigrationEnvStore).Once()
		mockMigrationEnvStore.EXPECT().Get().Return(EnvMetadata{}, ErrEnvMetadataNotSet).Once()

		// Execute - should succeed because ErrEnvMetadataNotSet is acceptable
		err := MergeDataStoreToCatalog(ctx, mockMigrationDS, mockCatalog)

		// Assert
		require.NoError(t, err)
	})

	t.Run("returns error when address fetch fails", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockTxCatalog := NewMockCatalogStore(t)
		mockMigrationDS := NewMockDataStore(t)
		mockMigrationAddressStore := NewMockAddressRefStore(t)

		// Setup WithTransaction
		mockCatalog.EXPECT().WithTransaction(ctx, mock.Anything).RunAndReturn(
			func(ctx context.Context, fn TransactionLogic) error {
				return fn(ctx, mockTxCatalog)
			},
		).Once()

		// Setup migration datastore to fail on address fetch
		mockMigrationDS.EXPECT().Addresses().Return(mockMigrationAddressStore)
		mockMigrationAddressStore.EXPECT().Fetch().Return(nil, errors.New("connection error"))

		// Execute
		err := MergeDataStoreToCatalog(ctx, mockMigrationDS, mockCatalog)

		// Assert
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to fetch address references from source store")
		require.ErrorContains(t, err, "connection error")
	})

	t.Run("returns error when chain metadata fetch fails", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockTxCatalog := NewMockCatalogStore(t)
		mockMigrationDS := NewMockDataStore(t)
		mockMigrationAddressStore := NewMockAddressRefStore(t)
		mockMigrationChainStore := NewMockChainMetadataStore(t)

		// Setup WithTransaction
		mockCatalog.EXPECT().WithTransaction(ctx, mock.Anything).RunAndReturn(
			func(ctx context.Context, fn TransactionLogic) error {
				return fn(ctx, mockTxCatalog)
			},
		).Once()

		// Setup migration datastore
		mockMigrationDS.EXPECT().Addresses().Return(mockMigrationAddressStore)
		mockMigrationAddressStore.EXPECT().Fetch().Return([]AddressRef{}, nil)

		mockMigrationDS.EXPECT().ChainMetadata().Return(mockMigrationChainStore)
		mockMigrationChainStore.EXPECT().Fetch().Return(nil, errors.New("database error"))

		// Execute
		err := MergeDataStoreToCatalog(ctx, mockMigrationDS, mockCatalog)

		// Assert
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to fetch chain metadata from source store")
		require.ErrorContains(t, err, "database error")
	})

	t.Run("returns error when contract metadata fetch fails", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockTxCatalog := NewMockCatalogStore(t)
		mockMigrationDS := NewMockDataStore(t)
		mockMigrationAddressStore := NewMockAddressRefStore(t)
		mockMigrationChainStore := NewMockChainMetadataStore(t)
		mockMigrationContractStore := NewMockContractMetadataStore(t)

		// Setup WithTransaction
		mockCatalog.EXPECT().WithTransaction(ctx, mock.Anything).RunAndReturn(
			func(ctx context.Context, fn TransactionLogic) error {
				return fn(ctx, mockTxCatalog)
			},
		).Once()

		// Setup migration datastore
		mockMigrationDS.EXPECT().Addresses().Return(mockMigrationAddressStore)
		mockMigrationAddressStore.EXPECT().Fetch().Return([]AddressRef{}, nil)

		mockMigrationDS.EXPECT().ChainMetadata().Return(mockMigrationChainStore)
		mockMigrationChainStore.EXPECT().Fetch().Return([]ChainMetadata{}, nil)

		mockMigrationDS.EXPECT().ContractMetadata().Return(mockMigrationContractStore)
		mockMigrationContractStore.EXPECT().Fetch().Return(nil, errors.New("network timeout"))

		// Execute
		err := MergeDataStoreToCatalog(ctx, mockMigrationDS, mockCatalog)

		// Assert
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to fetch contract metadata from source store")
		require.ErrorContains(t, err, "network timeout")
	})

	t.Run("returns error when env metadata fetch fails with non-ErrEnvMetadataNotSet error", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockTxCatalog := NewMockCatalogStore(t)
		mockMigrationDS := NewMockDataStore(t)
		mockMigrationAddressStore := NewMockAddressRefStore(t)
		mockMigrationChainStore := NewMockChainMetadataStore(t)
		mockMigrationContractStore := NewMockContractMetadataStore(t)
		mockMigrationEnvStore := NewMockEnvMetadataStore(t)

		// Setup WithTransaction
		mockCatalog.EXPECT().WithTransaction(ctx, mock.Anything).RunAndReturn(
			func(ctx context.Context, fn TransactionLogic) error {
				return fn(ctx, mockTxCatalog)
			},
		).Once()

		// Setup migration datastore
		mockMigrationDS.EXPECT().Addresses().Return(mockMigrationAddressStore)
		mockMigrationAddressStore.EXPECT().Fetch().Return([]AddressRef{}, nil)

		mockMigrationDS.EXPECT().ChainMetadata().Return(mockMigrationChainStore)
		mockMigrationChainStore.EXPECT().Fetch().Return([]ChainMetadata{}, nil)

		mockMigrationDS.EXPECT().ContractMetadata().Return(mockMigrationContractStore)
		mockMigrationContractStore.EXPECT().Fetch().Return([]ContractMetadata{}, nil)

		mockMigrationDS.EXPECT().EnvMetadata().Return(mockMigrationEnvStore)
		mockMigrationEnvStore.EXPECT().Get().Return(EnvMetadata{}, errors.New("connection timeout"))

		// Execute
		err := MergeDataStoreToCatalog(ctx, mockMigrationDS, mockCatalog)

		// Assert
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to fetch environment metadata from source store")
		require.ErrorContains(t, err, "connection timeout")
	})

	t.Run("returns error when address upsert fails", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockTxCatalog := NewMockCatalogStore(t)
		mockCatalogAddressStore := NewMockMutableRefStoreV2[AddressRefKey, AddressRef](t)

		mockMigrationDS := NewMockDataStore(t)
		mockMigrationAddressStore := NewMockAddressRefStore(t)

		testAddressRefs := []AddressRef{
			{
				Address:       "0x123",
				ChainSelector: 1,
				Type:          "contract",
				Version:       semver.MustParse("1.0.0"),
			},
		}

		// Setup WithTransaction
		mockCatalog.EXPECT().WithTransaction(ctx, mock.Anything).RunAndReturn(
			func(ctx context.Context, fn TransactionLogic) error {
				return fn(ctx, mockTxCatalog)
			},
		).Once()

		// Setup migration datastore
		mockMigrationDS.EXPECT().Addresses().Return(mockMigrationAddressStore)
		mockMigrationAddressStore.EXPECT().Fetch().Return(testAddressRefs, nil)

		// Setup catalog to fail on upsert
		mockTxCatalog.EXPECT().Addresses().Return(mockCatalogAddressStore)
		mockCatalogAddressStore.EXPECT().Upsert(ctx, testAddressRefs[0]).Return(errors.New("upsert failed"))

		// Execute
		err := MergeDataStoreToCatalog(ctx, mockMigrationDS, mockCatalog)

		// Assert
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to upsert address reference to catalog")
		require.ErrorContains(t, err, "upsert failed")
	})

	t.Run("handles empty migration datastore", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockTxCatalog := NewMockCatalogStore(t)
		mockCatalogEnvStore := NewMockMutableUnaryStoreV2[EnvMetadata](t)

		mockMigrationDS := NewMockDataStore(t)
		mockMigrationAddressStore := NewMockAddressRefStore(t)
		mockMigrationChainStore := NewMockChainMetadataStore(t)
		mockMigrationContractStore := NewMockContractMetadataStore(t)
		mockMigrationEnvStore := NewMockEnvMetadataStore(t)

		// Setup WithTransaction
		mockCatalog.EXPECT().WithTransaction(ctx, mock.Anything).RunAndReturn(
			func(ctx context.Context, fn TransactionLogic) error {
				return fn(ctx, mockTxCatalog)
			},
		).Once()

		// Setup migration datastore expectations - all empty
		mockMigrationDS.EXPECT().Addresses().Return(mockMigrationAddressStore).Once()
		mockMigrationAddressStore.EXPECT().Fetch().Return([]AddressRef{}, nil).Once()

		mockMigrationDS.EXPECT().ChainMetadata().Return(mockMigrationChainStore).Once()
		mockMigrationChainStore.EXPECT().Fetch().Return([]ChainMetadata{}, nil).Once()

		mockMigrationDS.EXPECT().ContractMetadata().Return(mockMigrationContractStore).Once()
		mockMigrationContractStore.EXPECT().Fetch().Return([]ContractMetadata{}, nil).Once()

		mockMigrationDS.EXPECT().EnvMetadata().Return(mockMigrationEnvStore).Once()
		mockMigrationEnvStore.EXPECT().Get().Return(EnvMetadata{Metadata: map[string]interface{}{}}, nil).Once()

		// Setup catalog expectations - env metadata set only
		mockTxCatalog.EXPECT().EnvMetadata().Return(mockCatalogEnvStore).Once()
		mockCatalogEnvStore.EXPECT().Set(ctx, mock.Anything).Return(nil).Once()

		// Execute
		err := MergeDataStoreToCatalog(ctx, mockMigrationDS, mockCatalog)

		// Assert
		require.NoError(t, err)
	})

	t.Run("transaction rollback on error", func(t *testing.T) {
		t.Parallel()

		// Create mocks
		mockCatalog := NewMockCatalogStore(t)
		mockMigrationDS := NewMockDataStore(t)

		// Setup WithTransaction to return an error (simulating rollback)
		expectedErr := errors.New("transaction rolled back")
		mockCatalog.EXPECT().WithTransaction(ctx, mock.Anything).Return(expectedErr).Once()

		// Execute
		err := MergeDataStoreToCatalog(ctx, mockMigrationDS, mockCatalog)

		// Assert
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}
