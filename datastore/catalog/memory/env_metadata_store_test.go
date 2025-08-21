package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

// setupEnvMetadataTestStore creates a new memory datastore for testing environment metadata
func setupEnvMetadataTestStore(t *testing.T) (*memoryDataStore, func()) {
	t.Helper()
	config := MemoryDataStoreConfig{
		Domain:      "test_domain",
		Environment: "catalog_testing",
	}
	store := NewMemoryDataStore(t, config)
	return store, func() {
		store.Close()
	}
}

func TestCatalogEnvMetadataStore_Get(t *testing.T) {
	t.Parallel()
	store, closer := setupEnvMetadataTestStore(t)
	defer closer()

	t.Run("not set", func(t *testing.T) {
		t.Parallel()

		_, err := store.EnvMetadata().Get(context.Background())
		require.Error(t, err)
		require.ErrorIs(t, err, datastore.ErrEnvMetadataNotSet)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		envMetadata := map[string]any{
			"domain":      "test_domain",
			"environment": "catalog_testing",
			"version":     float64(1), // JSON unmarshals numbers as float64
			"active":      true,
		}
		err := store.EnvMetadata().Set(context.Background(), envMetadata)
		require.NoError(t, err)

		result, err := store.EnvMetadata().Get(context.Background())
		require.NoError(t, err)
		require.Equal(t, envMetadata, result.Metadata)
	})

	t.Run("success with nil metadata", func(t *testing.T) {
		t.Parallel()

		store2, closer2 := setupEnvMetadataTestStore(t)
		defer closer2()

		err := store2.EnvMetadata().Set(context.Background(), nil)
		require.NoError(t, err)

		result, err := store2.EnvMetadata().Get(context.Background())
		require.NoError(t, err)
		require.Nil(t, result.Metadata)
	})
}

func TestCatalogEnvMetadataStore_Set(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		setup    func(store *memoryDataStore) any
		expected any
	}{
		{
			name: "simple metadata",
			setup: func(store *memoryDataStore) any {
				return map[string]any{
					"domain":      "test_domain",
					"environment": "test_env",
					"version":     float64(1), // JSON unmarshals numbers as float64
				}
			},
			expected: map[string]any{
				"domain":      "test_domain",
				"environment": "test_env",
				"version":     float64(1),
			},
		},
		{
			name: "complex metadata",
			setup: func(store *memoryDataStore) any {
				return map[string]any{
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
				}
			},
			expected: map[string]any{
				"domain":      "production",
				"environment": "mainnet",
				"version":     float64(2),
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
			name: "nil metadata",
			setup: func(store *memoryDataStore) any {
				return nil
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh store for each test case
			store, closer := setupEnvMetadataTestStore(t)
			defer closer()

			metadata := tt.setup(store)

			// Execute
			err := store.EnvMetadata().Set(context.Background(), metadata)
			require.NoError(t, err)

			// Verify
			result, err := store.EnvMetadata().Get(context.Background())
			require.NoError(t, err)
			require.Equal(t, tt.expected, result.Metadata)
		})
	}
}

func TestCatalogEnvMetadataStore_Set_Replace(t *testing.T) {
	t.Parallel()
	store, closer := setupEnvMetadataTestStore(t)
	defer closer()

	// Set initial metadata
	initialMetadata := map[string]any{
		"domain":      "test_domain",
		"environment": "test_env",
		"version":     float64(1), // JSON unmarshals numbers as float64
		"active":      true,
	}
	err := store.EnvMetadata().Set(context.Background(), initialMetadata)
	require.NoError(t, err)

	// Verify initial metadata
	result, err := store.EnvMetadata().Get(context.Background())
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
	err = store.EnvMetadata().Set(context.Background(), newMetadata)
	require.NoError(t, err)

	// Verify new metadata replaced the old one
	result, err = store.EnvMetadata().Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, newMetadata, result.Metadata)
}

