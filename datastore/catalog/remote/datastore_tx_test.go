package remote

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

func TestCatalogTransactions_Commit(t *testing.T) {
	t.Parallel()

	t.Log("Setup Store")
	catalog, err := setupStore(t, t.Context())
	if err != nil {
		t.Skipf("%s", err)
		return
	}

	t.Run("Begin Transaction", func(t *testing.T) {
		err := catalog.BeginTransaction()
		require.NoError(t, err)
	})

	t.Run("Add Contract Metadata", func(t *testing.T) {
		metadata := TestContractMetadata{
			Name:        "SomeContract",
			Version:     "1.0.0",
			Description: "New contract description",
			Tags:        []string{"first"},
		}
		err := catalog.ContractMetadata().Add(t.Context(), datastore.ContractMetadata{
			Address:       "0x12345678",
			ChainSelector: 1,
			Metadata:      metadata,
		})
		assert.NoError(t, err)
	})

	t.Run("Read Contract Metadata", func(t *testing.T) {
		result, err := catalog.ContractMetadata().Get(t.Context(), datastore.NewContractMetadataKey(1, "0x12345678"))
		require.NoError(t, err)
		require.NotNil(t, result)
		metadata, err := datastore.As[TestContractMetadata](result.Metadata)
		require.NoError(t, err)
		require.NotNil(t, metadata)

		assert.Equal(t, metadata.Name, "SomeContract")
		assert.Equal(t, metadata.Version, "1.0.0")
		assert.Equal(t, metadata.Description, "New contract description")
		assert.Equal(t, metadata.Tags, []string{"first"})
	})

	t.Run("Read Contract Metadata (from outside tx)", func(t *testing.T) {
		// Check that the write happened within the transaction, and therefore a read would
		// fail outside of the transactional context.
		_, err := catalog.ContractMetadata().Get(
			t.Context(),
			datastore.NewContractMetadataKey(1, "0x12345678"),
			datastore.IgnoreTransactionsGetOption,
		)
		require.ErrorContains(t, err, "no contract metadata record can be found")
	})

	t.Run("Commit Transaction", func(t *testing.T) {
		err := catalog.BeginTransaction()
		require.NoError(t, err)
	})

	t.Run("Read Contract Metadata (from outside tx)", func(t *testing.T) {
		// Read ignoring transaction context to ensure that the commit worked.
		result, err := catalog.ContractMetadata().
			Get(t.Context(), datastore.NewContractMetadataKey(1, "0x12345678"))
		require.NoError(t, err)
		require.NotNil(t, result)
		metadata, err := datastore.As[TestContractMetadata](result.Metadata)
		require.NoError(t, err)
		require.NotNil(t, metadata)

		assert.Equal(t, metadata.Name, "SomeContract")
		assert.Equal(t, metadata.Version, "1.0.0")
		assert.Equal(t, metadata.Description, "New contract description")
		assert.Equal(t, metadata.Tags, []string{"first"})
	})
}

func TestCatalogTransactions_WithTransactions_Commit(t *testing.T) {
	t.Parallel()

	t.Log("Setup Store")
	catalog, err := setupStore(t, t.Context())
	if err != nil {
		t.Skipf("%s", err)
		return
	}

	err = catalog.WithTransaction(t.Context(), func(ctx context.Context) error {
		t.Run("Add Contract Metadata", func(t *testing.T) {
			metadata := TestContractMetadata{
				Name:        "SomeContract",
				Version:     "2.0.0",
				Description: "New contract description",
				Tags:        []string{"first"},
			}
			err := catalog.ContractMetadata().Add(t.Context(), datastore.ContractMetadata{
				Address:       "0x12345678",
				ChainSelector: 1,
				Metadata:      metadata,
			})
			assert.NoError(t, err)
		})

		t.Run("Read Contract Metadata", func(t *testing.T) {
			result, err := catalog.ContractMetadata().Get(t.Context(), datastore.NewContractMetadataKey(1, "0x12345678"))
			require.NoError(t, err)
			require.NotNil(t, result)
			metadata, err := datastore.As[TestContractMetadata](result.Metadata)
			require.NoError(t, err)
			require.NotNil(t, metadata)

			assert.Equal(t, metadata.Name, "SomeContract")
			assert.Equal(t, metadata.Version, "2.0.0")
			assert.Equal(t, metadata.Description, "New contract description")
			assert.Equal(t, metadata.Tags, []string{"first"})
		})

		t.Run("Read Contract Metadata (from outside tx)", func(t *testing.T) {
			// Check that the write happened within the transaction, and therefore a read would
			// fail outside of the transactional context.
			_, err := catalog.ContractMetadata().Get(
				t.Context(),
				datastore.NewContractMetadataKey(1, "0x12345678"),
				datastore.IgnoreTransactionsGetOption,
			)
			require.ErrorContains(t, err, "no contract metadata record can be found")
		})
		return nil
	})
	require.NoError(t, err)

	t.Run("Read Contract Metadata (from outside tx)", func(t *testing.T) {
		// Read ignoring transaction context to ensure that the commit worked.
		result, err := catalog.ContractMetadata().
			Get(t.Context(), datastore.NewContractMetadataKey(1, "0x12345678"))
		require.NoError(t, err)
		require.NotNil(t, result)
		metadata, err := datastore.As[TestContractMetadata](result.Metadata)
		require.NoError(t, err)
		require.NotNil(t, metadata)

		assert.Equal(t, metadata.Name, "SomeContract")
		assert.Equal(t, metadata.Version, "2.0.0")
		assert.Equal(t, metadata.Description, "New contract description")
		assert.Equal(t, metadata.Tags, []string{"first"})
	})
}

