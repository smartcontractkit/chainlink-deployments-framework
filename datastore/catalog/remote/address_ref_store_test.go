package remote

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	pb "github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/remote/internal/protos"
)

const (
	// Default gRPC server address - can be overridden with CATALOG_GRPC_ADDRESS env var
	defaultGRPCAddress = "localhost:8080"
)

func TestCatalogAddressRefStore_Get(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(store *catalogAddressRefStore) datastore.AddressRefKey
		expectError bool
		errorType   error
	}{
		{
			name: "not_found",
			setup: func(store *catalogAddressRefStore) datastore.AddressRefKey {
				// Use a unique key that shouldn't exist
				version := semver.MustParse("99.99.99")
				return datastore.NewAddressRefKey(99999999, "NonExistentContract", version, "nonexistent")
			},
			expectError: true,
			errorType:   datastore.ErrAddressRefNotFound,
		},
		{
			name: "success",
			setup: func(store *catalogAddressRefStore) datastore.AddressRefKey {
				// Create and add a record first
				addressRef := newRandomAddressRef()
				err := store.Add(t.Context(), addressRef)
				require.NoError(t, err)

				return datastore.NewAddressRefKey(addressRef.ChainSelector, addressRef.Type, addressRef.Version, addressRef.Qualifier)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh store for each test case to avoid concurrency issues
			store := setupTestStore(t)

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
				require.Equal(t, key.Type(), result.Type)
				require.Equal(t, key.Version().String(), result.Version.String())
				require.Equal(t, key.Qualifier(), result.Qualifier)
			}
		})
	}
}

