package catalog

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	pb "github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/internal/protos"
)

const (
	// Default gRPC server address - can be overridden with CATALOG_GRPC_ADDRESS env var
	defaultChainGRPCAddress = "localhost:8080"
)

// TestChainMetadata is a JSON-serializable struct for testing chain metadata
type TestChainMetadata struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	IsTestnet   bool     `json:"isTestnet"`
}

// newTestChainMetadata creates a TestChainMetadata with default values
func newTestChainMetadata(name string) TestChainMetadata {
	return TestChainMetadata{
		Name:        name,
		Description: "Test chain for integration testing",
		Tags:        []string{"test", "integration"},
		IsTestnet:   true,
	}
}

// setupTestChainStore creates a real gRPC client connection to a local service
func setupTestChainStore(t *testing.T) (*catalogChainMetadataStore, func()) {
	t.Helper()
	// Get gRPC address from environment or use default
	address := os.Getenv("CATALOG_GRPC_ADDRESS")
	if address == "" {
		address = defaultChainGRPCAddress
	}

	// Create CatalogClient using the NewCatalogClient function
	catalogClient, err := NewCatalogClient(CatalogConfig{
		GRPC:  address,
		Creds: insecure.NewCredentials(),
	})
	if err != nil {
		t.Skipf("Failed to connect to gRPC server at %s: %v. Skipping integration tests.", address, err)
		return nil, func() {}
	}

	// Test if the service is actually available by making a simple call
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := catalogClient.DataAccess(ctx)
	if err != nil {
		t.Skipf("gRPC service not available at %s: %v. Skipping integration tests.", address, err)
		return nil, func() {}
	}
	_ = stream.CloseSend() // Close the test stream

	// Create store
	store := newCatalogChainMetadataStore(catalogChainMetadataStoreConfig{
		Domain:      "test-domain",
		Environment: "catalog_testing",
		Client:      catalogClient,
	})

	cleanup := func() {
		// Connection cleanup is handled internally by CatalogClient
	}

	return store, cleanup
}

// generateRandomChainSelector generates a random chain selector
func generateRandomChainSelector() uint64 {
	maxVal := big.NewInt(999999999) // Large but reasonable upper bound
	n, err := rand.Int(rand.Reader, maxVal)
	if err != nil {
		panic(fmt.Sprintf("failed to generate random chain selector: %v", err))
	}

	return n.Uint64() + 1 // Ensure it's not zero
}

func newRandomChainMetadata() datastore.ChainMetadata {
	id := uuid.New().String()[:8] // Use first 8 chars of UUID for uniqueness

	return datastore.ChainMetadata{
		ChainSelector: generateRandomChainSelector(),
		Metadata:      newTestChainMetadata("TestChain-" + id),
	}
}

func TestCatalogChainMetadataStore_Get(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(store *catalogChainMetadataStore) datastore.ChainMetadataKey
		expectError bool
		errorType   error
	}{
		{
			name: "not_found",
			setup: func(store *catalogChainMetadataStore) datastore.ChainMetadataKey {
				// Use a unique key that shouldn't exist
				return datastore.NewChainMetadataKey(99999999)
			},
			expectError: true,
			errorType:   datastore.ErrChainMetadataNotFound,
		},
		{
			name: "success",
			setup: func(store *catalogChainMetadataStore) datastore.ChainMetadataKey {
				// Create and add a record first
				metadata := newRandomChainMetadata()
				err := store.Add(context.Background(), metadata)
				require.NoError(t, err)

				return metadata.Key()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh store for each test case to avoid concurrency issues
			store, cleanup := setupTestChainStore(t)
			defer cleanup()

			key := tt.setup(store)

			// Execute
			result, err := store.Get(context.Background(), key)

			// Verify
			if tt.expectError {
				require.Error(t, err)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, key.ChainSelector(), result.ChainSelector)
				require.NotNil(t, result.Metadata)
			}
		})
	}
}

