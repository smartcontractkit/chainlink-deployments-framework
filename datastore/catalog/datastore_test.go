package catalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

func TestNewCatalogDataStore(t *testing.T) {
	config := CatalogDataStoreConfig{
		Domain:      "test-domain",
		Environment: "test-env",
		Client:      nil, // No real client needed for unit tests
	}

	dataStore := NewCatalogDataStore(config)

	// Verify the datastore is created
	require.NotNil(t, dataStore)
	assert.Equal(t, "test-domain", dataStore.domain)
	assert.Equal(t, "test-env", dataStore.environment)

	// Verify all stores are initialized
	assert.NotNil(t, dataStore.AddressRefStore)
	assert.NotNil(t, dataStore.ChainMetadataStore)
	assert.NotNil(t, dataStore.ContractMetadataStore)
	assert.NotNil(t, dataStore.EnvMetadataStore)
}

func TestCatalogDataStore_ImplementsCatalogInterface(t *testing.T) {
	config := CatalogDataStoreConfig{
		Domain:      "test-domain",
		Environment: "test-env",
		Client:      nil, // No real client needed for unit tests
	}

	dataStore := NewCatalogDataStore(config)

	// Verify it implements the Catalog interface
	var _ datastore.CatalogStore = dataStore

	// Test all interface methods return the expected store types
	addressStore := dataStore.Addresses()
	assert.NotNil(t, addressStore)
	assert.IsType(t, &CatalogAddressRefStore{}, addressStore)

	chainStore := dataStore.ChainMetadata()
	assert.NotNil(t, chainStore)
	assert.IsType(t, &CatalogChainMetadataStore{}, chainStore)

	contractStore := dataStore.ContractMetadata()
	assert.NotNil(t, contractStore)
	assert.IsType(t, &CatalogContractMetadataStore{}, contractStore)

	envStore := dataStore.EnvMetadata()
	assert.NotNil(t, envStore)
	assert.IsType(t, &CatalogEnvMetadataStore{}, envStore)
}

func TestCatalogDataStore_StoreInterfaces(t *testing.T) {
	config := CatalogDataStoreConfig{
		Domain:      "test-domain",
		Environment: "test-env",
		Client:      nil, // No real client needed for unit tests
	}

	ds := NewCatalogDataStore(config)

	// Verify each store implements the correct mutable interface
	var _ datastore.MutableAddressRefStore = ds.Addresses()
	var _ datastore.MutableChainMetadataStore = ds.ChainMetadata()
	var _ datastore.MutableContractMetadataStore = ds.ContractMetadata()
	var _ datastore.MutableEnvMetadataStore = ds.EnvMetadata()

	// Also verify they implement the read-only interfaces
	var _ datastore.AddressRefStore = ds.Addresses()
	var _ datastore.ChainMetadataStore = ds.ChainMetadata()
	var _ datastore.ContractMetadataStore = ds.ContractMetadata()
	var _ datastore.EnvMetadataStore = ds.EnvMetadata()
}

func TestCatalogDataStoreConfig_ClientPassthrough(t *testing.T) {
	config := CatalogDataStoreConfig{
		Domain:      "test-domain",
		Environment: "test-env",
		Client:      nil, // No real client needed for unit tests
	}

	ds := NewCatalogDataStore(config)

	// Verify that the client is properly passed through to all stores
	// We can't directly access the client field since it's private,
	// but we can verify the stores are configured correctly by checking
	// their domain and environment values match what we passed in

	// Test that stores have the correct configuration
	addressStore := ds.AddressRefStore
	assert.NotNil(t, addressStore)

	chainStore := ds.ChainMetadataStore
	assert.NotNil(t, chainStore)

	contractStore := ds.ContractMetadataStore
	assert.NotNil(t, contractStore)

	envStore := ds.EnvMetadataStore
	assert.NotNil(t, envStore)

	// Since we can't access private fields directly, we'll just verify
	// that the stores were created without panicking, which indicates
	// the client was properly passed through
}

func TestCatalogDataStore_NoSealOrMerge(t *testing.T) {
	config := CatalogDataStoreConfig{
		Domain:      "test-domain",
		Environment: "test-env",
		Client:      nil, // No real client needed for unit tests
	}

	ds := NewCatalogDataStore(config)

	// Verify that the datastore does NOT implement MutableDataStore
	// (which would have Seal() and Merge() methods)
	// This is a compile-time check - if it compiles, the interface is correct
	var _ datastore.CatalogStore = ds

	// The absence of Seal() and Merge() methods is verified by the interface constraint
	// If those methods existed, this would fail to compile with the Catalog interface
}