func TestCatalogAddressRefStore_Add(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(store *catalogAddressRefStore) datastore.AddressRef
		expectError bool
		errorCheck  func(error) bool
	}{
		{
			name: "success",
			setup: func(store *catalogAddressRefStore) datastore.AddressRef {
				return newRandomAddressRef()
			},
			expectError: false,
		},
		{
			name: "duplicate_error",
			setup: func(store *catalogAddressRefStore) datastore.AddressRef {
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
			store := setupTestStore(t)

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
		setup       func(store *catalogAddressRefStore) datastore.AddressRef
		expectError bool
		errorType   error
		verify      func(t *testing.T, store *catalogAddressRefStore, addressRef datastore.AddressRef)
	}{
		{
			name: "success",
			setup: func(store *catalogAddressRefStore) datastore.AddressRef {
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
			verify: func(t *testing.T, store *catalogAddressRefStore, addressRef datastore.AddressRef) {
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
			setup: func(store *catalogAddressRefStore) datastore.AddressRef {
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
			store := setupTestStore(t)

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
		setup  func(store *catalogAddressRefStore) datastore.AddressRef
		verify func(t *testing.T, store *catalogAddressRefStore, original datastore.AddressRef)
	}{
		{
			name: "insert_new_record",
			setup: func(store *catalogAddressRefStore) datastore.AddressRef {
				// Create a unique address ref for this test
				return newRandomAddressRef()
			},
			verify: func(t *testing.T, store *catalogAddressRefStore, original datastore.AddressRef) {
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
			setup: func(store *catalogAddressRefStore) datastore.AddressRef {
				// Create and add an address ref
				addressRef := newRandomAddressRef()
				err := store.Add(t.Context(), addressRef)
				require.NoError(t, err)

				// Modify the address ref with new unique values
				addressRef.Address = "0x" + randomHex(40)
				addressRef.Labels = datastore.NewLabelSet("modified", "test")

				return addressRef
			},
			verify: func(t *testing.T, store *catalogAddressRefStore, modified datastore.AddressRef) {
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
			store := setupTestStore(t)

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
	store := setupTestStore(t)

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
		setup        func(store *catalogAddressRefStore) (datastore.AddressRef, datastore.AddressRef)
		createFilter func(addressRef1, addressRef2 datastore.AddressRef) datastore.FilterFunc[datastore.AddressRefKey, datastore.AddressRef]
		minExpected  int
		verify       func(t *testing.T, results []datastore.AddressRef, addressRef1, addressRef2 datastore.AddressRef)
	}{
		{
			name:      "fetch_all",
			operation: "fetch",
			setup: func(store *catalogAddressRefStore) (datastore.AddressRef, datastore.AddressRef) {
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
			setup: func(store *catalogAddressRefStore) (datastore.AddressRef, datastore.AddressRef) {
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
			setup: func(store *catalogAddressRefStore) (datastore.AddressRef, datastore.AddressRef) {
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
			setup: func(store *catalogAddressRefStore) (datastore.AddressRef, datastore.AddressRef) {
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
			store := setupTestStore(t)

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

func TestCatalogAddressRefStore_ConversionHelpers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T, store *catalogAddressRefStore)
	}{
		{
			name: "keyToFilter",
			test: func(t *testing.T, store *catalogAddressRefStore) {
				t.Helper()
				version := semver.MustParse("1.2.3")
				key := datastore.NewAddressRefKey(12345, "LinkToken", version, "test")

				filter := store.keyToFilter(key)

				require.Equal(t, "test-domain", filter.Domain.Value)
				require.Equal(t, "catalog_testing", filter.Environment.Value)
				require.Equal(t, uint64(12345), filter.ChainSelector.Value)
				require.Equal(t, "LinkToken", filter.ContractType.Value)
				require.Equal(t, "1.2.3", filter.Version.Value)
				require.Equal(t, "test", filter.Qualifier.Value)
			},
		},
		{
			name: "protoToAddressRef_success",
			test: func(t *testing.T, store *catalogAddressRefStore) {
				t.Helper()
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

				require.NoError(t, err)
				require.Equal(t, "0x1234567890abcdef", addressRef.Address)
				require.Equal(t, uint64(12345), addressRef.ChainSelector)
				require.Equal(t, datastore.ContractType("LinkToken"), addressRef.Type)
				require.Equal(t, "1.0.0", addressRef.Version.String())
				require.Equal(t, "test", addressRef.Qualifier)
				require.Equal(t, []string{"label1", "label2"}, addressRef.Labels.List())
			},
		},
		{
			name: "protoToAddressRef_invalid_version",
			test: func(t *testing.T, store *catalogAddressRefStore) {
				t.Helper()
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

				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to parse version")
			},
		},
		{
			name: "addressRefToProto",
			test: func(t *testing.T, store *catalogAddressRefStore) {
				t.Helper()
				addressRef := newRandomAddressRef()

				protoRef := store.addressRefToProto(addressRef)

				require.Equal(t, "test-domain", protoRef.Domain)
				require.Equal(t, "catalog_testing", protoRef.Environment)
				require.Equal(t, addressRef.ChainSelector, protoRef.ChainSelector)
				require.Equal(t, string(addressRef.Type), protoRef.ContractType)
				require.Equal(t, addressRef.Version.String(), protoRef.Version)
				require.Equal(t, addressRef.Qualifier, protoRef.Qualifier)
				require.Equal(t, addressRef.Address, protoRef.Address)
				require.Equal(t, addressRef.Labels.List(), protoRef.LabelSet)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh store for each test case to avoid concurrency issues
			store := setupTestStore(t)

			tt.test(t, store)
		})
	}
}

// setupTestStore creates a real gRPC client connection to a local service
func setupTestStore(t *testing.T) *catalogAddressRefStore {
	t.Helper()
	// Get gRPC address from environment or use default
	address := os.Getenv("CATALOG_GRPC_ADDRESS")
	if address == "" {
		address = defaultGRPCAddress
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

	// Test if the service is actually available by making a simple call
	_, err = catalogClient.DataAccess()
	if err != nil {
		t.Skipf("gRPC service not available at %s: %v. Skipping integration tests.", address, err)
		return nil
	}
	t.Cleanup(func() {
		_ = catalogClient.CloseStream() // Close the test stream at the end of the test.
	})

	// Create store
	store := newCatalogAddressRefStore(catalogAddressRefStoreConfig{
		Domain:      "test-domain",
		Environment: "catalog_testing",
		Client:      catalogClient,
	})

	return store
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
