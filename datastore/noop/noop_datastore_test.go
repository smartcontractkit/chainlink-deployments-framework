package noop

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

func TestNewNoopDataStore(t *testing.T) {
	t.Parallel()

	ds := NewNoopDataStore()
	require.NotNil(t, ds)

	// Verify it implements the DataStore interface
	var _ datastore.DataStore = ds

	// Verify all stores are initialized
	require.NotNil(t, ds.Addresses())
	require.NotNil(t, ds.ChainMetadata())
	require.NotNil(t, ds.ContractMetadata())
	require.NotNil(t, ds.EnvMetadata())
}

func TestNoopDataStore_OperationsReturnErrors(t *testing.T) {
	t.Parallel()

	ds := NewNoopDataStore()

	// Test AddressRefStore operations
	_, err := ds.Addresses().Fetch()
	require.Error(t, err)
	require.Equal(t, ErrNoopNotSupported, err)

	// Test ChainMetadataStore operations
	_, err = ds.ChainMetadata().Fetch()
	require.Error(t, err)
	require.Equal(t, ErrNoopNotSupported, err)

	// Test ContractMetadataStore operations
	_, err = ds.ContractMetadata().Fetch()
	require.Error(t, err)
	require.Equal(t, ErrNoopNotSupported, err)

	// Test EnvMetadataStore operations
	_, err = ds.EnvMetadata().Get()
	require.Error(t, err)
	require.Equal(t, ErrNoopNotSupported, err)
}

func TestNoopDataStore_FilterOperationsReturnEmpty(t *testing.T) {
	t.Parallel()

	ds := NewNoopDataStore()

	// Test Filter operations return empty slices (don't error out)
	addressRefs := ds.Addresses().Filter()
	require.Empty(t, addressRefs)

	chainMetadata := ds.ChainMetadata().Filter()
	require.Empty(t, chainMetadata)

	contractMetadata := ds.ContractMetadata().Filter()
	require.Empty(t, contractMetadata)
}
