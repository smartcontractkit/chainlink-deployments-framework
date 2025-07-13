package catalog

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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

	// Create store
	store := NewCatalogContractMetadataStore(CatalogContractMetadataStoreConfig{
		Domain:      "test-domain",
		Environment: "catalog_testing",
		Client:      client,
	})

	return store, conn
}

// skipIfNoContractService skips the test if we can't connect to the gRPC service
func skipIfNoContractService(t *testing.T, conn *grpc.ClientConn) {
	if conn == nil {
		t.Skip("Skipping test: gRPC service not available")
	}
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
	max := big.NewInt(999999999) // Large but reasonable upper bound
	n, err := rand.Int(rand.Reader, max)
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
	}
}

func TestCatalogContractMetadataStore_Get(t *testing.T) {
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
			// Create a fresh store for each test case to avoid concurrency issues
			store, conn := setupTestContractStore(t)
			skipIfNoContractService(t, conn)
			defer conn.Close()

			key := tt.setup(store)

			// Execute
			result, err := store.Get(key)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, key.ChainSelector(), result.ChainSelector)
				assert.Equal(t, key.Address(), result.Address)
				assert.NotNil(t, result.Metadata)
			}
		})
	}
}

func TestCatalogContractMetadataStore_Add(t *testing.T) {
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
			// Create a fresh store for each test case to avoid concurrency issues
			store, conn := setupTestContractStore(t)
			skipIfNoContractService(t, conn)
			defer conn.Close()

			metadata := tt.setup(store)

			// Execute
			err := store.Add(metadata)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorCheck != nil {
					assert.True(t, tt.errorCheck(err))
				}
			} else {
				assert.NoError(t, err)

				// Verify we can get it back
				key := datastore.NewContractMetadataKey(metadata.ChainSelector, metadata.Address)
				retrieved, err := store.Get(key)
				require.NoError(t, err)

				assert.Equal(t, metadata.Address, retrieved.Address)
				assert.Equal(t, metadata.ChainSelector, retrieved.ChainSelector)

				concrete, err := datastore.As[TestContractMetadata](retrieved.Metadata)
				require.NoError(t, err)
				// Check that the metadata matches
				assert.Equal(t, metadata.Metadata, concrete)
			}
		})
	}
}

func TestCatalogContractMetadataStore_Update(t *testing.T) {
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

				// Modify the metadata
				updatedMetadata := newTestContractMetadata("UpdatedContract")
				updatedMetadata.Version = "2.0.0"
				updatedMetadata.Description = "Updated test contract"
				updatedMetadata.Tags = []string{"test", "updated"}
				metadata.Metadata = updatedMetadata
				return metadata
			},
			expectError: false,
			verify: func(t *testing.T, store *CatalogContractMetadataStore, metadata datastore.ContractMetadata) {
				// Verify the updated values
				key := datastore.NewContractMetadataKey(metadata.ChainSelector, metadata.Address)
				retrieved, err := store.Get(key)
				require.NoError(t, err)

				concrete, err := datastore.As[TestContractMetadata](retrieved.Metadata)
				require.NoError(t, err)
				assert.Equal(t, metadata.Metadata, concrete)
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
			// Create a fresh store for each test case to avoid concurrency issues
			store, conn := setupTestContractStore(t)
			skipIfNoContractService(t, conn)
			defer conn.Close()

			metadata := tt.setup(store)

			// Execute update
			err := store.Update(metadata)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
				if tt.verify != nil {
					tt.verify(t, store, metadata)
				}
			}
		})
	}
}

func TestCatalogContractMetadataStore_Upsert(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(store *CatalogContractMetadataStore) datastore.ContractMetadata
		verify func(t *testing.T, store *CatalogContractMetadataStore, original datastore.ContractMetadata)
	}{
		{
			name: "insert_new_record",
			setup: func(store *CatalogContractMetadataStore) datastore.ContractMetadata {
				// Create a unique contract metadata for this test
				return newRandomContractMetadata()
			},
			verify: func(t *testing.T, store *CatalogContractMetadataStore, original datastore.ContractMetadata) {
				// Verify we can get it back
				key := datastore.NewContractMetadataKey(original.ChainSelector, original.Address)
				retrieved, err := store.Get(key)
				require.NoError(t, err)

				concrete, err := datastore.As[TestContractMetadata](retrieved.Metadata)
				require.NoError(t, err)
				assert.Equal(t, original.Metadata, concrete)
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
			verify: func(t *testing.T, store *CatalogContractMetadataStore, modified datastore.ContractMetadata) {
				// Verify the updated values
				key := datastore.NewContractMetadataKey(modified.ChainSelector, modified.Address)
				retrieved, err := store.Get(key)
				require.NoError(t, err)

				concrete, err := datastore.As[TestContractMetadata](retrieved.Metadata)
				require.NoError(t, err)
				assert.Equal(t, modified.Metadata, concrete)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh store for each test case to avoid concurrency issues
			store, conn := setupTestContractStore(t)
			skipIfNoContractService(t, conn)
			defer conn.Close()

			metadata := tt.setup(store)

			// Execute upsert
			err := store.Upsert(metadata)

			// Verify
			assert.NoError(t, err)
			tt.verify(t, store, metadata)
		})
	}
}

