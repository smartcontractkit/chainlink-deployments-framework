package catalog

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/Masterminds/semver/v3"
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
	defaultGRPCAddress = "localhost:8080"
)

func TestCatalogAddressRefStore_Get(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(store *CatalogAddressRefStore) datastore.AddressRefKey
		expectError bool
		errorType   error
	}{
		{
			name: "not_found",
			setup: func(store *CatalogAddressRefStore) datastore.AddressRefKey {
				// Use a unique key that shouldn't exist
				version := semver.MustParse("99.99.99")
				return datastore.NewAddressRefKey(99999999, "NonExistentContract", version, "nonexistent")
			},
			expectError: true,
			errorType:   datastore.ErrAddressRefNotFound,
		},
		{
			name: "success",
			setup: func(store *CatalogAddressRefStore) datastore.AddressRefKey {
				// Create and add a record first
				addressRef := newRandomAddressRef()
				err := store.Add(addressRef)
				require.NoError(t, err)
				return datastore.NewAddressRefKey(addressRef.ChainSelector, addressRef.Type, addressRef.Version, addressRef.Qualifier)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh store for each test case to avoid concurrency issues
			store, conn := setupTestStore(t)
			skipIfNoService(t, conn)
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
				assert.Equal(t, key.Type(), result.Type)
				assert.Equal(t, key.Version().String(), result.Version.String())
				assert.Equal(t, key.Qualifier(), result.Qualifier)
			}
		})
	}
}

func TestCatalogAddressRefStore_Add(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(store *CatalogAddressRefStore) datastore.AddressRef
		expectError bool
		errorCheck  func(error) bool
	}{
		{
			name: "success",
			setup: func(store *CatalogAddressRefStore) datastore.AddressRef {
				return newRandomAddressRef()
			},
			expectError: false,
		},
		{
			name: "duplicate_error",
			setup: func(store *CatalogAddressRefStore) datastore.AddressRef {
				// Create and add a record first
				ref := newRandomAddressRef()
				err := store.Add(ref)
				require.NoError(t, err)
				// Return the same record to test duplicate
				return ref
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh store for each test case to avoid concurrency issues
			store, conn := setupTestStore(t)
			skipIfNoService(t, conn)
			defer conn.Close()

			addressRef := tt.setup(store)

			// Execute
			err := store.Add(addressRef)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorCheck != nil {
					assert.True(t, tt.errorCheck(err))
				}
			} else {
				assert.NoError(t, err)

				// Verify we can get it back
				key := datastore.NewAddressRefKey(addressRef.ChainSelector, addressRef.Type, addressRef.Version, addressRef.Qualifier)
				retrieved, err := store.Get(key)
				require.NoError(t, err)

				assert.Equal(t, addressRef.Address, retrieved.Address)
				assert.Equal(t, addressRef.ChainSelector, retrieved.ChainSelector)
				assert.Equal(t, addressRef.Type, retrieved.Type)
				assert.Equal(t, addressRef.Version.String(), retrieved.Version.String())
				assert.Equal(t, addressRef.Qualifier, retrieved.Qualifier)
				assert.Equal(t, addressRef.Labels.List(), retrieved.Labels.List())
			}
		})
	}
}