func TestCatalogChainMetadataStore_Add(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(store *catalogChainMetadataStore) datastore.ChainMetadata
		expectError bool
		errorCheck  func(error) bool
	}{
		{
			name: "success",
			setup: func(store *catalogChainMetadataStore) datastore.ChainMetadata {
				return newRandomChainMetadata()
			},
			expectError: false,
		},
		{
			name: "duplicate_error",
			setup: func(store *catalogChainMetadataStore) datastore.ChainMetadata {
				// Create and add a record first
				metadata := newRandomChainMetadata()
				err := store.Add(context.Background(), metadata)
				require.NoError(t, err)
				// Return the same record to test duplicate
				return metadata
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh store for each test case to avoid concurrency issues
			store, cleanup := setupTestChainStore(t)
			defer cleanup()

			metadata := tt.setup(store)

			// Execute
			err := store.Add(context.Background(), metadata)

			// Verify
			if tt.expectError {
				require.Error(t, err)
				if tt.errorCheck != nil {
					require.True(t, tt.errorCheck(err))
				}
			} else {
				require.NoError(t, err)

				// Verify we can get it back
				retrieved, err := store.Get(context.Background(), metadata.Key())
				require.NoError(t, err)

				require.Equal(t, metadata.ChainSelector, retrieved.ChainSelector)

				concrete, err := datastore.As[TestChainMetadata](retrieved.Metadata)
				require.NoError(t, err)
				// Check that the metadata matches
				require.Equal(t, metadata.Metadata, concrete)
			}
		})
	}
}

func TestCatalogChainMetadataStore_Update(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(store *catalogChainMetadataStore) datastore.ChainMetadata
		expectError bool
		errorType   error
		verify      func(t *testing.T, store *catalogChainMetadataStore, metadata datastore.ChainMetadata)
	}{
		{
			name: "success",
			setup: func(store *catalogChainMetadataStore) datastore.ChainMetadata {
				// Create and add chain metadata
				metadata := newRandomChainMetadata()
				err := store.Add(context.Background(), metadata)
				require.NoError(t, err)

				// Fetch the record to get the current version in cache
				fetchedMetadata, err := store.Get(context.Background(), metadata.Key())
				require.NoError(t, err)

				// Modify the metadata
				updatedMetadata := newTestChainMetadata("UpdatedChain")
				updatedMetadata.Description = "Updated test chain"
				updatedMetadata.Tags = []string{"test", "updated"}
				updatedMetadata.IsTestnet = false
				fetchedMetadata.Metadata = updatedMetadata

				return fetchedMetadata
			},
			expectError: false,
			verify: func(t *testing.T, store *catalogChainMetadataStore, metadata datastore.ChainMetadata) {
				t.Helper()
				// Verify the updated values
				retrieved, err := store.Get(context.Background(), metadata.Key())
				require.NoError(t, err)

				concrete, err := datastore.As[TestChainMetadata](retrieved.Metadata)
				require.NoError(t, err)
				// Check that the metadata matches
				require.Equal(t, metadata.Metadata, concrete)
			},
		},
		{
			name: "not_found",
			setup: func(store *catalogChainMetadataStore) datastore.ChainMetadata {
				// Try to update a record that doesn't exist
				return newRandomChainMetadata()
			},
			expectError: true,
			errorType:   datastore.ErrChainMetadataNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh store for each test case to avoid concurrency issues
			store, cleanup := setupTestChainStore(t)
			defer cleanup()

			metadata := tt.setup(store)

			// Execute update
			err := store.Update(context.Background(), metadata.Key(), metadata.Metadata)

			// Verify
			if tt.expectError {
				require.Error(t, err)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
				if tt.verify != nil {
					tt.verify(t, store, metadata)
				}
			}
		})
	}
}