func TestCatalogTransactions_Rollback(t *testing.T) {
	t.Parallel()

	t.Log("Setup Store")
	catalog, err := setupStore(t, t.Context())
	if err != nil {
		t.Skipf("%s", err)
		return
	}

	t.Run("Begin Transaction", func(t *testing.T) {
		err := catalog.BeginTransaction()
		require.NoError(t, err)
	})

	t.Run("Add Contract Metadata", func(t *testing.T) {
		metadata := TestContractMetadata{
			Name:        "SomeContract",
			Version:     "3.0.0",
			Description: "New contract description",
			Tags:        []string{"first"},
		}
		err := catalog.ContractMetadata().Add(
			t.Context(),
			datastore.ContractMetadata{
				Address:       "0x12345678",
				ChainSelector: 1,
				Metadata:      metadata,
			},
		)
		assert.NoError(t, err)
	})

	t.Run("Read Contract Metadata", func(t *testing.T) {
		result, err := catalog.ContractMetadata().
			Get(t.Context(), datastore.NewContractMetadataKey(1, "0x12345678"))
		require.NoError(t, err)
		require.NotNil(t, result)
		metadata, err := datastore.As[TestContractMetadata](result.Metadata)
		require.NoError(t, err)
		require.NotNil(t, metadata)

		assert.Equal(t, metadata.Name, "SomeContract")
		assert.Equal(t, metadata.Version, "3.0.0")
		assert.Equal(t, metadata.Description, "New contract description")
		assert.Equal(t, metadata.Tags, []string{"first"})
	})

	t.Run("Read Contract Metadata (from outside tx)", func(t *testing.T) {
		// Check that the write happened within the transaction, and therefore a read would
		// fail outside of the transactional context.
		_, err := catalog.ContractMetadata().Get(
			t.Context(),
			datastore.NewContractMetadataKey(1, "0x12345678"),
			datastore.IgnoreTransactionsGetOption,
		)
		require.ErrorContains(t, err, "no contract metadata record can be found")
	})

	t.Run("Rollback Transaction", func(t *testing.T) {
		err := catalog.BeginTransaction()
		require.NoError(t, err)
	})

	t.Run("Read Contract Metadata (from outside tx)", func(t *testing.T) {
		// Ensure the write didn't happen.
		_, err := catalog.ContractMetadata().Get(
			t.Context(),
			datastore.NewContractMetadataKey(1, "0x12345678"),
			datastore.IgnoreTransactionsGetOption,
		)
		require.ErrorContains(t, err, "no contract metadata record can be found")
	})
}

