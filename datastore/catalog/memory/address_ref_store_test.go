package memory

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

const (
	// Default gRPC server address - can be overridden with CATALOG_GRPC_ADDRESS env var
	defaultGRPCAddress = "localhost:8080"
)

func TestCatalogAddressRefStore_Get(t *testing.T) {
	t.Parallel()
	store, closer := setupTestStore(t)

	t.Run("not found", func(t *testing.T) {
		version := semver.MustParse("99.99.99")
		key := datastore.NewAddressRefKey(99999999, "NonExistentContract", version, "nonexistent")
		_, err := store.Get(t.Context(), key)
		require.Error(t, err)
		require.ErrorIs(t, err, datastore.ErrAddressRefNotFound)
	})

	t.Run("success", func(t *testing.T) {
		addressRef := newRandomAddressRef()
		err := store.Add(t.Context(), addressRef)
		require.NoError(t, err)
		key := datastore.NewAddressRefKey(addressRef.ChainSelector, addressRef.Type, addressRef.Version, addressRef.Qualifier)
		result, err := store.Get(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, key.ChainSelector(), result.ChainSelector)
		require.Equal(t, key.Type(), result.Type)
		require.Equal(t, key.Version().String(), result.Version.String())
		require.Equal(t, key.Qualifier(), result.Qualifier)
		require.Equal(t, addressRef.Address, result.Address)
		require.Equal(t, addressRef.Labels, result.Labels)
	})
	defer closer()
}

func TestCatalogAddressRefStore_Add(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(store *memoryAddressRefStore) datastore.AddressRef
		expectError bool
		errorCheck  func(error) bool
	}{
		{
			name: "success",
			setup: func(store *memoryAddressRefStore) datastore.AddressRef {
				return newRandomAddressRef()
			},
			expectError: false,
		},
		{
			name: "duplicate_error",
			setup: func(store *memoryAddressRefStore) datastore.AddressRef {
				// Create and add a record first
				ref := newRandomAddressRef()
				err := store.Add(t.Context(), ref)
				require.NoError(t, err)
				// Return the same record to test duplicate
				return ref
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh store for each test case to avoid concurrency issues
			store, closer := setupTestStore(t)
			defer closer()

			addressRef := tt.setup(store)

			// Execute
			err := store.Add(t.Context(), addressRef)

			// Verify
			if tt.expectError {
				require.Error(t, err)
				if tt.errorCheck != nil {
					require.True(t, tt.errorCheck(err))
				}
			} else {
				require.NoError(t, err)

				// Verify we can get it back
				key := datastore.NewAddressRefKey(addressRef.ChainSelector, addressRef.Type, addressRef.Version, addressRef.Qualifier)
				retrieved, err := store.Get(t.Context(), key)
				require.NoError(t, err)

				require.Equal(t, addressRef.Address, retrieved.Address)
				require.Equal(t, addressRef.ChainSelector, retrieved.ChainSelector)
				require.Equal(t, addressRef.Type, retrieved.Type)
				require.Equal(t, addressRef.Version.String(), retrieved.Version.String())
				require.Equal(t, addressRef.Qualifier, retrieved.Qualifier)
				require.Equal(t, addressRef.Labels.List(), retrieved.Labels.List())
			}
		})
	}
}

func TestCatalogAddressRefStore_Update(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(store *memoryAddressRefStore) datastore.AddressRef
		expectError bool
		errorType   error
		verify      func(t *testing.T, store *memoryAddressRefStore, addressRef datastore.AddressRef)
	}{
		{
			name: "success",
			setup: func(store *memoryAddressRefStore) datastore.AddressRef {
				// Create and add an address ref
				addressRef := newRandomAddressRef()
				err := store.Add(t.Context(), addressRef)
				require.NoError(t, err)

				// Modify the address ref with new unique values
				addressRef.Address = "0x" + randomHex(40)
				addressRef.Labels = datastore.NewLabelSet("updated", "test")

				return addressRef
			},
			expectError: false,
			verify: func(t *testing.T, store *memoryAddressRefStore, addressRef datastore.AddressRef) {
				t.Helper()
				// Verify the updated values
				key := datastore.NewAddressRefKey(addressRef.ChainSelector, addressRef.Type, addressRef.Version, addressRef.Qualifier)
				retrieved, err := store.Get(t.Context(), key)
				require.NoError(t, err)
				require.Equal(t, addressRef.Address, retrieved.Address)
				require.Equal(t, addressRef.Labels.List(), retrieved.Labels.List())
			},
		},
		{
			name: "not_found",
			setup: func(store *memoryAddressRefStore) datastore.AddressRef {
				// Try to update a record that doesn't exist
				return newRandomAddressRef()
			},
			expectError: true,
			errorType:   datastore.ErrAddressRefNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh store for each test case to avoid concurrency issues
			store, closer := setupTestStore(t)
			defer closer()

			addressRef := tt.setup(store)

			// Execute update
			err := store.Update(t.Context(), addressRef)

			// Verify
			if tt.expectError {
				require.Error(t, err)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
				if tt.verify != nil {
					tt.verify(t, store, addressRef)
				}
			}
		})
	}
}