func TestCatalogChainMetadataStore_Update_WithCustomUpdater(t *testing.T) {
	t.Parallel()

	// Test cases for different custom updater scenarios
	tests := []struct {
		name     string
		updater  datastore.MetadataUpdaterF
		incoming any
		verify   func(t *testing.T, result TestChainMetadata, original TestChainMetadata)
	}{
		{
			name:     "description_only_update",
			updater:  descriptionOnlyUpdater(),
			incoming: "Custom description via updater",
			verify: func(t *testing.T, result TestChainMetadata, original TestChainMetadata) {
				t.Helper()
				require.Equal(t, original.Name, result.Name)
				require.Equal(t, "Custom description via updater", result.Description)
				require.Equal(t, original.Tags, result.Tags)
				require.Equal(t, original.IsTestnet, result.IsTestnet)
			},
		},
		{
			name:     "tags_merge_update",
			updater:  smartTagMerger(),
			incoming: []string{"custom", "updater", "test"}, // "test" should not duplicate
			verify: func(t *testing.T, result TestChainMetadata, original TestChainMetadata) {
				t.Helper()
				require.Equal(t, original.Name, result.Name)
				require.Equal(t, original.Description, result.Description)
				require.Equal(t, original.IsTestnet, result.IsTestnet)

				// Should have original tags plus new ones (without duplicates)
				require.Contains(t, result.Tags, "test")        // from original
				require.Contains(t, result.Tags, "integration") // from original
				require.Contains(t, result.Tags, "custom")      // new
				require.Contains(t, result.Tags, "updater")     // new
				require.Len(t, result.Tags, 4)                  // should not duplicate "test"
			},
		},
		{
			name:    "whole_metadata_merge",
			updater: wholeMetadataMerger(),
			incoming: TestChainMetadata{
				Name:        "MergedChain",
				Description: "Merged via whole metadata updater",
				Tags:        []string{"merged", "complete"},
				IsTestnet:   false, // this will be ignored in favor of keeping original
			},
			verify: func(t *testing.T, result TestChainMetadata, original TestChainMetadata) {
				t.Helper()
				require.Equal(t, "MergedChain", result.Name)                              // should be updated
				require.Equal(t, "Merged via whole metadata updater", result.Description) // should be updated
				require.Equal(t, original.IsTestnet, result.IsTestnet)                    // should keep original

				// Tags should be merged (original + incoming)
				require.Contains(t, result.Tags, "test")        // from original
				require.Contains(t, result.Tags, "integration") // from original
				require.Contains(t, result.Tags, "merged")      // from incoming
				require.Contains(t, result.Tags, "complete")    // from incoming
				require.Len(t, result.Tags, 4)                  // all tags merged
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a fresh store and data for each test case
			store, cleanup := setupTestChainStore(t)
			defer cleanup()

			// Create and add initial chain metadata
			original := newRandomChainMetadata()
			err := store.Add(context.Background(), original)
			require.NoError(t, err)

			// Use the options pattern with custom updater
			err = store.Update(context.Background(), original.Key(), tt.incoming, datastore.WithUpdater(tt.updater))
			require.NoError(t, err)

			// Verify the update worked correctly
			retrieved, err := store.Get(context.Background(), original.Key())
			require.NoError(t, err)

			result, err := datastore.As[TestChainMetadata](retrieved.Metadata)
			require.NoError(t, err)

			originalMeta, err := datastore.As[TestChainMetadata](original.Metadata)
			require.NoError(t, err)

			tt.verify(t, result, originalMeta)
		})
	}
}

func TestCatalogChainMetadataStore_Upsert_WithCustomUpdater(t *testing.T) {
	t.Parallel()

	// Test cases for different custom updater scenarios with Upsert
	tests := []struct {
		name        string
		updater     datastore.MetadataUpdaterF
		incoming    any
		setupRecord bool // whether to create an existing record first
		verify      func(t *testing.T, result TestChainMetadata, original *TestChainMetadata)
	}{
		{
			name:        "update_existing_with_description_updater",
			updater:     descriptionOnlyUpdater(),
			incoming:    "Updated description via upsert",
			setupRecord: true, // create existing record first
			verify: func(t *testing.T, result TestChainMetadata, original *TestChainMetadata) {
				t.Helper()
				require.Equal(t, original.Name, result.Name)
				require.Equal(t, "Updated description via upsert", result.Description)
				require.Equal(t, original.Tags, result.Tags)
				require.Equal(t, original.IsTestnet, result.IsTestnet)
			},
		},
		{
			name:        "update_existing_with_tag_merger",
			updater:     smartTagMerger(),
			incoming:    []string{"upserted", "smart", "test"}, // "test" should not duplicate
			setupRecord: true,                                  // create existing record first
			verify: func(t *testing.T, result TestChainMetadata, original *TestChainMetadata) {
				t.Helper()
				require.Equal(t, original.Name, result.Name)
				require.Equal(t, original.Description, result.Description)
				require.Equal(t, original.IsTestnet, result.IsTestnet)

				// Should have original tags plus new ones (without duplicates)
				require.Contains(t, result.Tags, "test")        // from original
				require.Contains(t, result.Tags, "integration") // from original
				require.Contains(t, result.Tags, "upserted")    // new
				require.Contains(t, result.Tags, "smart")       // new
				require.Len(t, result.Tags, 4)                  // should not duplicate "test"
			},
		},
		{
			name:    "update_existing_with_whole_metadata_merger",
			updater: wholeMetadataMerger(),
			incoming: TestChainMetadata{
				Name:        "UpsertedChain",
				Description: "Upserted via whole metadata updater",
				Tags:        []string{"upserted", "complete"},
				IsTestnet:   false, // this will be ignored in favor of keeping original
			},
			setupRecord: true, // create existing record first
			verify: func(t *testing.T, result TestChainMetadata, original *TestChainMetadata) {
				t.Helper()
				require.Equal(t, "UpsertedChain", result.Name)                              // should be updated
				require.Equal(t, "Upserted via whole metadata updater", result.Description) // should be updated
				require.Equal(t, original.IsTestnet, result.IsTestnet)                      // should keep original

				// Tags should be merged (original + incoming)
				require.Contains(t, result.Tags, "test")        // from original
				require.Contains(t, result.Tags, "integration") // from original
				require.Contains(t, result.Tags, "upserted")    // from incoming
				require.Contains(t, result.Tags, "complete")    // from incoming
				require.Len(t, result.Tags, 4)                  // all tags merged
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a fresh store for each test case
			store, cleanup := setupTestChainStore(t)
			defer cleanup()

			var original *datastore.ChainMetadata
			var key datastore.ChainMetadataKey

			if tt.setupRecord {
				// Create and add initial chain metadata
				originalRecord := newRandomChainMetadata()
				err := store.Add(context.Background(), originalRecord)
				require.NoError(t, err)
				original = &originalRecord
				key = originalRecord.Key()
			} else {
				// Create a key for a non-existing record
				original = nil
				key = datastore.NewChainMetadataKey(generateRandomChainSelector())
			}

			// Use the options pattern with custom updater for Upsert
			err := store.Upsert(context.Background(), key, tt.incoming, datastore.WithUpdater(tt.updater))

			// For new records with whole metadata merger, the updater might fail
			// In that case, we expect the operation to succeed with identity updater fallback
			if tt.name == "insert_new_with_whole_metadata_merger" {
				// This might fail because wholeMetadataMerger expects both latest and incoming to be TestChainMetadata
				// but latest will be nil for new records. Let's handle this gracefully.
				if err != nil {
					// If the custom updater fails, try without it to verify the record would work normally
					err = store.Upsert(context.Background(), key, tt.incoming)
				}
			}
			require.NoError(t, err)

			// Verify the upsert worked correctly
			retrieved, err := store.Get(context.Background(), key)
			require.NoError(t, err)

			result, err := datastore.As[TestChainMetadata](retrieved.Metadata)
			require.NoError(t, err)

			var originalMeta *TestChainMetadata
			if original != nil {
				meta, err := datastore.As[TestChainMetadata](original.Metadata)
				require.NoError(t, err)
				originalMeta = &meta
			}

			tt.verify(t, result, originalMeta)
		})
	}
}

