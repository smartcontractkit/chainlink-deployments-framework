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
	pb "github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/protos"
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
func setupTestContractStore(t *testing.T) (*CatalogContractMetadataStore, *grpc.ClientConn) {
	t.Helper()
	// Get gRPC address from environment or use default
	address := os.Getenv("CATALOG_GRPC_ADDRESS")
	if address == "" {
		address = defaultContractGRPCAddress
	}

	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Skipf("Failed to connect to gRPC server at %s: %v. Skipping integration tests.", address, err)
		return nil, nil
	}

	// Create client
	client := pb.NewDeploymentsDatastoreClient(conn)

	// Test if the gRPC service is actually available by making a simple call
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := client.DataAccess(ctx)
	if err != nil {
		conn.Close()
		t.Skipf("gRPC service not available at %s: %v. Skipping integration tests.", address, err)

		return nil, nil
	}
	if stream != nil {
		_ = stream.CloseSend()
	}

	// Create store
	store := NewCatalogContractMetadataStore(CatalogContractMetadataStoreConfig{
		Domain:      "test-domain",
		Environment: "catalog_testing",
		Client:      client,
	})

	return store, conn
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
		setup       func(store *CatalogContractMetadataStore) datastore.ContractMetadataKey
		expectError bool
		errorType   error
	}{
		{
			name: "not_found",
			setup: func(store *CatalogContractMetadataStore) datastore.ContractMetadataKey {
				// Use a unique key that shouldn't exist
				return datastore.NewContractMetadataKey(99999999, "0xnonexistent1234567890abcdef1234567890ab")
			},
			expectError: true,
			errorType:   datastore.ErrContractMetadataNotFound,
		},
		{
			name: "success",
			setup: func(store *CatalogContractMetadataStore) datastore.ContractMetadataKey {
				// Create and add a record first
				metadata := newRandomContractMetadata()
				err := store.Add(metadata)
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
			store, conn := setupTestContractStore(t)
			defer conn.Close()

			key := tt.setup(store)

			// Execute
			result, err := store.Get(key)

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
				require.NotNil(t, result.Metadata)
			}
		})
	}
}

