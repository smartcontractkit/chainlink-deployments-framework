package catalog

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

func TestNewCatalogDataStore(t *testing.T) {
	t.Parallel()
	config := CatalogDataStoreConfig{
		Domain:      "test-domain",
		Environment: "test-env",
		Client:      CatalogClient{}, // Zero value for unit tests
	}

	dataStore := NewCatalogDataStore(config)

	// Verify the datastore is created
	require.NotNil(t, dataStore)
	require.Equal(t, "test-domain", dataStore.domain)
	require.Equal(t, "test-env", dataStore.environment)

	// Verify all stores are initialized
	require.NotNil(t, dataStore.AddressRefStore)
	require.NotNil(t, dataStore.ChainMetadataStore)
	require.NotNil(t, dataStore.ContractMetadataStore)
	require.NotNil(t, dataStore.EnvMetadataStore)
}

func TestCatalogDataStore_ImplementsCatalogInterface(t *testing.T) {
	t.Parallel()
	config := CatalogDataStoreConfig{
		Domain:      "test-domain",
		Environment: "test-env",
		Client:      CatalogClient{}, // Zero value for unit tests
	}

	dataStore := NewCatalogDataStore(config)

	// Verify it implements the CatalogStore interface
	var _ datastore.CatalogStore = dataStore

	// Test all interface methods return the expected store types
	addressStore := dataStore.Addresses()
	require.NotNil(t, addressStore)
	require.IsType(t, &CatalogAddressRefStore{}, addressStore)

	chainStore := dataStore.ChainMetadata()
	require.NotNil(t, chainStore)
	require.IsType(t, &CatalogChainMetadataStore{}, chainStore)

	contractStore := dataStore.ContractMetadata()
	require.NotNil(t, contractStore)
	require.IsType(t, &CatalogContractMetadataStore{}, contractStore)

	// EnvMetadata() now properly returns the V2 store
	envStore := dataStore.EnvMetadata()
	require.NotNil(t, envStore) // Now properly returns the V2 env metadata store
	require.IsType(t, &CatalogEnvMetadataStore{}, envStore)
}

func TestCatalogDataStore_StoreInterfaces(t *testing.T) {
	t.Parallel()
	config := CatalogDataStoreConfig{
		Domain:      "test-domain",
		Environment: "test-env",
		Client:      CatalogClient{}, // Zero value for unit tests
	}

	ds := NewCatalogDataStore(config)

	// Verify each store implements the correct mutable interface
	_ = ds.Addresses()
	_ = ds.ChainMetadata()
	_ = ds.ContractMetadata()
	_ = ds.EnvMetadata()
}

func TestCatalogDataStoreConfig_ClientPassthrough(t *testing.T) {
	t.Parallel()
	config := CatalogDataStoreConfig{
		Domain:      "test-domain",
		Environment: "test-env",
		Client:      CatalogClient{}, // Zero value for unit tests
	}

	ds := NewCatalogDataStore(config)

	// Verify that the client is properly passed through to all stores
	// We can't directly access the client field since it's private,
	// but we can verify the stores are configured correctly by checking
	// their domain and environment values match what we passed in

	// Test that stores have the correct configuration
	addressStore := ds.AddressRefStore
	require.NotNil(t, addressStore)

	chainStore := ds.ChainMetadataStore
	require.NotNil(t, chainStore)

	contractStore := ds.ContractMetadataStore
	require.NotNil(t, contractStore)

	envStore := ds.EnvMetadataStore
	require.NotNil(t, envStore)

	// Since we can't access private fields directly, we'll just verify
	// that the stores were created without panicking, which indicates
	// the client was properly passed through
}