func TestCatalogChainMetadataStore_Update_StaleVersion(t *testing.T) {
	t.Parallel()
	// Create two separate stores to simulate concurrent access
	store1, cleanup1 := setupTestChainStore(t)
	defer cleanup1()

	store2, cleanup2 := setupTestChainStore(t)
	defer cleanup2()

	// Add a chain metadata record using store1
	original := newRandomChainMetadata()
	err := store1.Add(context.Background(), original)
	require.NoError(t, err)

	// Both stores get the record to populate their caches with version 1
	key := datastore.NewChainMetadataKey(original.ChainSelector)
	first, err := store1.Get(context.Background(), key)
	require.NoError(t, err)

	second, err := store2.Get(context.Background(), key)
	require.NoError(t, err)

	// Store1 updates the record (this increments server version to 2)
	updatedMetadata := newTestChainMetadata("FirstUpdate")
	updatedMetadata.Description = "First update to chain"
	updatedMetadata.Tags = []string{"test", "first"}
	updatedMetadata.IsTestnet = false
	err = store1.Update(context.Background(), first.Key(), updatedMetadata)
	require.NoError(t, err)

	// Store2 also updates the record - this should succeed with V2 interface
	// because Update() fetches the current version internally
	secondUpdateMetadata := newTestChainMetadata("SecondUpdate")
	secondUpdateMetadata.Description = "Second update to chain"
	secondUpdateMetadata.Tags = []string{"test", "second"}
	secondUpdateMetadata.IsTestnet = true

	// Execute update with store2 (should succeed with V2 interface)
	err = store2.Update(context.Background(), second.Key(), secondUpdateMetadata)
	require.NoError(t, err)

	// Verify the final state reflects the second update
	final, err := store1.Get(context.Background(), key)
	require.NoError(t, err)

	concrete, err := datastore.As[TestChainMetadata](final.Metadata)
	require.NoError(t, err)
	require.Equal(t, "SecondUpdate", concrete.Name)
	require.Equal(t, "Second update to chain", concrete.Description)
	require.Equal(t, []string{"test", "second"}, concrete.Tags)
	require.True(t, concrete.IsTestnet)
}