func TestCatalogAddressRefStore_Upsert(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		setup  func(store *memoryAddressRefStore) datastore.AddressRef
		verify func(t *testing.T, store *memoryAddressRefStore, original datastore.AddressRef)
	}{
		{
			name: "insert_new_record",
			setup: func(store *memoryAddressRefStore) datastore.AddressRef {
				// Create a unique address ref for this test
				return newRandomAddressRef()
			},
			verify: func(t *testing.T, store *memoryAddressRefStore, original datastore.AddressRef) {
				t.Helper()
				// Verify we can get it back
				key := datastore.NewAddressRefKey(original.ChainSelector, original.Type, original.Version, original.Qualifier)
				retrieved, err := store.Get(t.Context(), key)
				require.NoError(t, err)
				require.Equal(t, original.Address, retrieved.Address)
			},
		},
		{
			name: "update_existing_record",
			setup: func(store *memoryAddressRefStore) datastore.AddressRef {
				// Create and add an address ref
				addressRef := newRandomAddressRef()
				err := store.Add(t.Context(), addressRef)
				require.NoError(t, err)

				// Modify the address ref with new unique values
				addressRef.Address = "0x" + randomHex(40)
				addressRef.Labels = datastore.NewLabelSet("modified", "test")

				return addressRef
			},
			verify: func(t *testing.T, store *memoryAddressRefStore, modified datastore.AddressRef) {
				t.Helper()
				// Verify the updated values
				key := datastore.NewAddressRefKey(modified.ChainSelector, modified.Type, modified.Version, modified.Qualifier)
				retrieved, err := store.Get(t.Context(), key)
				require.NoError(t, err)
				require.Equal(t, modified.Address, retrieved.Address)
				require.Equal(t, modified.Labels.List(), retrieved.Labels.List())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh store for each test case to avoid concurrency issues
			store, closer := setupTestStore(t)
			defer closer()

			addressRef := tt.setup(store)

			// Execute upsert
			err := store.Upsert(t.Context(), addressRef)

			// Verify
			require.NoError(t, err)
			tt.verify(t, store, addressRef)
		})
	}
}

func TestCatalogAddressRefStore_Delete(t *testing.T) {
	t.Parallel()
	store, closer := setupTestStore(t)
	defer closer()

	version := semver.MustParse("1.0.0")
	key := datastore.NewAddressRefKey(12345, "LinkToken", version, "test")

	// Execute
	err := store.Delete(t.Context(), key)

	// Verify
	require.Error(t, err)
	require.Contains(t, err.Error(), "delete operation not supported")
}