func TestCatalogEnvMetadataStore_Set_WithCustomUpdater(t *testing.T) {
	t.Parallel()
	store, closer := setupEnvMetadataTestStore(t)
	defer closer()

	// Set initial metadata
	initialMetadata := map[string]any{
		"domain":      "test_domain",
		"environment": "test_env",
		"version":     float64(1), // JSON unmarshals numbers as float64
		"config": map[string]any{
			"timeout": float64(5000),
			"retries": float64(3),
		},
	}
	err := store.EnvMetadata().Set(context.Background(), initialMetadata)
	require.NoError(t, err)

	// Update with custom merger that combines maps
	updateMetadata := map[string]any{
		"version": float64(2), // JSON unmarshals numbers as float64
		"config": map[string]any{
			"timeout":   float64(10000), // should override
			"newOption": "newValue",     // should be added
		},
		"newField": "addedField",
	}

	customUpdater := func(latest any, incoming any) (any, error) {
		latestMap, ok1 := latest.(map[string]any)
		incomingMap, ok2 := incoming.(map[string]any)
		if !ok1 || !ok2 {
			return incoming, nil // fallback to replacement
		}

		// Deep merge maps
		result := make(map[string]any)
		for k, v := range latestMap {
			result[k] = v
		}
		for k, v := range incomingMap {
			if k == "config" {
				// Special handling for config - merge nested maps
				if latestConfig, ok := result[k].(map[string]any); ok {
					if incomingConfig, ok := v.(map[string]any); ok {
						mergedConfig := make(map[string]any)
						for ck, cv := range latestConfig {
							mergedConfig[ck] = cv
						}
						for ck, cv := range incomingConfig {
							mergedConfig[ck] = cv
						}
						result[k] = mergedConfig

						continue
					}
				}
			}
			result[k] = v
		}
		return result, nil
	}

	err = store.EnvMetadata().Set(context.Background(), updateMetadata, datastore.WithUpdater(customUpdater))
	require.NoError(t, err)

	// Verify the merge
	result, err := store.EnvMetadata().Get(context.Background())
	require.NoError(t, err)

	resultMap, ok := result.Metadata.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "test_domain", resultMap["domain"])         // from original
	require.Equal(t, "test_env", resultMap["environment"])       // from original
	require.InEpsilon(t, float64(2), resultMap["version"], 0.01) // updated
	require.Equal(t, "addedField", resultMap["newField"])        // added

	// Check nested config merge
	configMap, ok := resultMap["config"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(10000), configMap["timeout"]) // updated
	require.Equal(t, float64(3), configMap["retries"])     // preserved from original
	require.Equal(t, "newValue", configMap["newOption"])   // added
}

func TestCatalogEnvMetadataStore_Transactions(t *testing.T) {
	t.Parallel()
	store, closer := setupEnvMetadataTestStore(t)
	defer closer()

	t.Run("transaction rollback", func(t *testing.T) {
		envMetadata := map[string]any{
			"domain":      "test_domain",
			"environment": "test_env",
			"version":     float64(1), // JSON unmarshals numbers as float64
		}

		err := store.WithTransaction(context.Background(), func(ctx context.Context, txStore datastore.BaseCatalogStore) error {
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
		_, err = store.EnvMetadata().Get(context.Background())
		require.Error(t, err)
		require.ErrorIs(t, err, datastore.ErrEnvMetadataNotSet)
	})

	t.Run("transaction commit", func(t *testing.T) {
		envMetadata := map[string]any{
			"domain":      "test_domain",
			"environment": "test_env",
			"version":     float64(1), // JSON unmarshals numbers as float64
		}

		err := store.WithTransaction(context.Background(), func(ctx context.Context, txStore datastore.BaseCatalogStore) error {
			// Set metadata within transaction
			return txStore.EnvMetadata().Set(ctx, envMetadata)
		})
		require.NoError(t, err)

		// Verify metadata exists after commit
		result, err := store.EnvMetadata().Get(context.Background())
		require.NoError(t, err)
		require.Equal(t, envMetadata, result.Metadata)
	})

	t.Run("ignore transactions option", func(t *testing.T) {
		envMetadata := map[string]any{
			"domain":      "test_domain",
			"environment": "test_env",
			"version":     float64(1), // JSON unmarshals numbers as float64
		}

		// Set metadata outside transaction
		err := store.EnvMetadata().Set(context.Background(), envMetadata)
		require.NoError(t, err)

		err = store.WithTransaction(context.Background(), func(ctx context.Context, txStore datastore.BaseCatalogStore) error {
			// Should be able to read with ignore transactions option
			result, getErr := txStore.EnvMetadata().Get(ctx, datastore.IgnoreTransactionsGetOption)
			require.NoError(t, getErr)
			require.Equal(t, envMetadata, result.Metadata)

			// Force rollback
			return errors.New("force rollback")
		})
		require.Error(t, err)

		// Metadata should still exist since it was set outside transaction
		_, err = store.EnvMetadata().Get(context.Background())
		require.NoError(t, err)
	})
}

func TestCatalogEnvMetadataStore_SingleRecord(t *testing.T) {
	t.Parallel()
	store, closer := setupEnvMetadataTestStore(t)
	defer closer()

	// Set initial metadata
	metadata1 := map[string]any{
		"version": float64(1),
		"config":  "initial",
	}
	err := store.EnvMetadata().Set(context.Background(), metadata1)
	require.NoError(t, err)

	result, err := store.EnvMetadata().Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, metadata1, result.Metadata)

	// Set new metadata - should replace, not add
	metadata2 := map[string]any{
		"version": float64(2),
		"config":  "updated",
	}
	err = store.EnvMetadata().Set(context.Background(), metadata2)
	require.NoError(t, err)

	result, err = store.EnvMetadata().Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, metadata2, result.Metadata)

	// There should still be only one record
	// We can't directly test this without accessing the database,
	// but the fact that Get() works correctly indicates single record behavior
}

func TestCatalogEnvMetadataStore_JSONSerialization(t *testing.T) {
	t.Parallel()
	store, closer := setupEnvMetadataTestStore(t)
	defer closer()

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

	err := store.EnvMetadata().Set(context.Background(), complexMetadata)
	require.NoError(t, err)

	result, err := store.EnvMetadata().Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, complexMetadata, result.Metadata)
}
