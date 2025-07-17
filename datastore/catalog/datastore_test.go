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

	// Verify all stores are initialized
	require.NotNil(t, dataStore.addressRefStore)
	require.NotNil(t, dataStore.chainMetadataStore)
	require.NotNil(t, dataStore.contractMetadataStore)
	require.NotNil(t, dataStore.envMetadataStore)
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
	require.IsType(t, &catalogAddressRefStore{}, addressStore)

	chainStore := dataStore.ChainMetadata()
	require.NotNil(t, chainStore)
	require.IsType(t, &catalogChainMetadataStore{}, chainStore)

	contractStore := dataStore.ContractMetadata()
	require.NotNil(t, contractStore)
	require.IsType(t, &catalogContractMetadataStore{}, contractStore)

	// EnvMetadata() now properly returns the V2 store
	envStore := dataStore.EnvMetadata()
	require.NotNil(t, envStore) // Now properly returns the V2 env metadata store
	require.IsType(t, &catalogEnvMetadataStore{}, envStore)
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
	// nolint:staticcheck
	var _ datastore.MutableRefStoreV2[datastore.AddressRefKey, datastore.AddressRef] = ds.Addresses()
	// nolint:staticcheck
	var _ datastore.MutableStoreV2[datastore.ChainMetadataKey, datastore.ChainMetadata] = ds.ChainMetadata()
	// nolint:staticcheck
	var _ datastore.MutableStoreV2[datastore.ContractMetadataKey, datastore.ContractMetadata] = ds.ContractMetadata()
	// nolint:staticcheck
	var _ datastore.MutableUnaryStoreV2[datastore.EnvMetadata] = ds.EnvMetadata()
}
