package remote

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	pb "github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/remote/internal/protos"
)

const (
	// Default gRPC server address - can be overridden with CATALOG_GRPC_ADDRESS env var
	defaultContractGRPCAddress = "localhost:8080"
)

// TestContractMetadata is a JSON-serializable struct for testing contract metadata
type TestContractMetadata struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// newTestContractMetadata creates a TestContractMetadata with default values
func newTestContractMetadata(name string) TestContractMetadata {
	return TestContractMetadata{
		Name:        name,
		Version:     "1.0.0",
		Description: "Test contract for integration testing",
		Tags:        []string{"test", "integration"},
	}
}

// setupTestContractStore creates a real gRPC client connection to a local service
func setupTestContractStore(t *testing.T) *catalogContractMetadataStore {
	t.Helper()
	// Get gRPC address from environment or use default
	address := os.Getenv("CATALOG_GRPC_ADDRESS")
	if address == "" {
		address = defaultContractGRPCAddress
	}

	// Create CatalogClient using the NewCatalogClient function
	catalogClient, err := NewCatalogClient(t.Context(), CatalogConfig{
		GRPC:  address,
		Creds: insecure.NewCredentials(),
	})
	if err != nil {
		t.Skipf("Failed to connect to gRPC server at %s: %v. Skipping integration tests.", address, err)
		return nil
	}

	// Test if the gRPC service is actually available by making a simple call
	_, err = catalogClient.DataAccess()
	if err != nil {
		t.Skipf("gRPC service not available at %s: %v. Skipping integration tests.", address, err)
		return nil
	}
	t.Cleanup(func() {
		_ = catalogClient.CloseStream() // Close the test stream at the end of the test.
	})

	// Create store
	store := newCatalogContractMetadataStore(catalogContractMetadataStoreConfig{
		Domain:      "test-domain",
		Environment: "catalog_testing",
		Client:      catalogClient,
	})

	return store
}

// generateRandomContractAddress generates a random contract address
func generateRandomContractAddress() string {
	bytes := make([]byte, 20) // 20 bytes = 40 hex chars for Ethereum address
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("failed to generate random address: %v", err))
	}

	return fmt.Sprintf("0x%x", bytes)
}

// generateRandomContractChainSelector generates a random chain selector
func generateRandomContractChainSelector() uint64 {
	maxVal := big.NewInt(999999999) // Large but reasonable upper bound
	n, err := rand.Int(rand.Reader, maxVal)
	if err != nil {
		panic(fmt.Sprintf("failed to generate random chain selector: %v", err))
	}

	return n.Uint64() + 1 // Ensure it's not zero
}

func newRandomContractMetadata() datastore.ContractMetadata {
	id := uuid.New().String()[:8] // Use first 8 chars of UUID for uniqueness

	return datastore.ContractMetadata{
		Address:       generateRandomContractAddress(),
		ChainSelector: generateRandomContractChainSelector(),
		Metadata:      newTestContractMetadata("TestContract-" + id),
	} // Version is managed internally by the store
}

func TestCatalogContractMetadataStore_Get(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(store *catalogContractMetadataStore) datastore.ContractMetadataKey
		expectError bool
		errorType   error
	}{
		{
			name: "not_found",
			setup: func(store *catalogContractMetadataStore) datastore.ContractMetadataKey {
				// Use a unique key that shouldn't exist
				return datastore.NewContractMetadataKey(99999999, "0xnonexistent1234567890abcdef1234567890ab")
			},
			expectError: true,
			errorType:   datastore.ErrContractMetadataNotFound,
		},
		{
			name: "success",
			setup: func(store *catalogContractMetadataStore) datastore.ContractMetadataKey {
				// Create and add a record first
				metadata := newRandomContractMetadata()
				err := store.Add(t.Context(), metadata)
				require.NoError(t, err)

				return datastore.NewContractMetadataKey(metadata.ChainSelector, metadata.Address)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh store for each test case to avoid concurrency issues
			store := setupTestContractStore(t)

			key := tt.setup(store)

			// Execute
			result, err := store.Get(t.Context(), key)

			// Verify
			if tt.expectError {
				require.Error(t, err)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, key.ChainSelector(), result.ChainSelector)
				require.Equal(t, key.Address(), result.Address)

				typedMeta, err := datastore.As[TestContractMetadata](result.Metadata)
				require.NoError(t, err)
				require.NotNil(t, typedMeta.Description)
			}
		})
	}
}

