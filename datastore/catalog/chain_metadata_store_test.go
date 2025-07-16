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
	"google.golang.org/grpc"
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
func setupTestChainStore(t *testing.T) (*CatalogChainMetadataStore, *grpc.ClientConn) {
	t.Helper()
	// Get gRPC address from environment or use default
	address := os.Getenv("CATALOG_GRPC_ADDRESS")
	if address == "" {
		address = defaultChainGRPCAddress
	}

	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Skipf("Failed to connect to gRPC server at %s: %v. Skipping integration tests.", address, err)
	}

	// Create client
	client := pb.NewDeploymentsDatastoreClient(conn)

	// Test if the service is actually available by making a simple call
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := client.DataAccess(ctx)
	if err != nil {
		conn.Close()
		t.Skipf("gRPC service not available at %s: %v. Skipping integration tests.", address, err)
	}
	_ = stream.CloseSend() // Close the test stream

	// Create store
	store := NewCatalogChainMetadataStore(CatalogChainMetadataStoreConfig{
		Domain:      "test-domain",
		Environment: "catalog_testing",
		Client:      client,
	})

	return store, conn
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
		setup       func(store *CatalogChainMetadataStore) datastore.ChainMetadataKey
		expectError bool
		errorType   error
	}{
		{
			name: "not_found",
			setup: func(store *CatalogChainMetadataStore) datastore.ChainMetadataKey {
				// Use a unique key that shouldn't exist
				return datastore.NewChainMetadataKey(99999999)
			},
			expectError: true,
			errorType:   datastore.ErrChainMetadataNotFound,
		},
		{
			name: "success",
			setup: func(store *CatalogChainMetadataStore) datastore.ChainMetadataKey {
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
			store, conn := setupTestChainStore(t)
			defer conn.Close()

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
		setup       func(store *CatalogChainMetadataStore) datastore.ChainMetadata
		expectError bool
		errorCheck  func(error) bool
	}{
		{
			name: "success",
			setup: func(store *CatalogChainMetadataStore) datastore.ChainMetadata {
				return newRandomChainMetadata()
			},
			expectError: false,
		},
		{
			name: "duplicate_error",
			setup: func(store *CatalogChainMetadataStore) datastore.ChainMetadata {
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
			store, conn := setupTestChainStore(t)
			defer conn.Close()

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
		setup       func(store *CatalogChainMetadataStore) datastore.ChainMetadata
		expectError bool
		errorType   error
		verify      func(t *testing.T, store *CatalogChainMetadataStore, metadata datastore.ChainMetadata)
	}{
		{
			name: "success",
			setup: func(store *CatalogChainMetadataStore) datastore.ChainMetadata {
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
			verify: func(t *testing.T, store *CatalogChainMetadataStore, metadata datastore.ChainMetadata) {
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
			setup: func(store *CatalogChainMetadataStore) datastore.ChainMetadata {
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
			store, conn := setupTestChainStore(t)
			defer conn.Close()

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

func TestCatalogChainMetadataStore_Update_StaleVersion(t *testing.T) {
	t.Parallel()
	// Create two separate stores to simulate concurrent access
	store1, conn1 := setupTestChainStore(t)
	defer conn1.Close()

	store2, conn2 := setupTestChainStore(t)
	defer conn2.Close()

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
	require.Equal(t, true, concrete.IsTestnet)
}

func TestCatalogChainMetadataStore_Upsert(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(store *CatalogChainMetadataStore) datastore.ChainMetadata
		expectError bool
		errorType   error
		verify      func(t *testing.T, store *CatalogChainMetadataStore, original datastore.ChainMetadata)
	}{
		{
			name: "insert_new_record",
			setup: func(store *CatalogChainMetadataStore) datastore.ChainMetadata {
				// Create a unique chain metadata for this test
				return newRandomChainMetadata()
			},
			expectError: false,
			verify: func(t *testing.T, store *CatalogChainMetadataStore, original datastore.ChainMetadata) {
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
			setup: func(store *CatalogChainMetadataStore) datastore.ChainMetadata {
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
			verify: func(t *testing.T, store *CatalogChainMetadataStore, modified datastore.ChainMetadata) {
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
			store, conn := setupTestChainStore(t)
			defer conn.Close()

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
	store1, conn1 := setupTestChainStore(t)
	defer conn1.Close()

	store2, conn2 := setupTestChainStore(t)
	defer conn2.Close()

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
	require.Equal(t, true, concrete.IsTestnet)
}

func TestCatalogChainMetadataStore_Delete(t *testing.T) {
	t.Parallel()
	store, conn := setupTestChainStore(t)
	defer conn.Close()

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
		setup        func(store *CatalogChainMetadataStore) (datastore.ChainMetadata, datastore.ChainMetadata)
		createFilter func(metadata1, metadata2 datastore.ChainMetadata) datastore.FilterFunc[datastore.ChainMetadataKey, datastore.ChainMetadata]
		minExpected  int
		verify       func(t *testing.T, results []datastore.ChainMetadata, metadata1, metadata2 datastore.ChainMetadata)
	}{
		{
			name:      "fetch_all",
			operation: "fetch",
			setup: func(store *CatalogChainMetadataStore) (datastore.ChainMetadata, datastore.ChainMetadata) {
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
			setup: func(store *CatalogChainMetadataStore) (datastore.ChainMetadata, datastore.ChainMetadata) {
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
			store, conn := setupTestChainStore(t)
			defer conn.Close()

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
				results = store.Filter(context.Background(), filterFunc)
			}

			// Verify
			if tt.operation == "fetch" {
				require.NoError(t, err)
			}
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
		test func(t *testing.T, store *CatalogChainMetadataStore)
	}{
		{
			name: "keyToFilter",
			test: func(t *testing.T, store *CatalogChainMetadataStore) {
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
			test: func(t *testing.T, store *CatalogChainMetadataStore) {
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
			test: func(t *testing.T, store *CatalogChainMetadataStore) {
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
			test: func(t *testing.T, store *CatalogChainMetadataStore) {
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
			test: func(t *testing.T, store *CatalogChainMetadataStore) {
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
			store, conn := setupTestChainStore(t)
			defer conn.Close()

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
				require.Equal(t, true, merged.IsTestnet) // Should keep original
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
				require.Equal(t, true, updated.IsTestnet)       // Should be unchanged
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