func TestCatalogChainMetadataStore_Upsert(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(store *catalogChainMetadataStore) datastore.ChainMetadata
		expectError bool
		errorType   error
		verify      func(t *testing.T, store *catalogChainMetadataStore, original datastore.ChainMetadata)
	}{
		{
			name: "insert_new_record",
			setup: func(store *catalogChainMetadataStore) datastore.ChainMetadata {
				// Create a unique chain metadata for this test
				return newRandomChainMetadata()
			},
			expectError: false,
			verify: func(t *testing.T, store *catalogChainMetadataStore, original datastore.ChainMetadata) {
				t.Helper()
				// Verify we can get it back
				key := datastore.NewChainMetadataKey(original.ChainSelector)
				retrieved, err := store.Get(context.Background(), key)
				require.NoError(t, err)

				concrete, err := datastore.As[TestChainMetadata](retrieved.Metadata)
				require.NoError(t, err)
				require.Equal(t, original.Metadata, concrete)
			},
		},
		{
			name: "update_existing_record",
			setup: func(store *catalogChainMetadataStore) datastore.ChainMetadata {
				// Create and add chain metadata
				metadata := newRandomChainMetadata()
				err := store.Add(context.Background(), metadata)
				require.NoError(t, err)

				// Modify the metadata
				upsertedMetadata := newTestChainMetadata("UpsertedChain")
				upsertedMetadata.Description = "Upserted test chain"
				upsertedMetadata.Tags = []string{"test", "upserted"}
				upsertedMetadata.IsTestnet = false
				metadata.Metadata = upsertedMetadata

				return metadata
			},
			expectError: false,
			verify: func(t *testing.T, store *catalogChainMetadataStore, modified datastore.ChainMetadata) {
				t.Helper()
				// Verify the updated values
				key := datastore.NewChainMetadataKey(modified.ChainSelector)
				retrieved, err := store.Get(context.Background(), key)
				require.NoError(t, err)

				concrete, err := datastore.As[TestChainMetadata](retrieved.Metadata)
				require.NoError(t, err)
				require.Equal(t, modified.Metadata, concrete)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh store for each test case to avoid concurrency issues
			store, cleanup := setupTestChainStore(t)
			defer cleanup()

			metadata := tt.setup(store)

			// Execute upsert
			err := store.Upsert(context.Background(), metadata.Key(), metadata.Metadata)

			// Verify
			if tt.expectError {
				require.Error(t, err)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
				if tt.verify != nil {
					tt.verify(t, store, metadata)
				}
			}
		})
	}
}

func TestCatalogChainMetadataStore_Upsert_StaleVersion(t *testing.T) {
	t.Parallel()
	// Create two separate stores to simulate concurrent access
	store1, cleanup1 := setupTestChainStore(t)
	defer cleanup1()

	store2, cleanup2 := setupTestChainStore(t)
	defer cleanup2()

	// Add a chain metadata record using store1
	original := newRandomChainMetadata()
	err := store1.Add(context.Background(), original)
	require.NoError(t, err)

	// Both stores get the record to populate their caches with version 1
	key := datastore.NewChainMetadataKey(original.ChainSelector)
	first, err := store1.Get(context.Background(), key)
	require.NoError(t, err)

	second, err := store2.Get(context.Background(), key)
	require.NoError(t, err)

	// Store1 updates the record (this increments server version to 2)
	updatedMetadata := newTestChainMetadata("FirstUpdate")
	updatedMetadata.Description = "First update to chain"
	updatedMetadata.Tags = []string{"test", "first"}
	updatedMetadata.IsTestnet = false
	err = store1.Update(context.Background(), first.Key(), updatedMetadata)
	require.NoError(t, err)

	// Store2 upserts the record - this should succeed with V2 interface
	// because Upsert() fetches the current version internally
	upsertMetadata := newTestChainMetadata("UpsertUpdate")
	upsertMetadata.Description = "Upsert update to chain"
	upsertMetadata.Tags = []string{"test", "upserted"}
	upsertMetadata.IsTestnet = true

	// Execute upsert with store2 (should succeed with V2 interface)
	err = store2.Upsert(context.Background(), second.Key(), upsertMetadata)
	require.NoError(t, err)

	// Verify the final state reflects the upsert
	final, err := store1.Get(context.Background(), key)
	require.NoError(t, err)

	concrete, err := datastore.As[TestChainMetadata](final.Metadata)
	require.NoError(t, err)
	require.Equal(t, "UpsertUpdate", concrete.Name)
	require.Equal(t, "Upsert update to chain", concrete.Description)
	require.Equal(t, []string{"test", "upserted"}, concrete.Tags)
	require.True(t, concrete.IsTestnet)
}