func TestCatalogContractMetadataStore_Add(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(store *catalogContractMetadataStore) datastore.ContractMetadata
		expectError bool
		errorCheck  func(error) bool
	}{
		{
			name: "success",
			setup: func(store *catalogContractMetadataStore) datastore.ContractMetadata {
				return newRandomContractMetadata()
			},
			expectError: false,
		},
		{
			name: "duplicate_error",
			setup: func(store *catalogContractMetadataStore) datastore.ContractMetadata {
				// Create and add a record first
				metadata := newRandomContractMetadata()
				err := store.Add(t.Context(), metadata)
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
			store := setupTestContractStore(t)

			metadata := tt.setup(store)

			// Execute
			err := store.Add(t.Context(), metadata)

			// Verify
			if tt.expectError {
				require.Error(t, err)
				if tt.errorCheck != nil {
					require.True(t, tt.errorCheck(err))
				}
			} else {
				require.NoError(t, err)

				// Verify we can get it back
				key := datastore.NewContractMetadataKey(metadata.ChainSelector, metadata.Address)
				retrieved, err := store.Get(t.Context(), key)
				require.NoError(t, err)

				require.Equal(t, metadata.Address, retrieved.Address)
				require.Equal(t, metadata.ChainSelector, retrieved.ChainSelector)

				concrete, err := datastore.As[TestContractMetadata](retrieved.Metadata)
				require.NoError(t, err)
				// Check that the metadata matches
				require.Equal(t, metadata.Metadata, concrete)
			}
		})
	}
}

