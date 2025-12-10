package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

// setupEnvMetadataTestStore creates a new memory datastore for testing environment metadata
func setupEnvMetadataTestStore(t *testing.T) *memoryCatalogDataStore {
	t.Helper()
	store, err := NewMemoryCatalogDataStore()
	require.NoError(t, err)

	return store
}

func TestCatalogEnvMetadataStore_Get(t *testing.T) {
	t.Parallel()

	t.Run("not set", func(t *testing.T) {
		t.Parallel()
		store := setupEnvMetadataTestStore(t)

		_, err := store.EnvMetadata().Get(t.Context())
		require.Error(t, err)
		require.ErrorIs(t, err, datastore.ErrEnvMetadataNotSet)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		store := setupEnvMetadataTestStore(t)
		envMetadata := map[string]any{
			"domain":      "test_domain",
			"environment": "catalog_testing",
			"version":     float64(1), // JSON unmarshals numbers as float64
			"active":      true,
		}
		err := store.EnvMetadata().Set(t.Context(), envMetadata)
		require.NoError(t, err)

		result, err := store.EnvMetadata().Get(t.Context())
		require.NoError(t, err)
		require.Equal(t, envMetadata, result.Metadata)
	})

	t.Run("success with nil metadata", func(t *testing.T) {
		t.Parallel()
		store := setupEnvMetadataTestStore(t)

		err := store.EnvMetadata().Set(t.Context(), nil)
		require.NoError(t, err)

		result, err := store.EnvMetadata().Get(t.Context())
		require.NoError(t, err)
		require.Nil(t, result.Metadata)
	})
}

func TestCatalogEnvMetadataStore_Set(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata any
	}{
		{
			name: "simple metadata",
			metadata: map[string]any{
				"domain":      "test_domain",
				"environment": "test_env",
				"version":     float64(1), // JSON unmarshals numbers as float64
			},
		},
		{
			name: "complex metadata",
			metadata: map[string]any{
				"domain":      "production",
				"environment": "mainnet",
				"version":     float64(2), // JSON unmarshals numbers as float64
				"config": map[string]any{
					"maxRetries":    float64(3),
					"timeout":       float64(30000),
					"enableLogging": true,
				},
				"chains": []any{
					map[string]any{"name": "Ethereum", "id": float64(1)},
					map[string]any{"name": "Polygon", "id": float64(137)},
				},
				"features": []any{"monitoring", "alerting", "backup"},
			},
		},
		{
			name:     "nil metadata",
			metadata: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := setupEnvMetadataTestStore(t)

			// Execute
			err := store.EnvMetadata().Set(context.Background(), tt.metadata)
			require.NoError(t, err)

			// Verify
			result, err := store.EnvMetadata().Get(context.Background())
			require.NoError(t, err)
			require.Equal(t, tt.metadata, result.Metadata)
		})
	}
}

func TestCatalogEnvMetadataStore_Set_Replace(t *testing.T) {
	t.Parallel()

	store := setupEnvMetadataTestStore(t)

	// Set initial metadata
	initialMetadata := map[string]any{
		"domain":      "test_domain",
		"environment": "test_env",
		"version":     float64(1), // JSON unmarshals numbers as float64
		"active":      true,
	}
	err := store.EnvMetadata().Set(t.Context(), initialMetadata)
	require.NoError(t, err)

	// Verify initial metadata
	result, err := store.EnvMetadata().Get(t.Context())
	require.NoError(t, err)
	require.Equal(t, initialMetadata, result.Metadata)

	// Replace with new metadata
	newMetadata := map[string]any{
		"domain":      "production",
		"environment": "mainnet",
		"version":     float64(2), // JSON unmarshals numbers as float64
		"active":      false,
		"newField":    "newValue",
	}
	err = store.EnvMetadata().Set(t.Context(), newMetadata)
	require.NoError(t, err)

	// Verify new metadata replaced the old one
	result, err = store.EnvMetadata().Get(t.Context())
	require.NoError(t, err)
	require.Equal(t, newMetadata, result.Metadata)
}

func TestCatalogEnvMetadataStore_Set_WithCustomUpdater(t *testing.T) {
	t.Parallel()

	store := setupEnvMetadataTestStore(t)

	// Set initial metadata
	initialMetadata := map[string]any{
		"value": "initial",
	}
	err := store.EnvMetadata().Set(t.Context(), initialMetadata)
	require.NoError(t, err)

	// Update with a simple custom updater that appends to a string
	updateMetadata := map[string]any{
		"value": "updated",
	}

	// Simple custom updater that concatenates strings to prove it was called
	customUpdater := func(latest any, incoming any) (any, error) {
		latestMap, _ := latest.(map[string]any)
		incomingMap, _ := incoming.(map[string]any)

		result := map[string]any{
			"value": latestMap["value"].(string) + "_" + incomingMap["value"].(string) + "_custom",
		}

		return result, nil
	}

	err = store.EnvMetadata().Set(t.Context(), updateMetadata, datastore.WithUpdater(customUpdater))
	require.NoError(t, err)

	// Verify the custom updater was used
	result, err := store.EnvMetadata().Get(t.Context())
	require.NoError(t, err)

	resultMap, ok := result.Metadata.(map[string]any)
	require.True(t, ok)
	// If the custom updater was called, we should see "initial_updated_custom"
	require.Equal(t, "initial_updated_custom", resultMap["value"])
}