func TestCatalogChainMetadataStore_Delete(t *testing.T) {
	t.Parallel()
	store, cleanup := setupTestChainStore(t)
	defer cleanup()

	key := datastore.NewChainMetadataKey(12345)

	// Execute
	err := store.Delete(context.Background(), key)

	// Verify
	require.Error(t, err)
	require.Contains(t, err.Error(), "delete operation not supported")
}

func TestCatalogChainMetadataStore_FetchAndFilter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		operation    string
		setup        func(store *catalogChainMetadataStore) (datastore.ChainMetadata, datastore.ChainMetadata)
		createFilter func(metadata1, metadata2 datastore.ChainMetadata) datastore.FilterFunc[datastore.ChainMetadataKey, datastore.ChainMetadata]
		minExpected  int
		verify       func(t *testing.T, results []datastore.ChainMetadata, metadata1, metadata2 datastore.ChainMetadata)
	}{
		{
			name:      "fetch_all",
			operation: "fetch",
			setup: func(store *catalogChainMetadataStore) (datastore.ChainMetadata, datastore.ChainMetadata) {
				// Setup test data with unique chain selectors
				metadata1 := newRandomChainMetadata()
				chainSelector1 := generateRandomChainSelector()
				metadata1.ChainSelector = chainSelector1
				err := store.Add(context.Background(), metadata1)
				require.NoError(t, err)

				metadata2 := newRandomChainMetadata()
				chainSelector2 := generateRandomChainSelector()
				// Ensure different chain selectors
				for chainSelector2 == chainSelector1 {
					chainSelector2 = generateRandomChainSelector()
				}
				metadata2.ChainSelector = chainSelector2
				err = store.Add(context.Background(), metadata2)
				require.NoError(t, err)

				return metadata1, metadata2
			},
			createFilter: nil,
			minExpected:  2,
			verify: func(t *testing.T, results []datastore.ChainMetadata, metadata1, metadata2 datastore.ChainMetadata) {
				t.Helper()
				// Check that our records are in the results
				foundFirst := false
				foundSecond := false
				for _, result := range results {
					if result.ChainSelector == metadata1.ChainSelector {
						foundFirst = true
					}
					if result.ChainSelector == metadata2.ChainSelector {
						foundSecond = true
					}
				}
				require.True(t, foundFirst, "First chain metadata not found in fetch results")
				require.True(t, foundSecond, "Second chain metadata not found in fetch results")
			},
		},
		{
			name:      "filter_by_chain_selector",
			operation: "filter",
			setup: func(store *catalogChainMetadataStore) (datastore.ChainMetadata, datastore.ChainMetadata) {
				// Setup test data with unique chain selectors
				metadata1 := newRandomChainMetadata()
				chainSelector1 := generateRandomChainSelector()
				metadata1.ChainSelector = chainSelector1
				err := store.Add(context.Background(), metadata1)
				require.NoError(t, err)

				metadata2 := newRandomChainMetadata()
				chainSelector2 := generateRandomChainSelector()
				// Ensure different chain selectors
				for chainSelector2 == chainSelector1 {
					chainSelector2 = generateRandomChainSelector()
				}
				metadata2.ChainSelector = chainSelector2
				err = store.Add(context.Background(), metadata2)
				require.NoError(t, err)

				return metadata1, metadata2
			},
			createFilter: func(metadata1, metadata2 datastore.ChainMetadata) datastore.FilterFunc[datastore.ChainMetadataKey, datastore.ChainMetadata] {
				// Use the proper filter from datastore/filters.go
				return datastore.ChainMetadataByChainSelector(metadata1.ChainSelector)
			},
			minExpected: 1,
			verify: func(t *testing.T, results []datastore.ChainMetadata, metadata1, metadata2 datastore.ChainMetadata) {
				t.Helper()
				// All results should have the chain selector from metadata1
				for _, result := range results {
					require.Equal(t, metadata1.ChainSelector, result.ChainSelector)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh store for each test case to avoid concurrency issues
			store, cleanup := setupTestChainStore(t)
			defer cleanup()

			metadata1, metadata2 := tt.setup(store)

			var results []datastore.ChainMetadata
			var err error

			// Execute operation
			switch tt.operation {
			case "fetch":
				results, err = store.Fetch(context.Background())
			case "filter":
				var filterFunc datastore.FilterFunc[datastore.ChainMetadataKey, datastore.ChainMetadata]
				if tt.createFilter != nil {
					filterFunc = tt.createFilter(metadata1, metadata2)
				}
				results, err = store.Filter(context.Background(), filterFunc)
			}

			// Verify
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(results), tt.minExpected)
			if tt.verify != nil {
				tt.verify(t, results, metadata1, metadata2)
			}
		})
	}
}

// Test updater functions that demonstrate different patterns for MetadataUpdaterF