func TestCatalogTransactions_WithTransactions_Rollback(t *testing.T) {
	t.Parallel()

	t.Log("Setup Store")
	catalog, err := setupStore(t, t.Context())
	if err != nil {
		t.Skipf("%s", err)
		return
	}
	err = catalog.WithTransaction(t.Context(), func(ctx context.Context) error {

		t.Run("Add Contract Metadata", func(t *testing.T) {
			metadata := TestContractMetadata{
				Name:        "SomeContract",
				Version:     "4.0.0",
				Description: "New contract description",
				Tags:        []string{"first"},
			}
			err := catalog.ContractMetadata().Add(
				t.Context(),
				datastore.ContractMetadata{
					Address:       "0x12345678",
					ChainSelector: 1,
					Metadata:      metadata,
				},
			)
			assert.NoError(t, err)
		})

		t.Run("Read Contract Metadata", func(t *testing.T) {
			result, err := catalog.ContractMetadata().
				Get(t.Context(), datastore.NewContractMetadataKey(1, "0x12345678"))
			require.NoError(t, err)
			require.NotNil(t, result)
			metadata, err := datastore.As[TestContractMetadata](result.Metadata)
			require.NoError(t, err)
			require.NotNil(t, metadata)

			assert.Equal(t, metadata.Name, "SomeContract")
			assert.Equal(t, metadata.Version, "4.0.0")
			assert.Equal(t, metadata.Description, "New contract description")
			assert.Equal(t, metadata.Tags, []string{"first"})
		})

		t.Run("Read Contract Metadata (from outside tx)", func(t *testing.T) {
			// Check that the write happened within the transaction, and therefore a read would
			// fail outside of the transactional context.
			_, err := catalog.ContractMetadata().Get(
				t.Context(),
				datastore.NewContractMetadataKey(1, "0x12345678"),
				datastore.IgnoreTransactionsGetOption,
			)
			require.ErrorContains(t, err, "no contract metadata record can be found")
		})

		return fmt.Errorf("foo")
	})
	require.ErrorContains(t, err, "foo")

	t.Run("Read Contract Metadata (from outside tx)", func(t *testing.T) {
		// Ensure the write didn't happen.
		_, err := catalog.ContractMetadata().Get(
			t.Context(),
			datastore.NewContractMetadataKey(1, "0x12345678"),
			datastore.IgnoreTransactionsGetOption,
		)
		require.ErrorContains(t, err, "no contract metadata record can be found")
	})
}

func TestCatalogTransactions_DoubleBegin(t *testing.T) {
	t.Skipf("Expected behavior from server has a bug, so skip for now.")
	t.Parallel()

	t.Log("Setup Store")
	catalog, err := setupStore(t, t.Context())
	if err != nil {
		t.Skipf("%s", err)
		return
	}

	t.Run("Begin Transaction", func(t *testing.T) {
		err := catalog.BeginTransaction()
		require.NoError(t, err)
	})

	t.Run("Begin Second (conflicting) Transaction", func(t *testing.T) {
		err := catalog.BeginTransaction()
		require.ErrorContains(t, err, "blah")
	})
}

func TestCatalogTransactions_OrphanCommit(t *testing.T) {
	t.Parallel()

	t.Log("Setup Store")
	catalog, err := setupStore(t, t.Context())
	if err != nil {
		t.Skipf("%s", err)
		return
	}

	t.Run("Commit Without Transaction", func(t *testing.T) {
		err := catalog.CommitTransaction()
		require.NoError(t, err)
	})
}

func TestCatalogTransactions_OrphanRollback(t *testing.T) {
	t.Parallel()

	t.Log("Setup Store")
	catalog, err := setupStore(t, t.Context())
	if err != nil {
		t.Skipf("%s", err)
		return
	}

	t.Run("Commit Without Transaction", func(t *testing.T) {
		err := catalog.CommitTransaction()
		require.NoError(t, err)
	})
}

// setupStore creates a new catalog store.
//
// It returns the underlying type, since the *Transaction methods aren't exposed in the API yet.
func setupStore(t *testing.T, ctx context.Context) (*catalogDataStore, error) {
	t.Helper()
	// Get gRPC address from environment or use default
	address := os.Getenv("CATALOG_GRPC_ADDRESS")
	if address == "" {
		address = defaultGRPCAddress
	}
	// Create CatalogClient using the NewCatalogClient function
	catalogClient, err := NewCatalogClient(ctx, CatalogConfig{
		GRPC:  address,
		Creds: insecure.NewCredentials(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server at %s: %v. Skipping integration tests", address, err)
	}
	// Test if the service is actually available by making a simple call
	_, err = catalogClient.DataAccess()
	if err != nil {
		return nil, fmt.Errorf("gRPC service not available at %s: %v. Skipping integration tests", address, err)
	}
	config := CatalogDataStoreConfig{
		Domain:      fmt.Sprintf("test-domain-%d", rand.IntN(1000000000)),
		Environment: "catalog_testing",
		Client:      catalogClient,
	}
	return NewCatalogDataStore(config), nil
}