func TestCatalogEnvMetadataStore_Transactions(t *testing.T) {
	t.Parallel()
	t.Run("transaction rollback", func(t *testing.T) {
		t.Parallel()
		store := setupEnvMetadataTestStore(t)

		envMetadata := map[string]any{
			"domain":      "test_domain",
			"environment": "test_env",
			"version":     float64(1), // JSON unmarshals numbers as float64
		}

		err := store.WithTransaction(t.Context(), func(ctx context.Context, txStore datastore.BaseCatalogStore) error {
			// Set metadata within transaction
			setErr := txStore.EnvMetadata().Set(ctx, envMetadata)
			require.NoError(t, setErr)

			// Verify it exists within transaction
			_, getErr := txStore.EnvMetadata().Get(ctx)
			require.NoError(t, getErr)

			// Force rollback by returning error
			return errors.New("force rollback")
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "force rollback")

		// Verify metadata doesn't exist after rollback
		_, err = store.EnvMetadata().Get(t.Context())
		require.Error(t, err)
		require.ErrorIs(t, err, datastore.ErrEnvMetadataNotSet)
	})

	t.Run("transaction commit", func(t *testing.T) {
		t.Parallel()
		store := setupEnvMetadataTestStore(t)

		envMetadata := map[string]any{
			"domain":      "test_domain",
			"environment": "test_env",
			"version":     float64(1), // JSON unmarshals numbers as float64
		}

		err := store.WithTransaction(t.Context(), func(ctx context.Context, txStore datastore.BaseCatalogStore) error {
			// Set metadata within transaction
			return txStore.EnvMetadata().Set(ctx, envMetadata)
		})
		require.NoError(t, err)

		// Verify metadata exists after commit
		result, err := store.EnvMetadata().Get(t.Context())
		require.NoError(t, err)
		require.Equal(t, envMetadata, result.Metadata)
	})

	t.Run("ignore transactions option", func(t *testing.T) {
		t.Parallel()
		store := setupEnvMetadataTestStore(t)

		envMetadata := map[string]any{
			"domain":      "test_domain",
			"environment": "test_env",
			"version":     float64(1), // JSON unmarshals numbers as float64
		}

		// Set metadata outside transaction
		err := store.EnvMetadata().Set(t.Context(), envMetadata)
		require.NoError(t, err)

		err = store.WithTransaction(t.Context(), func(ctx context.Context, txStore datastore.BaseCatalogStore) error {
			// Should be able to read with ignore transactions option
			result, getErr := txStore.EnvMetadata().Get(ctx, datastore.IgnoreTransactionsGetOption)
			require.NoError(t, getErr)
			require.Equal(t, envMetadata, result.Metadata)

			// Force rollback
			return errors.New("force rollback")
		})
		require.Error(t, err)

		// Metadata should still exist since it was set outside transaction
		_, err = store.EnvMetadata().Get(t.Context())
		require.NoError(t, err)
	})
}

func TestCatalogEnvMetadataStore_SingleRecord(t *testing.T) {
	t.Parallel()
	store := setupEnvMetadataTestStore(t)

	// Set initial metadata
	metadata1 := map[string]any{
		"version": float64(1),
		"config":  "initial",
	}
	err := store.EnvMetadata().Set(t.Context(), metadata1)
	require.NoError(t, err)

	result, err := store.EnvMetadata().Get(t.Context())
	require.NoError(t, err)
	require.Equal(t, metadata1, result.Metadata)

	// Set new metadata - should replace, not add
	metadata2 := map[string]any{
		"version": float64(2),
		"config":  "updated",
	}
	err = store.EnvMetadata().Set(t.Context(), metadata2)
	require.NoError(t, err)

	result, err = store.EnvMetadata().Get(t.Context())
	require.NoError(t, err)
	require.Equal(t, metadata2, result.Metadata)

	// There should still be only one record
	// We can't directly test this without accessing the database,
	// but the fact that Get() works correctly indicates single record behavior
}

func TestCatalogEnvMetadataStore_JSONSerialization(t *testing.T) {
	t.Parallel()
	store := setupEnvMetadataTestStore(t)

	// Test various JSON-serializable types
	complexMetadata := map[string]any{
		"string":  "test string",
		"number":  float64(42), // JSON unmarshals numbers as float64
		"boolean": true,
		"null":    nil,
		"array":   []any{"item1", float64(2), true, nil},
		"object": map[string]any{
			"nested": map[string]any{
				"deeply": map[string]any{
					"nested": "value",
				},
			},
		},
	}

	err := store.EnvMetadata().Set(t.Context(), complexMetadata)
	require.NoError(t, err)

	result, err := store.EnvMetadata().Get(t.Context())
	require.NoError(t, err)
	require.Equal(t, complexMetadata, result.Metadata)
}
