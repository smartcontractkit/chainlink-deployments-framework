package catalog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewCatalogDataStore(t *testing.T) {
	t.Parallel()
	config := CatalogDataStoreConfig{
		Domain:      "test-domain",
		Environment: "test-env",
		Client:      &CatalogClient{}, // Zero value for unit tests
	}

	dataStore := NewCatalogDataStore(config)

	// Verify the datastore is created
	require.NotNil(t, dataStore)

	// Verify all stores are initialized
	require.NotNil(t, dataStore.Addresses())
	require.NotNil(t, dataStore.ChainMetadata())
	require.NotNil(t, dataStore.ContractMetadata())
	require.NotNil(t, dataStore.EnvMetadata())
}