func TestCatalogAddressRefStore_Update(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(store *CatalogAddressRefStore) datastore.AddressRef
		expectError bool
		errorType   error
		verify      func(t *testing.T, store *CatalogAddressRefStore, addressRef datastore.AddressRef)
	}{
		{
			name: "success",
			setup: func(store *CatalogAddressRefStore) datastore.AddressRef {
				// Create and add an address ref
				addressRef := newRandomAddressRef()
				err := store.Add(addressRef)
				require.NoError(t, err)

				// Modify the address ref with new unique values
				addressRef.Address = "0x" + randomHex(40)
				addressRef.Labels = datastore.NewLabelSet("updated", "test")
				return addressRef
			},
			expectError: false,
			verify: func(t *testing.T, store *CatalogAddressRefStore, addressRef datastore.AddressRef) {
				// Verify the updated values
				key := datastore.NewAddressRefKey(addressRef.ChainSelector, addressRef.Type, addressRef.Version, addressRef.Qualifier)
				retrieved, err := store.Get(key)
				require.NoError(t, err)
				assert.Equal(t, addressRef.Address, retrieved.Address)
				assert.Equal(t, addressRef.Labels.List(), retrieved.Labels.List())
			},
		},
		{
			name: "not_found",
			setup: func(store *CatalogAddressRefStore) datastore.AddressRef {
				// Try to update a record that doesn't exist
				return newRandomAddressRef()
			},
			expectError: true,
			errorType:   datastore.ErrAddressRefNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh store for each test case to avoid concurrency issues
			store, conn := setupTestStore(t)
			skipIfNoService(t, conn)
			defer conn.Close()

			addressRef := tt.setup(store)

			// Execute update
			err := store.Update(addressRef)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
				if tt.verify != nil {
					tt.verify(t, store, addressRef)
				}
			}
		})
	}
}

func TestCatalogAddressRefStore_Upsert(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(store *CatalogAddressRefStore) datastore.AddressRef
		verify func(t *testing.T, store *CatalogAddressRefStore, original datastore.AddressRef)
	}{
		{
			name: "insert_new_record",
			setup: func(store *CatalogAddressRefStore) datastore.AddressRef {
				// Create a unique address ref for this test
				return newRandomAddressRef()
			},
			verify: func(t *testing.T, store *CatalogAddressRefStore, original datastore.AddressRef) {
				// Verify we can get it back
				key := datastore.NewAddressRefKey(original.ChainSelector, original.Type, original.Version, original.Qualifier)
				retrieved, err := store.Get(key)
				require.NoError(t, err)
				assert.Equal(t, original.Address, retrieved.Address)
			},
		},
		{
			name: "update_existing_record",
			setup: func(store *CatalogAddressRefStore) datastore.AddressRef {
				// Create and add an address ref
				addressRef := newRandomAddressRef()
				err := store.Add(addressRef)
				require.NoError(t, err)

				// Modify the address ref with new unique values
				addressRef.Address = "0x" + randomHex(40)
				addressRef.Labels = datastore.NewLabelSet("modified", "test")
				return addressRef
			},
			verify: func(t *testing.T, store *CatalogAddressRefStore, modified datastore.AddressRef) {
				// Verify the updated values
				key := datastore.NewAddressRefKey(modified.ChainSelector, modified.Type, modified.Version, modified.Qualifier)
				retrieved, err := store.Get(key)
				require.NoError(t, err)
				assert.Equal(t, modified.Address, retrieved.Address)
				assert.Equal(t, modified.Labels.List(), retrieved.Labels.List())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh store for each test case to avoid concurrency issues
			store, conn := setupTestStore(t)
			skipIfNoService(t, conn)
			defer conn.Close()

			addressRef := tt.setup(store)

			// Execute upsert
			err := store.Upsert(addressRef)

			// Verify
			assert.NoError(t, err)
			tt.verify(t, store, addressRef)
		})
	}
}

func TestCatalogAddressRefStore_Delete(t *testing.T) {
	store, conn := setupTestStore(t)
	skipIfNoService(t, conn)
	defer conn.Close()

	version := semver.MustParse("1.0.0")
	key := datastore.NewAddressRefKey(12345, "LinkToken", version, "test")

	// Execute
	err := store.Delete(key)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delete operation not supported")
}