func TestCatalogContractMetadataStore_Update(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(store *catalogContractMetadataStore) datastore.ContractMetadata
		expectError bool
		errorType   error
		verify      func(t *testing.T, store *catalogContractMetadataStore, metadata datastore.ContractMetadata)
	}{
		{
			name: "success",
			setup: func(store *catalogContractMetadataStore) datastore.ContractMetadata {
				// Create and add contract metadata
				metadata := newRandomContractMetadata()
				err := store.Add(t.Context(), metadata)
				require.NoError(t, err)

				// Fetch the record to get the current version in cache
				fetchedMetadata, err := store.Get(t.Context(), metadata.Key())
				require.NoError(t, err)

				// Modify the metadata
				updatedMetadata := newTestContractMetadata("UpdatedContract")
				updatedMetadata.Version = "2.0.0"
				updatedMetadata.Description = "Updated test contract"
				updatedMetadata.Tags = []string{"test", "updated"}
				fetchedMetadata.Metadata = updatedMetadata

				return fetchedMetadata
			},
			expectError: false,
			verify: func(t *testing.T, store *catalogContractMetadataStore, metadata datastore.ContractMetadata) {
				t.Helper()
				// Verify the updated values
				key := datastore.NewContractMetadataKey(metadata.ChainSelector, metadata.Address)
				retrieved, err := store.Get(t.Context(), key)
				require.NoError(t, err)

				concrete, err := datastore.As[TestContractMetadata](retrieved.Metadata)
				require.NoError(t, err)
				// Check that the metadata matches
				require.Equal(t, metadata.Metadata, concrete)
			},
		},
		{
			name: "not_found",
			setup: func(store *catalogContractMetadataStore) datastore.ContractMetadata {
				// Try to update a record that doesn't exist
				return newRandomContractMetadata()
			},
			expectError: true,
			errorType:   datastore.ErrContractMetadataNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh store for each test case to avoid concurrency issues
			store := setupTestContractStore(t)

			metadata := tt.setup(store)

			// Execute update
			err := store.Update(t.Context(), metadata.Key(), metadata.Metadata)

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

func TestCatalogContractMetadataStore_Update_StaleVersion(t *testing.T) {
	t.Parallel()
	// Create two separate stores to simulate concurrent access
	store1 := setupTestContractStore(t)

	store2 := setupTestContractStore(t)

	// Add a contract metadata record using store1
	original := newRandomContractMetadata()
	err := store1.Add(t.Context(), original)
	require.NoError(t, err)

	// Both stores get the record to populate their caches with version 1
	key := datastore.NewContractMetadataKey(original.ChainSelector, original.Address)
	first, err := store1.Get(t.Context(), key)
	require.NoError(t, err)

	second, err := store2.Get(t.Context(), key)
	require.NoError(t, err)

	// Store1 updates the record (this increments server version to 2)
	updatedMetadata := newTestContractMetadata("FirstUpdate")
	updatedMetadata.Version = "2.0.0"
	err = store1.Update(t.Context(), first.Key(), updatedMetadata)
	require.NoError(t, err)

	// Store2 tries to update using its cached version (still version 1, now stale)
	// In V2, this should succeed because performUpsertOrUpdate fetches latest version internally
	staleMetadata := newTestContractMetadata("StaleUpdate")
	staleMetadata.Version = "3.0.0"

	// Execute update with store2 (should succeed in V2 due to internal version fetching)
	err = store2.Update(t.Context(), second.Key(), staleMetadata)

	// Verify both updates succeeded
	require.NoError(t, err)

	// Verify the final state - should have the second update's metadata
	final, err := store1.Get(t.Context(), key)
	require.NoError(t, err)

	concrete, err := datastore.As[TestContractMetadata](final.Metadata)
	require.NoError(t, err)
	require.Equal(t, staleMetadata, concrete)
}

func TestCatalogContractMetadataStore_Upsert(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(store *catalogContractMetadataStore) datastore.ContractMetadata
		expectError bool
		errorType   error
		verify      func(t *testing.T, store *catalogContractMetadataStore, original datastore.ContractMetadata)
	}{
		{
			name: "insert_new_record",
			setup: func(store *catalogContractMetadataStore) datastore.ContractMetadata {
				// Create a unique contract metadata for this test
				return newRandomContractMetadata()
			},
			expectError: false,
			verify: func(t *testing.T, store *catalogContractMetadataStore, original datastore.ContractMetadata) {
				t.Helper()
				// Verify we can get it back
				key := datastore.NewContractMetadataKey(original.ChainSelector, original.Address)
				retrieved, err := store.Get(t.Context(), key)
				require.NoError(t, err)

				concrete, err := datastore.As[TestContractMetadata](retrieved.Metadata)
				require.NoError(t, err)
				require.Equal(t, original.Metadata, concrete)
			},
		},
		{
			name: "update_existing_record",
			setup: func(store *catalogContractMetadataStore) datastore.ContractMetadata {
				// Create and add contract metadata
				metadata := newRandomContractMetadata()
				err := store.Add(t.Context(), metadata)
				require.NoError(t, err)

				// Modify the metadata
				upsertedMetadata := newTestContractMetadata("UpsertedContract")
				upsertedMetadata.Version = "3.0.0"
				upsertedMetadata.Description = "Upserted test contract"
				upsertedMetadata.Tags = []string{"test", "upserted"}
				metadata.Metadata = upsertedMetadata

				return metadata
			},
			expectError: false,
			verify: func(t *testing.T, store *catalogContractMetadataStore, modified datastore.ContractMetadata) {
				t.Helper()
				// Verify the updated values
				key := datastore.NewContractMetadataKey(modified.ChainSelector, modified.Address)
				retrieved, err := store.Get(t.Context(), key)
				require.NoError(t, err)

				concrete, err := datastore.As[TestContractMetadata](retrieved.Metadata)
				require.NoError(t, err)
				require.Equal(t, modified.Metadata, concrete)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh store for each test case to avoid concurrency issues
			store := setupTestContractStore(t)

			metadata := tt.setup(store)

			// Execute upsert
			err := store.Upsert(t.Context(), metadata.Key(), metadata.Metadata)

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

func TestCatalogContractMetadataStore_Upsert_StaleVersion(t *testing.T) {
	t.Parallel()
	// Create two separate stores to simulate concurrent access
	store1 := setupTestContractStore(t)

	store2 := setupTestContractStore(t)

	// Add a contract metadata record using store1
	original := newRandomContractMetadata()
	err := store1.Add(t.Context(), original)
	require.NoError(t, err)

	// Both stores get the record to populate their caches with version 1
	key := datastore.NewContractMetadataKey(original.ChainSelector, original.Address)
	first, err := store1.Get(t.Context(), key)
	require.NoError(t, err)

	second, err := store2.Get(t.Context(), key)
	require.NoError(t, err)

	// Store1 updates the record (this increments server version to 2)
	updatedMetadata := newTestContractMetadata("FirstUpdate")
	updatedMetadata.Version = "2.0.0"
	err = store1.Update(t.Context(), first.Key(), updatedMetadata)
	require.NoError(t, err)

	// Store2 tries to upsert using its cached version (still version 1, now stale)
	// In V2, this should succeed because performUpsertOrUpdate fetches latest version internally
	staleMetadata := newTestContractMetadata("UpsertStaleUpdate")
	staleMetadata.Version = "3.0.0"

	// Execute upsert with store2 (should succeed in V2 due to internal version fetching)
	err = store2.Upsert(t.Context(), second.Key(), staleMetadata)

	// Verify both operations succeeded
	require.NoError(t, err)

	// Verify the final state - should have the second upsert's metadata
	final, err := store1.Get(t.Context(), key)
	require.NoError(t, err)

	concrete, err := datastore.As[TestContractMetadata](final.Metadata)
	require.NoError(t, err)
	require.Equal(t, staleMetadata, concrete)
}

func TestCatalogContractMetadataStore_Delete(t *testing.T) {
	t.Parallel()
	store := setupTestContractStore(t)

	key := datastore.NewContractMetadataKey(12345, "0x1234567890abcdef1234567890abcdef12345678")

	// Execute
	err := store.Delete(t.Context(), key)

	// Verify
	require.Error(t, err)
	require.Contains(t, err.Error(), "delete operation not supported")
}

func TestCatalogContractMetadataStore_FetchAndFilter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		operation    string
		setup        func(store *catalogContractMetadataStore) (datastore.ContractMetadata, datastore.ContractMetadata)
		createFilter func(metadata1, metadata2 datastore.ContractMetadata) datastore.FilterFunc[datastore.ContractMetadataKey, datastore.ContractMetadata]
		minExpected  int
		verify       func(t *testing.T, results []datastore.ContractMetadata, metadata1, metadata2 datastore.ContractMetadata)
	}{
		{
			name:      "fetch_all",
			operation: "fetch",
			setup: func(store *catalogContractMetadataStore) (datastore.ContractMetadata, datastore.ContractMetadata) {
				// Setup test data with unique chain selectors
				metadata1 := newRandomContractMetadata()
				chainSelector1 := generateRandomContractChainSelector()
				metadata1.ChainSelector = chainSelector1
				err := store.Add(t.Context(), metadata1)
				require.NoError(t, err)

				metadata2 := newRandomContractMetadata()
				chainSelector2 := generateRandomContractChainSelector()
				// Ensure different chain selectors
				for chainSelector2 == chainSelector1 {
					chainSelector2 = generateRandomContractChainSelector()
				}
				metadata2.ChainSelector = chainSelector2
				err = store.Add(t.Context(), metadata2)
				require.NoError(t, err)

				return metadata1, metadata2
			},
			createFilter: nil,
			minExpected:  2,
			verify: func(t *testing.T, results []datastore.ContractMetadata, metadata1, metadata2 datastore.ContractMetadata) {
				t.Helper()
				// Check that our records are in the results
				foundFirst := false
				foundSecond := false
				for _, result := range results {
					if result.Address == metadata1.Address && result.ChainSelector == metadata1.ChainSelector {
						foundFirst = true
					}
					if result.Address == metadata2.Address && result.ChainSelector == metadata2.ChainSelector {
						foundSecond = true
					}
				}
				require.True(t, foundFirst, "First contract metadata not found in fetch results")
				require.True(t, foundSecond, "Second contract metadata not found in fetch results")
			},
		},
		{
			name:      "filter_by_chain_selector",
			operation: "filter",
			setup: func(store *catalogContractMetadataStore) (datastore.ContractMetadata, datastore.ContractMetadata) {
				// Setup test data with unique chain selectors
				metadata1 := newRandomContractMetadata()
				chainSelector1 := generateRandomContractChainSelector()
				metadata1.ChainSelector = chainSelector1
				err := store.Add(t.Context(), metadata1)
				require.NoError(t, err)

				metadata2 := newRandomContractMetadata()
				chainSelector2 := generateRandomContractChainSelector()
				// Ensure different chain selectors
				for chainSelector2 == chainSelector1 {
					chainSelector2 = generateRandomContractChainSelector()
				}
				metadata2.ChainSelector = chainSelector2
				err = store.Add(t.Context(), metadata2)
				require.NoError(t, err)

				return metadata1, metadata2
			},
			createFilter: func(metadata1, metadata2 datastore.ContractMetadata) datastore.FilterFunc[datastore.ContractMetadataKey, datastore.ContractMetadata] {
				// Use the proper filter from datastore/filters.go
				return datastore.ContractMetadataByChainSelector(metadata1.ChainSelector)
			},
			minExpected: 1,
			verify: func(t *testing.T, results []datastore.ContractMetadata, metadata1, metadata2 datastore.ContractMetadata) {
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
			store := setupTestContractStore(t)

			metadata1, metadata2 := tt.setup(store)

			var results []datastore.ContractMetadata
			var err error

			// Execute operation
			switch tt.operation {
			case "fetch":
				results, err = store.Fetch(t.Context())
			case "filter":
				var filterFunc datastore.FilterFunc[datastore.ContractMetadataKey, datastore.ContractMetadata]
				if tt.createFilter != nil {
					filterFunc = tt.createFilter(metadata1, metadata2)
				}
				results, err = store.Filter(t.Context(), filterFunc)
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

func TestCatalogContractMetadataStore_ConversionHelpers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T, store *catalogContractMetadataStore)
	}{
		{
			name: "keyToFilter",
			test: func(t *testing.T, store *catalogContractMetadataStore) {
				t.Helper()
				key := datastore.NewContractMetadataKey(12345, "0x1234567890abcdef1234567890abcdef12345678")

				filter := store.keyToFilter(key)

				require.Equal(t, "test-domain", filter.Domain.Value)
				require.Equal(t, "catalog_testing", filter.Environment.Value)
				require.Equal(t, uint64(12345), filter.ChainSelector.Value)
				require.Equal(t, "0x1234567890abcdef1234567890abcdef12345678", filter.Address.Value)
			},
		},
		{
			name: "protoToContractMetadata_success",
			test: func(t *testing.T, store *catalogContractMetadataStore) {
				t.Helper()
				protoMetadata := &pb.ContractMetadata{
					Domain:        "test-domain",
					Environment:   "catalog_testing",
					ChainSelector: 12345,
					Address:       "0x1234567890abcdef1234567890abcdef12345678",
					Metadata:      `{"name":"TestContract","version":"1.0.0"}`,
					RowVersion:    1,
				}

				metadata, err := store.protoToContractMetadata(protoMetadata)

				require.NoError(t, err)
				require.Equal(t, "0x1234567890abcdef1234567890abcdef12345678", metadata.Address)
				require.Equal(t, uint64(12345), metadata.ChainSelector)
				require.NotNil(t, metadata.Metadata)

				// Check JSON unmarshaling - it will be unmarshaled as map[string]interface{}
				// since that's what json.Unmarshal defaults to for interface{}
				metadataMap := metadata.Metadata.(map[string]interface{})
				require.Equal(t, "TestContract", metadataMap["name"])
				require.Equal(t, "1.0.0", metadataMap["version"])
			},
		},
		{
			name: "protoToContractMetadata_invalid_json",
			test: func(t *testing.T, store *catalogContractMetadataStore) {
				t.Helper()
				protoMetadata := &pb.ContractMetadata{
					Domain:        "test-domain",
					Environment:   "catalog_testing",
					ChainSelector: 12345,
					Address:       "0x1234567890abcdef1234567890abcdef12345678",
					Metadata:      `{invalid-json`,
					RowVersion:    1,
				}

				_, err := store.protoToContractMetadata(protoMetadata)

				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to unmarshal metadata JSON")
			},
		},
		{
			name: "contractMetadataToProto",
			test: func(t *testing.T, store *catalogContractMetadataStore) {
				t.Helper()
				metadata := newRandomContractMetadata()

				protoMetadata := store.contractMetadataToProto(metadata, 0)

				require.Equal(t, "test-domain", protoMetadata.Domain)
				require.Equal(t, "catalog_testing", protoMetadata.Environment)
				require.Equal(t, metadata.ChainSelector, protoMetadata.ChainSelector)
				require.Equal(t, metadata.Address, protoMetadata.Address)
				require.NotEmpty(t, protoMetadata.Metadata)
				require.Equal(t, int32(0), protoMetadata.RowVersion) // Should be 0 initially

				// Verify JSON marshaling worked
				require.Contains(t, protoMetadata.Metadata, "name")
				require.Contains(t, protoMetadata.Metadata, "version")
			},
		},
		{
			name: "version_handling",
			test: func(t *testing.T, store *catalogContractMetadataStore) {
				t.Helper()
				// Test protoToContractMetadata with version
				protoMetadata := &pb.ContractMetadata{
					Domain:        "test-domain",
					Environment:   "catalog_testing",
					ChainSelector: 12345,
					Address:       "0x1234567890abcdef1234567890abcdef12345678",
					Metadata:      `{"name":"TestContract","version":"1.0.0"}`,
					RowVersion:    5,
				}

				metadata, err := store.protoToContractMetadata(protoMetadata)
				require.NoError(t, err)

				// Test contractMetadataToProto with specific version
				protoResult := store.contractMetadataToProto(metadata, 7)
				require.Equal(t, int32(7), protoResult.RowVersion)

				// Test with version 0 (default for new records)
				protoResult0 := store.contractMetadataToProto(metadata, 0)
				require.Equal(t, int32(0), protoResult0.RowVersion)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh store for each test case to avoid concurrency issues
			store := setupTestContractStore(t)

			tt.test(t, store)
		})
	}
}

// Test updater functions that demonstrate different patterns for MetadataUpdaterF

// wholeContractMetadataMerger demonstrates merging two complete TestContractMetadata structs
func wholeContractMetadataMerger() datastore.MetadataUpdaterF {
	return func(latest any, incoming any) (any, error) {
		// Both latest and incoming are complete TestContractMetadata structs
		latestMeta, err := datastore.As[TestContractMetadata](latest)
		if err != nil {
			return nil, err
		}

		incomingMeta, err := datastore.As[TestContractMetadata](incoming)
		if err != nil {
			return nil, err
		}

		// Merge logic - keep some fields from latest, update others from incoming
		merged := TestContractMetadata{
			Name:        incomingMeta.Name,                             // Always update name
			Version:     incomingMeta.Version,                          // Always update version
			Description: incomingMeta.Description,                      // Always update description
			Tags:        append(latestMeta.Tags, incomingMeta.Tags...), // Merge tags
		}

		return merged, nil
	}
}

// versionOnlyUpdater demonstrates updating only the version field
func versionOnlyUpdater() datastore.MetadataUpdaterF {
	return func(latest any, incoming any) (any, error) {
		// latest is full metadata, incoming is just a string version
		latestMeta, err := datastore.As[TestContractMetadata](latest)
		if err != nil {
			return nil, err
		}

		newVersion, err := datastore.As[string](incoming)
		if err != nil {
			return nil, err
		}

		// Update only the version field
		updated := latestMeta
		updated.Version = newVersion

		return updated, nil
	}
}

// smartContractTagMerger demonstrates intelligent tag merging without duplicates
func smartContractTagMerger() datastore.MetadataUpdaterF {
	return func(latest any, incoming any) (any, error) {
		// latest is full metadata, incoming is just new tags to add
		latestMeta, err := datastore.As[TestContractMetadata](latest)
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

func TestCatalogContractMetadataStore_UpdaterExamples(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		latest   any
		incoming any
		updater  datastore.MetadataUpdaterF
		verify   func(t *testing.T, result any)
	}{
		{
			name: "whole_metadata_merge",
			latest: TestContractMetadata{
				Name:        "OriginalContract",
				Version:     "1.0.0",
				Description: "Original contract description",
				Tags:        []string{"old"},
			},
			incoming: TestContractMetadata{
				Name:        "NewContract",
				Version:     "2.0.0",
				Description: "New contract description",
				Tags:        []string{"new"},
			},
			updater: wholeContractMetadataMerger(),
			verify: func(t *testing.T, result any) {
				t.Helper()
				merged, err := datastore.As[TestContractMetadata](result)
				require.NoError(t, err)
				require.Equal(t, "NewContract", merged.Name)
				require.Equal(t, "2.0.0", merged.Version)
				require.Equal(t, "New contract description", merged.Description)
				require.Contains(t, merged.Tags, "old")
				require.Contains(t, merged.Tags, "new")
			},
		},
		{
			name: "version_only_update",
			latest: TestContractMetadata{
				Name:        "OriginalContract",
				Version:     "1.0.0",
				Description: "Original description",
				Tags:        []string{"old"},
			},
			incoming: "3.0.0",
			updater:  versionOnlyUpdater(),
			verify: func(t *testing.T, result any) {
				t.Helper()
				updated, err := datastore.As[TestContractMetadata](result)
				require.NoError(t, err)
				require.Equal(t, "OriginalContract", updated.Name)            // Should be unchanged
				require.Equal(t, "3.0.0", updated.Version)                    // Should be updated
				require.Equal(t, "Original description", updated.Description) // Should be unchanged
				require.Equal(t, []string{"old"}, updated.Tags)               // Should be unchanged
			},
		},
		{
			name: "smart_tag_merging",
			latest: TestContractMetadata{
				Name:        "OriginalContract",
				Version:     "1.0.0",
				Description: "Original description",
				Tags:        []string{"old"},
			},
			incoming: []string{"new", "additional", "old"}, // "old" should not duplicate
			updater:  smartContractTagMerger(),
			verify: func(t *testing.T, result any) {
				t.Helper()
				updated, err := datastore.As[TestContractMetadata](result)
				require.NoError(t, err)
				require.Equal(t, "OriginalContract", updated.Name)
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

			// Execute the updater
			result, err := tt.updater(tt.latest, tt.incoming)
			require.NoError(t, err)

			// Verify the result
			tt.verify(t, result)
		})
	}
}

func TestCatalogContractMetadataStore_Update_WithCustomUpdater(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		setup        func(store *catalogContractMetadataStore) datastore.ContractMetadataKey
		incomingData any
		updater      datastore.MetadataUpdaterF
		expectError  bool
		errorType    error
		verifyResult func(t *testing.T, store *catalogContractMetadataStore, key datastore.ContractMetadataKey)
	}{
		{
			name: "update_with_description_updater",
			setup: func(store *catalogContractMetadataStore) datastore.ContractMetadataKey {
				metadata := newRandomContractMetadata()
				err := store.Add(t.Context(), metadata)
				require.NoError(t, err)

				return datastore.NewContractMetadataKey(metadata.ChainSelector, metadata.Address)
			},
			incomingData: "Updated description for contract",
			updater:      versionOnlyUpdater(), // Reuse existing updater but for description
			expectError:  false,
			verifyResult: func(t *testing.T, store *catalogContractMetadataStore, key datastore.ContractMetadataKey) {
				t.Helper()
				result, err := store.Get(t.Context(), key)
				require.NoError(t, err)

				metadata, err := datastore.As[TestContractMetadata](result.Metadata)
				require.NoError(t, err)
				require.Equal(t, "Updated description for contract", metadata.Version) // Using version field as proxy
			},
		},
		{
			name: "update_with_tag_merger",
			setup: func(store *catalogContractMetadataStore) datastore.ContractMetadataKey {
				metadata := newRandomContractMetadata()
				// Ensure we have some initial tags
				testMeta := metadata.Metadata.(TestContractMetadata)
				testMeta.Tags = []string{"existing", "initial"}
				metadata.Metadata = testMeta
				err := store.Add(t.Context(), metadata)
				require.NoError(t, err)

				return datastore.NewContractMetadataKey(metadata.ChainSelector, metadata.Address)
			},
			incomingData: []string{"new", "updated", "existing"}, // "existing" should not duplicate
			updater:      smartContractTagMerger(),
			expectError:  false,
			verifyResult: func(t *testing.T, store *catalogContractMetadataStore, key datastore.ContractMetadataKey) {
				t.Helper()
				result, err := store.Get(t.Context(), key)
				require.NoError(t, err)

				metadata, err := datastore.As[TestContractMetadata](result.Metadata)
				require.NoError(t, err)
				require.Contains(t, metadata.Tags, "existing")
				require.Contains(t, metadata.Tags, "initial")
				require.Contains(t, metadata.Tags, "new")
				require.Contains(t, metadata.Tags, "updated")
				require.Len(t, metadata.Tags, 4) // No duplicates
			},
		},
		{
			name: "update_with_whole_metadata_merger",
			setup: func(store *catalogContractMetadataStore) datastore.ContractMetadataKey {
				metadata := newRandomContractMetadata()
				err := store.Add(t.Context(), metadata)
				require.NoError(t, err)

				return datastore.NewContractMetadataKey(metadata.ChainSelector, metadata.Address)
			},
			incomingData: TestContractMetadata{
				Name:        "MergedContract",
				Version:     "3.0.0",
				Description: "Merged contract description",
				Tags:        []string{"merged", "updated"},
			},
			updater:     wholeContractMetadataMerger(),
			expectError: false,
			verifyResult: func(t *testing.T, store *catalogContractMetadataStore, key datastore.ContractMetadataKey) {
				t.Helper()
				result, err := store.Get(t.Context(), key)
				require.NoError(t, err)

				metadata, err := datastore.As[TestContractMetadata](result.Metadata)
				require.NoError(t, err)
				require.Equal(t, "MergedContract", metadata.Name)
				require.Equal(t, "3.0.0", metadata.Version)
				require.Equal(t, "Merged contract description", metadata.Description)
				require.Contains(t, metadata.Tags, "merged")
				require.Contains(t, metadata.Tags, "updated")
				// Should also contain original tags due to merging
				require.Contains(t, metadata.Tags, "test")
				require.Contains(t, metadata.Tags, "integration")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := setupTestContractStore(t)

			key := tt.setup(store)

			// Execute update with custom updater
			err := store.Update(t.Context(), key, tt.incomingData, datastore.WithUpdater(tt.updater))

			// Verify
			if tt.expectError {
				require.Error(t, err)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
				if tt.verifyResult != nil {
					tt.verifyResult(t, store, key)
				}
			}
		})
	}
}

func TestCatalogContractMetadataStore_Upsert_WithCustomUpdater(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		setup        func(store *catalogContractMetadataStore) datastore.ContractMetadataKey
		incomingData any
		updater      datastore.MetadataUpdaterF
		expectError  bool
		errorType    error
		verifyResult func(t *testing.T, store *catalogContractMetadataStore, key datastore.ContractMetadataKey)
	}{
		{
			name: "update_existing_with_description_updater",
			setup: func(store *catalogContractMetadataStore) datastore.ContractMetadataKey {
				metadata := newRandomContractMetadata()
				err := store.Add(t.Context(), metadata)
				require.NoError(t, err)

				return datastore.NewContractMetadataKey(metadata.ChainSelector, metadata.Address)
			},
			incomingData: "5.0.0", // New version
			updater:      versionOnlyUpdater(),
			expectError:  false,
			verifyResult: func(t *testing.T, store *catalogContractMetadataStore, key datastore.ContractMetadataKey) {
				t.Helper()
				result, err := store.Get(t.Context(), key)
				require.NoError(t, err)

				metadata, err := datastore.As[TestContractMetadata](result.Metadata)
				require.NoError(t, err)
				require.Equal(t, "5.0.0", metadata.Version) // Should be updated
				// Other fields should remain from original
				require.Contains(t, metadata.Name, "TestContract")
				require.Equal(t, "Test contract for integration testing", metadata.Description)
			},
		},
		{
			name: "update_existing_with_tag_merger",
			setup: func(store *catalogContractMetadataStore) datastore.ContractMetadataKey {
				metadata := newRandomContractMetadata()
				// Ensure we have some initial tags
				testMeta := metadata.Metadata.(TestContractMetadata)
				testMeta.Tags = []string{"original", "base"}
				metadata.Metadata = testMeta
				err := store.Add(t.Context(), metadata)
				require.NoError(t, err)

				return datastore.NewContractMetadataKey(metadata.ChainSelector, metadata.Address)
			},
			incomingData: []string{"enhanced", "improved", "original"}, // "original" should not duplicate
			updater:      smartContractTagMerger(),
			expectError:  false,
			verifyResult: func(t *testing.T, store *catalogContractMetadataStore, key datastore.ContractMetadataKey) {
				t.Helper()
				result, err := store.Get(t.Context(), key)
				require.NoError(t, err)

				metadata, err := datastore.As[TestContractMetadata](result.Metadata)
				require.NoError(t, err)
				require.Contains(t, metadata.Tags, "original")
				require.Contains(t, metadata.Tags, "base")
				require.Contains(t, metadata.Tags, "enhanced")
				require.Contains(t, metadata.Tags, "improved")
				require.Len(t, metadata.Tags, 4) // No duplicates
			},
		},
		{
			name: "update_existing_with_whole_metadata_merger",
			setup: func(store *catalogContractMetadataStore) datastore.ContractMetadataKey {
				metadata := newRandomContractMetadata()
				err := store.Add(t.Context(), metadata)
				require.NoError(t, err)

				return datastore.NewContractMetadataKey(metadata.ChainSelector, metadata.Address)
			},
			incomingData: TestContractMetadata{
				Name:        "FullyMergedContract",
				Version:     "4.0.0",
				Description: "Fully merged via upsert",
				Tags:        []string{"merged", "comprehensive"},
			},
			updater:     wholeContractMetadataMerger(),
			expectError: false,
			verifyResult: func(t *testing.T, store *catalogContractMetadataStore, key datastore.ContractMetadataKey) {
				t.Helper()
				result, err := store.Get(t.Context(), key)
				require.NoError(t, err)

				metadata, err := datastore.As[TestContractMetadata](result.Metadata)
				require.NoError(t, err)
				require.Equal(t, "FullyMergedContract", metadata.Name)
				require.Equal(t, "4.0.0", metadata.Version)
				require.Equal(t, "Fully merged via upsert", metadata.Description)
				require.Contains(t, metadata.Tags, "merged")
				require.Contains(t, metadata.Tags, "comprehensive")
				// Should also contain original tags due to merging
				require.Contains(t, metadata.Tags, "test")
				require.Contains(t, metadata.Tags, "integration")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := setupTestContractStore(t)

			key := tt.setup(store)

			// Execute upsert with custom updater
			err := store.Upsert(t.Context(), key, tt.incomingData, datastore.WithUpdater(tt.updater))

			// Verify
			if tt.expectError {
				require.Error(t, err)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
				if tt.verifyResult != nil {
					tt.verifyResult(t, store, key)
				}
			}
		})
	}
}