// wholeMetadataMerger demonstrates merging two complete TestChainMetadata structs
func wholeMetadataMerger() datastore.MetadataUpdaterF {
	return func(latest any, incoming any) (any, error) {
		// Both latest and incoming are complete TestChainMetadata structs
		latestMeta, err := datastore.As[TestChainMetadata](latest)
		if err != nil {
			return nil, err
		}

		incomingMeta, err := datastore.As[TestChainMetadata](incoming)
		if err != nil {
			return nil, err
		}

		// Merge logic - keep some fields from latest, update others from incoming
		merged := TestChainMetadata{
			Name:        incomingMeta.Name,                             // Always update name
			Description: incomingMeta.Description,                      // Always update description
			Tags:        append(latestMeta.Tags, incomingMeta.Tags...), // Merge tags
			IsTestnet:   latestMeta.IsTestnet,                          // Keep original testnet flag
		}

		return merged, nil
	}
}

// descriptionOnlyUpdater demonstrates updating only a specific field
func descriptionOnlyUpdater() datastore.MetadataUpdaterF {
	return func(latest any, incoming any) (any, error) {
		// latest is full metadata, incoming is just a string description
		latestMeta, err := datastore.As[TestChainMetadata](latest)
		if err != nil {
			return nil, err
		}

		newDescription, err := datastore.As[string](incoming)
		if err != nil {
			return nil, err
		}

		// Update only the description field
		updated := latestMeta
		updated.Description = newDescription

		return updated, nil
	}
}

// smartTagMerger demonstrates intelligent tag merging without duplicates
func smartTagMerger() datastore.MetadataUpdaterF {
	return func(latest any, incoming any) (any, error) {
		// latest is full metadata, incoming is just new tags to add
		latestMeta, err := datastore.As[TestChainMetadata](latest)
		if err != nil {
			return nil, err
		}

		newTags, err := datastore.As[[]string](incoming)
		if err != nil {
			return nil, err
		}

		// Smart merge - avoid duplicates using a simple approach
		existingTags := latestMeta.Tags
		for _, newTag := range newTags {
			// Check if tag already exists
			found := false
			for _, existing := range existingTags {
				if existing == newTag {
					found = true
					break
				}
			}
			if !found {
				existingTags = append(existingTags, newTag)
			}
		}

		result := latestMeta
		result.Tags = existingTags

		return result, nil
	}
}

func TestCatalogChainMetadataStore_ConversionHelpers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T, store *catalogChainMetadataStore)
	}{
		{
			name: "keyToFilter",
			test: func(t *testing.T, store *catalogChainMetadataStore) {
				t.Helper()
				key := datastore.NewChainMetadataKey(12345)

				filter := store.keyToFilter(key)

				require.Equal(t, "test-domain", filter.Domain.Value)
				require.Equal(t, "catalog_testing", filter.Environment.Value)
				require.Equal(t, uint64(12345), filter.ChainSelector.Value)
			},
		},
		{
			name: "protoToChainMetadata_success",
			test: func(t *testing.T, store *catalogChainMetadataStore) {
				t.Helper()
				protoMetadata := &pb.ChainMetadata{
					Domain:        "test-domain",
					Environment:   "catalog_testing",
					ChainSelector: 12345,
					Metadata:      `{"name":"TestChain","description":"Test chain"}`,
					RowVersion:    1,
				}

				metadata, err := store.protoToChainMetadata(protoMetadata)

				require.NoError(t, err)
				require.Equal(t, uint64(12345), metadata.ChainSelector)
				require.NotNil(t, metadata.Metadata)

				// Check JSON unmarshaling - it will be unmarshaled as map[string]interface{}
				// since that's what json.Unmarshal defaults to for interface{}
				metadataMap := metadata.Metadata.(map[string]interface{})
				require.Equal(t, "TestChain", metadataMap["name"])
				require.Equal(t, "Test chain", metadataMap["description"])
			},
		},
		{
			name: "protoToChainMetadata_invalid_json",
			test: func(t *testing.T, store *catalogChainMetadataStore) {
				t.Helper()
				protoMetadata := &pb.ChainMetadata{
					Domain:        "test-domain",
					Environment:   "catalog_testing",
					ChainSelector: 12345,
					Metadata:      `{invalid-json`,
					RowVersion:    1,
				}

				_, err := store.protoToChainMetadata(protoMetadata)

				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to unmarshal metadata JSON")
			},
		},
		{
			name: "chainMetadataToProto",
			test: func(t *testing.T, store *catalogChainMetadataStore) {
				t.Helper()
				metadata := newRandomChainMetadata()

				protoMetadata := store.chainMetadataToProto(metadata, 0)

				require.Equal(t, "test-domain", protoMetadata.Domain)
				require.Equal(t, "catalog_testing", protoMetadata.Environment)
				require.Equal(t, metadata.ChainSelector, protoMetadata.ChainSelector)
				require.NotEmpty(t, protoMetadata.Metadata)
				require.Equal(t, int32(0), protoMetadata.RowVersion) // Should be 0 initially

				// Verify JSON marshaling worked
				require.Contains(t, protoMetadata.Metadata, "name")
				require.Contains(t, protoMetadata.Metadata, "description")
			},
		},
		{
			name: "version_handling",
			test: func(t *testing.T, store *catalogChainMetadataStore) {
				t.Helper()
				// Test protoToChainMetadata with version
				protoMetadata := &pb.ChainMetadata{
					Domain:        "test-domain",
					Environment:   "catalog_testing",
					ChainSelector: 12345,
					Metadata:      `{"name":"TestChain","description":"Test chain"}`,
					RowVersion:    5,
				}

				metadata, err := store.protoToChainMetadata(protoMetadata)
				require.NoError(t, err)

				// Test chainMetadataToProto with specific version
				protoResult := store.chainMetadataToProto(metadata, 7)
				require.Equal(t, int32(7), protoResult.RowVersion)

				// Test with version 0 (default for new records)
				protoResult0 := store.chainMetadataToProto(metadata, 0)
				require.Equal(t, int32(0), protoResult0.RowVersion)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh store for each test case to avoid concurrency issues
			store, cleanup := setupTestChainStore(t)
			defer cleanup()

			tt.test(t, store)
		})
	}
}