func TestCatalogAddressRefStore_FetchAndFilter(t *testing.T) {
	tests := []struct {
		name         string
		operation    string
		setup        func(store *CatalogAddressRefStore) (datastore.AddressRef, datastore.AddressRef)
		createFilter func(addressRef1, addressRef2 datastore.AddressRef) datastore.FilterFunc[datastore.AddressRefKey, datastore.AddressRef]
		minExpected  int
		verify       func(t *testing.T, results []datastore.AddressRef, addressRef1, addressRef2 datastore.AddressRef)
	}{
		{
			name:      "fetch_all",
			operation: "fetch",
			setup: func(store *CatalogAddressRefStore) (datastore.AddressRef, datastore.AddressRef) {
				// Setup test data with unique chain selectors
				addressRef1 := newRandomAddressRef()
				chainSelector1 := randomChainSelector()
				addressRef1.ChainSelector = chainSelector1
				err := store.Add(addressRef1)
				require.NoError(t, err)

				addressRef2 := newRandomAddressRef()
				chainSelector2 := randomChainSelector()
				// Ensure different chain selectors
				for chainSelector2 == chainSelector1 {
					chainSelector2 = randomChainSelector()
				}
				addressRef2.ChainSelector = chainSelector2
				err = store.Add(addressRef2)
				require.NoError(t, err)

				return addressRef1, addressRef2
			},
			createFilter: nil,
			minExpected:  2,
			verify: func(t *testing.T, results []datastore.AddressRef, addressRef1, addressRef2 datastore.AddressRef) {
				// Check that our records are in the results
				foundFirst := false
				foundSecond := false
				for _, result := range results {
					if result.Address == addressRef1.Address && result.ChainSelector == addressRef1.ChainSelector {
						foundFirst = true
					}
					if result.Address == addressRef2.Address && result.ChainSelector == addressRef2.ChainSelector {
						foundSecond = true
					}
				}
				assert.True(t, foundFirst, "First address ref not found in fetch results")
				assert.True(t, foundSecond, "Second address ref not found in fetch results")
			},
		},
		{
			name:      "filter_by_chain_selector",
			operation: "filter",
			setup: func(store *CatalogAddressRefStore) (datastore.AddressRef, datastore.AddressRef) {
				// Setup test data with unique chain selectors
				addressRef1 := newRandomAddressRef()
				chainSelector1 := randomChainSelector()
				addressRef1.ChainSelector = chainSelector1
				err := store.Add(addressRef1)
				require.NoError(t, err)

				addressRef2 := newRandomAddressRef()
				chainSelector2 := randomChainSelector()
				// Ensure different chain selectors
				for chainSelector2 == chainSelector1 {
					chainSelector2 = randomChainSelector()
				}
				addressRef2.ChainSelector = chainSelector2
				err = store.Add(addressRef2)
				require.NoError(t, err)

				return addressRef1, addressRef2
			},
			createFilter: func(addressRef1, addressRef2 datastore.AddressRef) datastore.FilterFunc[datastore.AddressRefKey, datastore.AddressRef] {
				// Use the proper filter from datastore/filters.go
				return datastore.AddressRefByChainSelector(addressRef1.ChainSelector)
			},
			minExpected: 1,
			verify: func(t *testing.T, results []datastore.AddressRef, addressRef1, addressRef2 datastore.AddressRef) {
				// All results should have the chain selector from addressRef1
				for _, result := range results {
					assert.Equal(t, addressRef1.ChainSelector, result.ChainSelector)
				}
			},
		},
		{
			name:      "filter_by_address",
			operation: "filter",
			setup: func(store *CatalogAddressRefStore) (datastore.AddressRef, datastore.AddressRef) {
				// Setup test data with unique addresses
				addressRef1 := newRandomAddressRef()
				err := store.Add(addressRef1)
				require.NoError(t, err)

				addressRef2 := newRandomAddressRef()
				err = store.Add(addressRef2)
				require.NoError(t, err)

				return addressRef1, addressRef2
			},
			createFilter: func(addressRef1, addressRef2 datastore.AddressRef) datastore.FilterFunc[datastore.AddressRefKey, datastore.AddressRef] {
				// Use AddressRefByAddress filter
				return datastore.AddressRefByAddress(addressRef1.Address)
			},
			minExpected: 1,
			verify: func(t *testing.T, results []datastore.AddressRef, addressRef1, addressRef2 datastore.AddressRef) {
				// All results should have the address from addressRef1
				for _, result := range results {
					assert.Equal(t, addressRef1.Address, result.Address)
				}
			},
		},
		{
			name:      "filter_by_contract_type",
			operation: "filter",
			setup: func(store *CatalogAddressRefStore) (datastore.AddressRef, datastore.AddressRef) {
				// Setup test data with different contract types
				addressRef1 := newRandomAddressRef()
				addressRef1.Type = "UniqueContract1"
				err := store.Add(addressRef1)
				require.NoError(t, err)

				addressRef2 := newRandomAddressRef()
				addressRef2.Type = "UniqueContract2"
				err = store.Add(addressRef2)
				require.NoError(t, err)

				return addressRef1, addressRef2
			},
			createFilter: func(addressRef1, addressRef2 datastore.AddressRef) datastore.FilterFunc[datastore.AddressRefKey, datastore.AddressRef] {
				// Use AddressRefByType filter
				return datastore.AddressRefByType(addressRef1.Type)
			},
			minExpected: 1,
			verify: func(t *testing.T, results []datastore.AddressRef, addressRef1, addressRef2 datastore.AddressRef) {
				// All results should have the contract type from addressRef1
				for _, result := range results {
					assert.Equal(t, addressRef1.Type, result.Type)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh store for each test case to avoid concurrency issues
			store, conn := setupTestStore(t)
			skipIfNoService(t, conn)
			defer conn.Close()

			addressRef1, addressRef2 := tt.setup(store)

			var results []datastore.AddressRef
			var err error

			// Execute operation
			switch tt.operation {
			case "fetch":
				results, err = store.Fetch()
			case "filter":
				var filterFunc datastore.FilterFunc[datastore.AddressRefKey, datastore.AddressRef]
				if tt.createFilter != nil {
					filterFunc = tt.createFilter(addressRef1, addressRef2)
				}
				results = store.Filter(filterFunc)
			}

			// Verify
			if tt.operation == "fetch" {
				assert.NoError(t, err)
			}
			assert.GreaterOrEqual(t, len(results), tt.minExpected)
			if tt.verify != nil {
				tt.verify(t, results, addressRef1, addressRef2)
			}
		})
	}
}

func TestCatalogAddressRefStore_ConversionHelpers(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T, store *CatalogAddressRefStore)
	}{
		{
			name: "keyToFilter",
			test: func(t *testing.T, store *CatalogAddressRefStore) {
				version := semver.MustParse("1.2.3")
				key := datastore.NewAddressRefKey(12345, "LinkToken", version, "test")

				filter := store.keyToFilter(key)

				assert.Equal(t, "test-domain", filter.Domain.Value)
				assert.Equal(t, "catalog_testing", filter.Environment.Value)
				assert.Equal(t, uint64(12345), filter.ChainSelector.Value)
				assert.Equal(t, "LinkToken", filter.ContractType.Value)
				assert.Equal(t, "1.2.3", filter.Version.Value)
				assert.Equal(t, "test", filter.Qualifier.Value)
			},
		},
		{
			name: "protoToAddressRef_success",
			test: func(t *testing.T, store *CatalogAddressRefStore) {
				protoRef := &pb.AddressReference{
					Domain:        "test-domain",
					Environment:   "catalog_testing",
					ChainSelector: 12345,
					ContractType:  "LinkToken",
					Version:       "1.0.0",
					Qualifier:     "test",
					Address:       "0x1234567890abcdef",
					LabelSet:      []string{"label1", "label2"},
				}

				addressRef, err := store.protoToAddressRef(protoRef)

				assert.NoError(t, err)
				assert.Equal(t, "0x1234567890abcdef", addressRef.Address)
				assert.Equal(t, uint64(12345), addressRef.ChainSelector)
				assert.Equal(t, datastore.ContractType("LinkToken"), addressRef.Type)
				assert.Equal(t, "1.0.0", addressRef.Version.String())
				assert.Equal(t, "test", addressRef.Qualifier)
				assert.Equal(t, []string{"label1", "label2"}, addressRef.Labels.List())
			},
		},
		{
			name: "protoToAddressRef_invalid_version",
			test: func(t *testing.T, store *CatalogAddressRefStore) {
				protoRef := &pb.AddressReference{
					Domain:        "test-domain",
					Environment:   "catalog_testing",
					ChainSelector: 12345,
					ContractType:  "LinkToken",
					Version:       "invalid-version",
					Qualifier:     "test",
					Address:       "0x1234567890abcdef",
					LabelSet:      []string{"label1", "label2"},
				}

				_, err := store.protoToAddressRef(protoRef)

				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to parse version")
			},
		},
		{
			name: "addressRefToProto",
			test: func(t *testing.T, store *CatalogAddressRefStore) {
				addressRef := newRandomAddressRef()

				protoRef := store.addressRefToProto(addressRef)

				assert.Equal(t, "test-domain", protoRef.Domain)
				assert.Equal(t, "catalog_testing", protoRef.Environment)
				assert.Equal(t, addressRef.ChainSelector, protoRef.ChainSelector)
				assert.Equal(t, string(addressRef.Type), protoRef.ContractType)
				assert.Equal(t, addressRef.Version.String(), protoRef.Version)
				assert.Equal(t, addressRef.Qualifier, protoRef.Qualifier)
				assert.Equal(t, addressRef.Address, protoRef.Address)
				assert.Equal(t, addressRef.Labels.List(), protoRef.LabelSet)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh store for each test case to avoid concurrency issues
			store, conn := setupTestStore(t)
			skipIfNoService(t, conn)
			defer conn.Close()

			tt.test(t, store)
		})
	}
}