func TestCatalogContractMetadataStore_Delete(t *testing.T) {
	store, conn := setupTestContractStore(t)
	skipIfNoContractService(t, conn)
	defer conn.Close()

	key := datastore.NewContractMetadataKey(12345, "0x1234567890abcdef1234567890abcdef12345678")

	// Execute
	err := store.Delete(key)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delete operation not supported")
}

func TestCatalogContractMetadataStore_FetchAndFilter(t *testing.T) {
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
				assert.True(t, foundFirst, "First contract metadata not found in fetch results")
				assert.True(t, foundSecond, "Second contract metadata not found in fetch results")
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
				// All results should have the chain selector from metadata1
				for _, result := range results {
					assert.Equal(t, metadata1.ChainSelector, result.ChainSelector)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh store for each test case to avoid concurrency issues
			store, conn := setupTestContractStore(t)
			skipIfNoContractService(t, conn)
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
				assert.NoError(t, err)
			}
			assert.GreaterOrEqual(t, len(results), tt.minExpected)
			if tt.verify != nil {
				tt.verify(t, results, metadata1, metadata2)
			}
		})
	}
}

func TestCatalogContractMetadataStore_ConversionHelpers(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T, store *CatalogContractMetadataStore)
	}{
		{
			name: "keyToFilter",
			test: func(t *testing.T, store *CatalogContractMetadataStore) {
				key := datastore.NewContractMetadataKey(12345, "0x1234567890abcdef1234567890abcdef12345678")

				filter := store.keyToFilter(key)

				assert.Equal(t, "test-domain", filter.Domain.Value)
				assert.Equal(t, "catalog_testing", filter.Environment.Value)
				assert.Equal(t, uint64(12345), filter.ChainSelector.Value)
				assert.Equal(t, "0x1234567890abcdef1234567890abcdef12345678", filter.Address.Value)
			},
		},
		{
			name: "protoToContractMetadata_success",
			test: func(t *testing.T, store *CatalogContractMetadataStore) {
				protoMetadata := &pb.ContractMetadata{
					Domain:        "test-domain",
					Environment:   "catalog_testing",
					ChainSelector: 12345,
					Address:       "0x1234567890abcdef1234567890abcdef12345678",
					Metadata:      `{"name":"TestContract","version":"1.0.0"}`,
					RowVersion:    1,
				}

				metadata, err := store.protoToContractMetadata(protoMetadata)

				assert.NoError(t, err)
				assert.Equal(t, "0x1234567890abcdef1234567890abcdef12345678", metadata.Address)
				assert.Equal(t, uint64(12345), metadata.ChainSelector)
				assert.NotNil(t, metadata.Metadata)

				// Check JSON unmarshaling - it will be unmarshaled as map[string]interface{}
				// since that's what json.Unmarshal defaults to for interface{}
				metadataMap := metadata.Metadata.(map[string]interface{})
				assert.Equal(t, "TestContract", metadataMap["name"])
				assert.Equal(t, "1.0.0", metadataMap["version"])
			},
		},
		{
			name: "protoToContractMetadata_invalid_json",
			test: func(t *testing.T, store *CatalogContractMetadataStore) {
				protoMetadata := &pb.ContractMetadata{
					Domain:        "test-domain",
					Environment:   "catalog_testing",
					ChainSelector: 12345,
					Address:       "0x1234567890abcdef1234567890abcdef12345678",
					Metadata:      `{invalid-json`,
					RowVersion:    1,
				}

				_, err := store.protoToContractMetadata(protoMetadata)

				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to unmarshal metadata JSON")
			},
		},
		{
			name: "contractMetadataToProto",
			test: func(t *testing.T, store *CatalogContractMetadataStore) {
				metadata := newRandomContractMetadata()

				protoMetadata := store.contractMetadataToProto(metadata)

				assert.Equal(t, "test-domain", protoMetadata.Domain)
				assert.Equal(t, "catalog_testing", protoMetadata.Environment)
				assert.Equal(t, metadata.ChainSelector, protoMetadata.ChainSelector)
				assert.Equal(t, metadata.Address, protoMetadata.Address)
				assert.NotEmpty(t, protoMetadata.Metadata)
				assert.Equal(t, int32(1), protoMetadata.RowVersion) // Should be 0 initially

				// Verify JSON marshaling worked
				assert.Contains(t, protoMetadata.Metadata, "name")
				assert.Contains(t, protoMetadata.Metadata, "version")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh store for each test case to avoid concurrency issues
			store, conn := setupTestContractStore(t)
			skipIfNoContractService(t, conn)
			defer conn.Close()

			tt.test(t, store)
		})
	}
}