func TestCatalogContractMetadataStore_Add(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(store *CatalogContractMetadataStore) datastore.ContractMetadata
		expectError bool
		errorCheck  func(error) bool
	}{
		{
			name: "success",
			setup: func(store *CatalogContractMetadataStore) datastore.ContractMetadata {
				return newRandomContractMetadata()
			},
			expectError: false,
		},
		{
			name: "duplicate_error",
			setup: func(store *CatalogContractMetadataStore) datastore.ContractMetadata {
				// Create and add a record first
				metadata := newRandomContractMetadata()
				err := store.Add(metadata)
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
			store, conn := setupTestContractStore(t)
			defer conn.Close()

			metadata := tt.setup(store)

			// Execute
			err := store.Add(metadata)

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
				retrieved, err := store.Get(key)
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
		setup       func(store *CatalogContractMetadataStore) datastore.ContractMetadata
		expectError bool
		errorType   error
		verify      func(t *testing.T, store *CatalogContractMetadataStore, metadata datastore.ContractMetadata)
	}{
		{
			name: "success",
			setup: func(store *CatalogContractMetadataStore) datastore.ContractMetadata {
				// Create and add contract metadata
				metadata := newRandomContractMetadata()
				err := store.Add(metadata)
				require.NoError(t, err)

				// Fetch the record to get the current version in cache
				fetchedMetadata, err := store.Get(metadata.Key())
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
			verify: func(t *testing.T, store *CatalogContractMetadataStore, metadata datastore.ContractMetadata) {
				t.Helper()
				// Verify the updated values
				key := datastore.NewContractMetadataKey(metadata.ChainSelector, metadata.Address)
				retrieved, err := store.Get(key)
				require.NoError(t, err)

				concrete, err := datastore.As[TestContractMetadata](retrieved.Metadata)
				require.NoError(t, err)
				// Check that the metadata matches
				require.Equal(t, metadata.Metadata, concrete)
			},
		},
		{
			name: "not_found",
			setup: func(store *CatalogContractMetadataStore) datastore.ContractMetadata {
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
			store, conn := setupTestContractStore(t)
			defer conn.Close()

			metadata := tt.setup(store)

			// Execute update
			err := store.Update(metadata)

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
	store1, conn1 := setupTestContractStore(t)
	defer conn1.Close()

	store2, conn2 := setupTestContractStore(t)
	defer conn2.Close()

	// Add a contract metadata record using store1
	original := newRandomContractMetadata()
	err := store1.Add(original)
	require.NoError(t, err)

	// Both stores get the record to populate their caches with version 1
	key := datastore.NewContractMetadataKey(original.ChainSelector, original.Address)
	first, err := store1.Get(key)
	require.NoError(t, err)

	second, err := store2.Get(key)
	require.NoError(t, err)

	// Store1 updates the record (this increments server version to 2)
	updatedMetadata := newTestContractMetadata("FirstUpdate")
	updatedMetadata.Version = "2.0.0"
	first.Metadata = updatedMetadata
	err = store1.Update(first)
	require.NoError(t, err)

	// Store2 tries to update using its cached version (still version 1, now stale)
	staleMetadata := newTestContractMetadata("StaleUpdate")
	staleMetadata.Version = "3.0.0"
	second.Metadata = staleMetadata

	// Execute update with store2 (should fail due to stale version)
	err = store2.Update(second)

	// Verify we get the expected stale version error
	require.Error(t, err)
	require.ErrorIs(t, err, datastore.ErrContractMetadataStale)
}

func TestCatalogContractMetadataStore_Upsert(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(store *CatalogContractMetadataStore) datastore.ContractMetadata
		expectError bool
		errorType   error
		verify      func(t *testing.T, store *CatalogContractMetadataStore, original datastore.ContractMetadata)
	}{
		{
			name: "insert_new_record",
			setup: func(store *CatalogContractMetadataStore) datastore.ContractMetadata {
				// Create a unique contract metadata for this test
				return newRandomContractMetadata()
			},
			expectError: false,
			verify: func(t *testing.T, store *CatalogContractMetadataStore, original datastore.ContractMetadata) {
				t.Helper()
				// Verify we can get it back
				key := datastore.NewContractMetadataKey(original.ChainSelector, original.Address)
				retrieved, err := store.Get(key)
				require.NoError(t, err)

				concrete, err := datastore.As[TestContractMetadata](retrieved.Metadata)
				require.NoError(t, err)
				require.Equal(t, original.Metadata, concrete)
			},
		},
		{
			name: "update_existing_record",
			setup: func(store *CatalogContractMetadataStore) datastore.ContractMetadata {
				// Create and add contract metadata
				metadata := newRandomContractMetadata()
				err := store.Add(metadata)
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
			verify: func(t *testing.T, store *CatalogContractMetadataStore, modified datastore.ContractMetadata) {
				t.Helper()
				// Verify the updated values
				key := datastore.NewContractMetadataKey(modified.ChainSelector, modified.Address)
				retrieved, err := store.Get(key)
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
			store, conn := setupTestContractStore(t)
			defer conn.Close()

			metadata := tt.setup(store)

			// Execute upsert
			err := store.Upsert(metadata)

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
	store1, conn1 := setupTestContractStore(t)
	defer conn1.Close()

	store2, conn2 := setupTestContractStore(t)
	defer conn2.Close()

	// Add a contract metadata record using store1
	original := newRandomContractMetadata()
	err := store1.Add(original)
	require.NoError(t, err)

	// Both stores get the record to populate their caches with version 1
	key := datastore.NewContractMetadataKey(original.ChainSelector, original.Address)
	first, err := store1.Get(key)
	require.NoError(t, err)

	second, err := store2.Get(key)
	require.NoError(t, err)

	// Store1 updates the record (this increments server version to 2)
	updatedMetadata := newTestContractMetadata("FirstUpdate")
	updatedMetadata.Version = "2.0.0"
	first.Metadata = updatedMetadata
	err = store1.Update(first)
	require.NoError(t, err)

	// Store2 tries to upsert using its cached version (still version 1, now stale)
	staleMetadata := newTestContractMetadata("UpsertStaleUpdate")
	staleMetadata.Version = "3.0.0"
	second.Metadata = staleMetadata

	// Execute upsert with store2 (should fail due to stale version)
	err = store2.Upsert(second)

	// Verify we get the expected stale version error
	require.Error(t, err)
	require.ErrorIs(t, err, datastore.ErrContractMetadataStale)
}

func TestCatalogContractMetadataStore_Delete(t *testing.T) {
	t.Parallel()
	store, conn := setupTestContractStore(t)
	defer conn.Close()

	key := datastore.NewContractMetadataKey(12345, "0x1234567890abcdef1234567890abcdef12345678")

	// Execute
	err := store.Delete(key)

	// Verify
	require.Error(t, err)
	require.Contains(t, err.Error(), "delete operation not supported")
}

func TestCatalogContractMetadataStore_FetchAndFilter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		operation    string
		setup        func(store *CatalogContractMetadataStore) (datastore.ContractMetadata, datastore.ContractMetadata)
		createFilter func(metadata1, metadata2 datastore.ContractMetadata) datastore.FilterFunc[datastore.ContractMetadataKey, datastore.ContractMetadata]
		minExpected  int
		verify       func(t *testing.T, results []datastore.ContractMetadata, metadata1, metadata2 datastore.ContractMetadata)
	}{
		{
			name:      "fetch_all",
			operation: "fetch",
			setup: func(store *CatalogContractMetadataStore) (datastore.ContractMetadata, datastore.ContractMetadata) {
				// Setup test data with unique chain selectors
				metadata1 := newRandomContractMetadata()
				chainSelector1 := generateRandomContractChainSelector()
				metadata1.ChainSelector = chainSelector1
				err := store.Add(metadata1)
				require.NoError(t, err)

				metadata2 := newRandomContractMetadata()
				chainSelector2 := generateRandomContractChainSelector()
				// Ensure different chain selectors
				for chainSelector2 == chainSelector1 {
					chainSelector2 = generateRandomContractChainSelector()
				}
				metadata2.ChainSelector = chainSelector2
				err = store.Add(metadata2)
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
			setup: func(store *CatalogContractMetadataStore) (datastore.ContractMetadata, datastore.ContractMetadata) {
				// Setup test data with unique chain selectors
				metadata1 := newRandomContractMetadata()
				chainSelector1 := generateRandomContractChainSelector()
				metadata1.ChainSelector = chainSelector1
				err := store.Add(metadata1)
				require.NoError(t, err)

				metadata2 := newRandomContractMetadata()
				chainSelector2 := generateRandomContractChainSelector()
				// Ensure different chain selectors
				for chainSelector2 == chainSelector1 {
					chainSelector2 = generateRandomContractChainSelector()
				}
				metadata2.ChainSelector = chainSelector2
				err = store.Add(metadata2)
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
			store, conn := setupTestContractStore(t)
			defer conn.Close()

			metadata1, metadata2 := tt.setup(store)

			var results []datastore.ContractMetadata
			var err error

			// Execute operation
			switch tt.operation {
			case "fetch":
				results, err = store.Fetch()
			case "filter":
				var filterFunc datastore.FilterFunc[datastore.ContractMetadataKey, datastore.ContractMetadata]
				if tt.createFilter != nil {
					filterFunc = tt.createFilter(metadata1, metadata2)
				}
				results = store.Filter(filterFunc)
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

func TestCatalogContractMetadataStore_ConversionHelpers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T, store *CatalogContractMetadataStore)
	}{
		{
			name: "keyToFilter",
			test: func(t *testing.T, store *CatalogContractMetadataStore) {
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
			test: func(t *testing.T, store *CatalogContractMetadataStore) {
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
			test: func(t *testing.T, store *CatalogContractMetadataStore) {
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
			test: func(t *testing.T, store *CatalogContractMetadataStore) {
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
			test: func(t *testing.T, store *CatalogContractMetadataStore) {
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
			store, conn := setupTestContractStore(t)
			defer conn.Close()

			tt.test(t, store)
		})
	}
}