// setupTestStore creates a real gRPC client connection to a local service
func setupTestStore(t *testing.T) (*CatalogAddressRefStore, *grpc.ClientConn) {
	// Get gRPC address from environment or use default
	address := os.Getenv("CATALOG_GRPC_ADDRESS")
	if address == "" {
		address = defaultGRPCAddress
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
	store := NewCatalogAddressRefStore(CatalogAddressRefStoreConfig{
		Domain:      "test-domain",
		Environment: "catalog_testing",
		Client:      client,
	})

	return store, conn
}

// skipIfNoService skips the test if we can't connect to the gRPC service
func skipIfNoService(t *testing.T, conn *grpc.ClientConn) {
	if conn == nil {
		t.Skip("Skipping test: gRPC service not available")
	}
}

// randomHex generates a random hex string of specified length
func randomHex(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("failed to generate random bytes: %v", err))
	}
	return fmt.Sprintf("%x", bytes)
}

// randomChainSelector generates a random chain selector
func randomChainSelector() uint64 {
	max := big.NewInt(999999999) // Large but reasonable upper bound
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic(fmt.Sprintf("failed to generate random chain selector: %v", err))
	}
	return n.Uint64() + 1 // Ensure it's not zero
}

func newRandomAddressRef() datastore.AddressRef {
	version := semver.MustParse("1.0.0")
	id := uuid.New().String()[:8] // Use first 8 chars of UUID for uniqueness

	return datastore.AddressRef{
		Address:       "0x" + randomHex(40), // 40 hex chars = 20 bytes (standard address length)
		ChainSelector: randomChainSelector(),
		Type:          "TestContract",
		Version:       version,
		Qualifier:     "test-" + id, // Use UUID-based unique qualifier
		Labels:        datastore.NewLabelSet("test", "integration"),
	}
}