func TestCatalogChainMetadataStore_UpdaterExamples(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		updater datastore.MetadataUpdaterF
		verify  func(t *testing.T, result any)
	}{
		{
			name:    "whole_metadata_merge",
			updater: wholeMetadataMerger(),
			verify: func(t *testing.T, result any) {
				t.Helper()
				merged, err := datastore.As[TestChainMetadata](result)
				require.NoError(t, err)
				require.Equal(t, "NewChain", merged.Name)
				require.Equal(t, "New description", merged.Description)
				require.Contains(t, merged.Tags, "old")
				require.Contains(t, merged.Tags, "new")
				require.True(t, merged.IsTestnet) // Should keep original
			},
		},
		{
			name:    "description_only_update",
			updater: descriptionOnlyUpdater(),
			verify: func(t *testing.T, result any) {
				t.Helper()
				updated, err := datastore.As[TestChainMetadata](result)
				require.NoError(t, err)
				require.Equal(t, "OriginalChain", updated.Name) // Should be unchanged
				require.Equal(t, "Updated description only", updated.Description)
				require.Equal(t, []string{"old"}, updated.Tags) // Should be unchanged
				require.True(t, updated.IsTestnet)              // Should be unchanged
			},
		},
		{
			name:    "smart_tag_merging",
			updater: smartTagMerger(),
			verify: func(t *testing.T, result any) {
				t.Helper()
				updated, err := datastore.As[TestChainMetadata](result)
				require.NoError(t, err)
				require.Equal(t, "OriginalChain", updated.Name)
				require.Contains(t, updated.Tags, "old")
				require.Contains(t, updated.Tags, "new")
				require.Contains(t, updated.Tags, "additional")
				require.Len(t, updated.Tags, 3) // Should not have duplicates
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup test data based on the test case
			var latest, incoming any

			switch tt.name {
			case "whole_metadata_merge":
				latest = TestChainMetadata{
					Name:        "OriginalChain",
					Description: "Original description",
					Tags:        []string{"old"},
					IsTestnet:   true,
				}
				incoming = TestChainMetadata{
					Name:        "NewChain",
					Description: "New description",
					Tags:        []string{"new"},
					IsTestnet:   false,
				}
			case "description_only_update":
				latest = TestChainMetadata{
					Name:        "OriginalChain",
					Description: "Original description",
					Tags:        []string{"old"},
					IsTestnet:   true,
				}
				incoming = "Updated description only"
			case "smart_tag_merging":
				latest = TestChainMetadata{
					Name:        "OriginalChain",
					Description: "Original description",
					Tags:        []string{"old"},
					IsTestnet:   true,
				}
				incoming = []string{"new", "additional", "old"} // "old" should not duplicate
			}

			// Execute the updater
			result, err := tt.updater(latest, incoming)
			require.NoError(t, err)

			// Verify the result
			tt.verify(t, result)
		})
	}
}
