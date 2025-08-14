package remote

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

//nolint:paralleltest
func TestCatalogTransactions_Commit(t *testing.T) {
	t.Parallel()

	t.Log("Setup Store")
	catalog, err := setupStore(t, t.Context())
	if err != nil {
		t.Skipf("%s", err)
		return
	}

	t.Run("Begin Transaction", func(t *testing.T) {
		err := catalog.beginTransaction()
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

		assert.Equal(t, "SomeContract", metadata.Name)
		assert.Equal(t, "1.0.0", metadata.Version)
		assert.Equal(t, "New contract description", metadata.Description)
		assert.Equal(t, []string{"first"}, metadata.Tags)
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
		err := catalog.beginTransaction()
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

		assert.Equal(t, "SomeContract", metadata.Name)
		assert.Equal(t, "1.0.0", metadata.Version)
		assert.Equal(t, "New contract description", metadata.Description)
		assert.Equal(t, []string{"first"}, metadata.Tags)
	})
}

//nolint:paralleltest
func TestCatalogTransactions_WithTransactions_Commit(t *testing.T) {
	t.Parallel()

	t.Log("Setup Store")
	catalog, err := setupStore(t, t.Context())
	if err != nil {
		t.Skipf("%s", err)
		return
	}

	err = catalog.WithTransaction(t.Context(), func(ctx context.Context, catalog datastore.BaseCatalogStore) error {
		t.Run("Add Contract Metadata", func(t *testing.T) {
			metadata := TestContractMetadata{
				Name:        "SomeContract",
				Version:     "2.0.0",
				Description: "New contract description",
				Tags:        []string{"first"},
			}
			err2 := catalog.ContractMetadata().Add(t.Context(), datastore.ContractMetadata{
				Address:       "0x12345678",
				ChainSelector: 1,
				Metadata:      metadata,
			})
			assert.NoError(t, err2)
		})

		t.Run("Read Contract Metadata", func(t *testing.T) {
			result, err2 := catalog.ContractMetadata().Get(t.Context(), datastore.NewContractMetadataKey(1, "0x12345678"))
			require.NoError(t, err2)
			require.NotNil(t, result)
			metadata, err2 := datastore.As[TestContractMetadata](result.Metadata)
			require.NoError(t, err2)
			require.NotNil(t, metadata)

			assert.Equal(t, "SomeContract", metadata.Name)
			assert.Equal(t, "2.0.0", metadata.Version)
			assert.Equal(t, "New contract description", metadata.Description)
			assert.Equal(t, []string{"first"}, metadata.Tags)
		})

		t.Run("Read Contract Metadata (from outside tx)", func(t *testing.T) {
			// Check that the write happened within the transaction, and therefore a read would
			// fail outside of the transactional context.
			_, err2 := catalog.ContractMetadata().Get(
				t.Context(),
				datastore.NewContractMetadataKey(1, "0x12345678"),
				datastore.IgnoreTransactionsGetOption,
			)
			require.ErrorContains(t, err2, "no contract metadata record can be found")
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

		assert.Equal(t, "SomeContract", metadata.Name)
		assert.Equal(t, "2.0.0", metadata.Version)
		assert.Equal(t, "New contract description", metadata.Description)
		assert.Equal(t, []string{"first"}, metadata.Tags)
	})
}

//nolint:paralleltest
func TestCatalogTransactions_Rollback(t *testing.T) {
	t.Parallel()

	t.Log("Setup Store")
	catalog, err := setupStore(t, t.Context())
	if err != nil {
		t.Skipf("%s", err)
		return
	}

	t.Run("Begin Transaction", func(t *testing.T) {
		err := catalog.beginTransaction()
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

		assert.Equal(t, "SomeContract", metadata.Name)
		assert.Equal(t, "3.0.0", metadata.Version)
		assert.Equal(t, "New contract description", metadata.Description)
		assert.Equal(t, []string{"first"}, metadata.Tags)
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
		err := catalog.beginTransaction()
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

//nolint:paralleltest
func TestCatalogTransactions_WithTransactions_Rollback(t *testing.T) {
	t.Parallel()

	t.Log("Setup Store")
	catalog, err := setupStore(t, t.Context())
	if err != nil {
		t.Skipf("%s", err)
		return
	}
	err = catalog.WithTransaction(t.Context(), func(ctx context.Context, catalog datastore.BaseCatalogStore) error {
		t.Run("Add Contract Metadata", func(t *testing.T) {
			metadata := TestContractMetadata{
				Name:        "SomeContract",
				Version:     "4.0.0",
				Description: "New contract description",
				Tags:        []string{"first"},
			}
			err2 := catalog.ContractMetadata().Add(
				t.Context(),
				datastore.ContractMetadata{
					Address:       "0x12345678",
					ChainSelector: 1,
					Metadata:      metadata,
				},
			)
			assert.NoError(t, err2)
		})
		t.Run("Read Contract Metadata", func(t *testing.T) {
			result, err2 := catalog.ContractMetadata().
				Get(t.Context(), datastore.NewContractMetadataKey(1, "0x12345678"))
			require.NoError(t, err2)
			require.NotNil(t, result)
			metadata, err2 := datastore.As[TestContractMetadata](result.Metadata)
			require.NoError(t, err2)
			require.NotNil(t, metadata)

			assert.Equal(t, "SomeContract", metadata.Name)
			assert.Equal(t, "4.0.0", metadata.Version)
			assert.Equal(t, "New contract description", metadata.Description)
			assert.Equal(t, []string{"first"}, metadata.Tags)
		})

		t.Run("Read Contract Metadata (from outside tx)", func(t *testing.T) {
			// Check that the write happened within the transaction, and therefore a read would
			// fail outside of the transactional context.
			_, err2 := catalog.ContractMetadata().Get(
				t.Context(),
				datastore.NewContractMetadataKey(1, "0x12345678"),
				datastore.IgnoreTransactionsGetOption,
			)
			require.ErrorContains(t, err2, "no contract metadata record can be found")
		})

		return errors.New("foo")
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

//nolint:paralleltest
func TestCatalogTransactions_WithTransactions_Panic(t *testing.T) {
	t.Parallel()

	t.Log("Setup Store")
	catalog, err := setupStore(t, t.Context())
	if err != nil {
		t.Skipf("%s", err)
		return
	}
	defer func() {
		// Do this check in a defer, since the panic logic handling in WithTransaction re-panics
		r := recover()
		assert.NotNil(t, r)
		assert.Equal(t, "foo", r)
		t.Run("Ensure write was rolled-back", func(t *testing.T) {
			_, err2 := catalog.ContractMetadata().Get(
				t.Context(),
				datastore.NewContractMetadataKey(1, "0x12345678"),
				datastore.IgnoreTransactionsGetOption,
			)
			require.ErrorContains(t, err2, "no contract metadata record can be found")
		})
	}()

	_ = catalog.WithTransaction(t.Context(), func(ctx context.Context, catalog datastore.BaseCatalogStore) error {
		t.Run("Add Contract Metadata", func(t *testing.T) {
			metadata := TestContractMetadata{
				Name:        "SomeContract",
				Version:     "4.0.0",
				Description: "New contract description",
				Tags:        []string{"first"},
			}
			err2 := catalog.ContractMetadata().Add(
				t.Context(),
				datastore.ContractMetadata{
					Address:       "0x12345678",
					ChainSelector: 1,
					Metadata:      metadata,
				},
			)
			assert.NoError(t, err2)
		})

		t.Run("Read Contract Metadata", func(t *testing.T) {
			result, err2 := catalog.ContractMetadata().
				Get(t.Context(), datastore.NewContractMetadataKey(1, "0x12345678"))
			require.NoError(t, err2)
			require.NotNil(t, result)
			metadata, err2 := datastore.As[TestContractMetadata](result.Metadata)
			require.NoError(t, err2)
			require.NotNil(t, metadata)

			assert.Equal(t, "SomeContract", metadata.Name)
			assert.Equal(t, "4.0.0", metadata.Version)
			assert.Equal(t, "New contract description", metadata.Description)
			assert.Equal(t, []string{"first"}, metadata.Tags)
		})
		panic("foo")
	})
}

//nolint:paralleltest
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
		err := catalog.beginTransaction()
		require.NoError(t, err)
	})

	t.Run("Begin Second (conflicting) Transaction", func(t *testing.T) {
		err := catalog.beginTransaction()
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
		err := catalog.commitTransaction()
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
		err := catalog.commitTransaction()
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
		return nil, fmt.Errorf("failed to connect to gRPC server at %s: %w. Skipping integration tests", address, err)
	}
	// Test if the service is actually available by making a simple call
	_, err = catalogClient.DataAccess()
	if err != nil {
		return nil, fmt.Errorf("gRPC service not available at %s: %w. Skipping integration tests", address, err)
	}
	random, err := rand.Int(rand.Reader, big.NewInt(1000000000))
	require.NoError(t, err)
	config := CatalogDataStoreConfig{
		Domain:      fmt.Sprintf("test-domain-%d", random),
		Environment: "catalog_testing",
		Client:      catalogClient,
	}

	return NewCatalogDataStore(config), nil
}