func TestCatalogAddressRefStore_FetchAndFilter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		operation    string
		setup        func(store *memoryAddressRefStore) (datastore.AddressRef, datastore.AddressRef)
		createFilter func(addressRef1, addressRef2 datastore.AddressRef) datastore.FilterFunc[datastore.AddressRefKey, datastore.AddressRef]
		minExpected  int
		verify       func(t *testing.T, results []datastore.AddressRef, addressRef1, addressRef2 datastore.AddressRef)
	}{
		{
			name:      "fetch_all",
			operation: "fetch",
			setup: func(store *memoryAddressRefStore) (datastore.AddressRef, datastore.AddressRef) {
				// Setup test data with unique chain selectors
				addressRef1 := newRandomAddressRef()
				chainSelector1 := randomChainSelector()
				addressRef1.ChainSelector = chainSelector1
				err := store.Add(t.Context(), addressRef1)
				require.NoError(t, err)

				addressRef2 := newRandomAddressRef()
				chainSelector2 := randomChainSelector()
				// Ensure different chain selectors
				for chainSelector2 == chainSelector1 {
					chainSelector2 = randomChainSelector()
				}
				addressRef2.ChainSelector = chainSelector2
				err = store.Add(t.Context(), addressRef2)
				require.NoError(t, err)

				return addressRef1, addressRef2
			},
			createFilter: nil,
			minExpected:  2,
			verify: func(t *testing.T, results []datastore.AddressRef, addressRef1, addressRef2 datastore.AddressRef) {
				t.Helper()
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
				require.True(t, foundFirst, "First address ref not found in fetch results")
				require.True(t, foundSecond, "Second address ref not found in fetch results")
			},
		},
		{
			name:      "filter_by_chain_selector",
			operation: "filter",
			setup: func(store *memoryAddressRefStore) (datastore.AddressRef, datastore.AddressRef) {
				// Setup test data with unique chain selectors
				addressRef1 := newRandomAddressRef()
				chainSelector1 := randomChainSelector()
				addressRef1.ChainSelector = chainSelector1
				err := store.Add(t.Context(), addressRef1)
				require.NoError(t, err)

				addressRef2 := newRandomAddressRef()
				chainSelector2 := randomChainSelector()
				// Ensure different chain selectors
				for chainSelector2 == chainSelector1 {
					chainSelector2 = randomChainSelector()
				}
				addressRef2.ChainSelector = chainSelector2
				err = store.Add(t.Context(), addressRef2)
				require.NoError(t, err)

				return addressRef1, addressRef2
			},
			createFilter: func(addressRef1, addressRef2 datastore.AddressRef) datastore.FilterFunc[datastore.AddressRefKey, datastore.AddressRef] {
				// Use the proper filter from datastore/filters.go
				return datastore.AddressRefByChainSelector(addressRef1.ChainSelector)
			},
			minExpected: 1,
			verify: func(t *testing.T, results []datastore.AddressRef, addressRef1, addressRef2 datastore.AddressRef) {
				t.Helper()
				// All results should have the chain selector from addressRef1
				for _, result := range results {
					require.Equal(t, addressRef1.ChainSelector, result.ChainSelector)
				}
			},
		},
		{
			name:      "filter_by_address",
			operation: "filter",
			setup: func(store *memoryAddressRefStore) (datastore.AddressRef, datastore.AddressRef) {
				// Setup test data with unique addresses
				addressRef1 := newRandomAddressRef()
				err := store.Add(t.Context(), addressRef1)
				require.NoError(t, err)

				addressRef2 := newRandomAddressRef()
				err = store.Add(t.Context(), addressRef2)
				require.NoError(t, err)

				return addressRef1, addressRef2
			},
			createFilter: func(addressRef1, addressRef2 datastore.AddressRef) datastore.FilterFunc[datastore.AddressRefKey, datastore.AddressRef] {
				// Use AddressRefByAddress filter
				return datastore.AddressRefByAddress(addressRef1.Address)
			},
			minExpected: 1,
			verify: func(t *testing.T, results []datastore.AddressRef, addressRef1, addressRef2 datastore.AddressRef) {
				t.Helper()
				// All results should have the address from addressRef1
				for _, result := range results {
					require.Equal(t, addressRef1.Address, result.Address)
				}
			},
		},
		{
			name:      "filter_by_contract_type",
			operation: "filter",
			setup: func(store *memoryAddressRefStore) (datastore.AddressRef, datastore.AddressRef) {
				// Setup test data with different contract types
				addressRef1 := newRandomAddressRef()
				addressRef1.Type = "UniqueContract1"
				err := store.Add(t.Context(), addressRef1)
				require.NoError(t, err)

				addressRef2 := newRandomAddressRef()
				addressRef2.Type = "UniqueContract2"
				err = store.Add(t.Context(), addressRef2)
				require.NoError(t, err)

				return addressRef1, addressRef2
			},
			createFilter: func(addressRef1, addressRef2 datastore.AddressRef) datastore.FilterFunc[datastore.AddressRefKey, datastore.AddressRef] {
				// Use AddressRefByType filter
				return datastore.AddressRefByType(addressRef1.Type)
			},
			minExpected: 1,
			verify: func(t *testing.T, results []datastore.AddressRef, addressRef1, addressRef2 datastore.AddressRef) {
				t.Helper()
				// All results should have the contract type from addressRef1
				for _, result := range results {
					require.Equal(t, addressRef1.Type, result.Type)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh store for each test case to avoid concurrency issues
			store, closer := setupTestStore(t)
			defer closer()

			addressRef1, addressRef2 := tt.setup(store)

			var results []datastore.AddressRef
			var err error

			// Execute operation
			switch tt.operation {
			case "fetch":
				results, err = store.Fetch(t.Context())
			case "filter":
				var filterFunc datastore.FilterFunc[datastore.AddressRefKey, datastore.AddressRef]
				if tt.createFilter != nil {
					filterFunc = tt.createFilter(addressRef1, addressRef2)
				}
				results, err = store.Filter(t.Context(), filterFunc)
			}

			// Verify
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(results), tt.minExpected)
			if tt.verify != nil {
				tt.verify(t, results, addressRef1, addressRef2)
			}
		})
	}
}

// setupTestStore creates a real gRPC client connection to a local service
func setupTestStore(t *testing.T) (*memoryAddressRefStore, func()) {
	t.Helper()
	config := MemoryDataStoreConfig{
		Domain:      "test_domain",
		Environment: "catalog_testing",
	}
	store := NewMemoryDataStore(t, config)
	return store.Addresses().(*memoryAddressRefStore), func() {
		store.Close()
	}
}

// randomHex generates a random hex string of specified length
func randomHex(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("failed to generate random bytes: %v", err))
	}

	return hex.EncodeToString(bytes)
}

// randomChainSelector generates a random chain selector
func randomChainSelector() uint64 {
	maxVal := big.NewInt(999999999) // Large but reasonable upper bound
	n, err := rand.Int(rand.Reader, maxVal)
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
