package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

// setupChainMetadataTestStore creates a new memory datastore for testing chain metadata
func setupChainMetadataTestStore(t *testing.T) *memoryCatalogDataStore {
	t.Helper()
	store, err := NewMemoryCatalogDataStore()
	require.NoError(t, err)

	return store
}

func TestCatalogChainMetadataStore_Get(t *testing.T) {
	t.Parallel()

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		store := setupChainMetadataTestStore(t)

		key := datastore.NewChainMetadataKey(99999999)
		_, err := store.ChainMetadata().Get(t.Context(), key)
		require.Error(t, err)
		require.ErrorIs(t, err, datastore.ErrChainMetadataNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		store := setupChainMetadataTestStore(t)

		chainMetadata := newRandomChainMetadata()
		err := store.ChainMetadata().Add(t.Context(), chainMetadata)
		require.NoError(t, err)

		key := datastore.NewChainMetadataKey(chainMetadata.ChainSelector)
		result, err := store.ChainMetadata().Get(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, key.ChainSelector(), result.ChainSelector)
		require.Equal(t, chainMetadata.Metadata, result.Metadata)
	})

	t.Run("success with nil metadata", func(t *testing.T) {
		t.Parallel()
		store := setupChainMetadataTestStore(t)

		chainMetadata := datastore.ChainMetadata{
			ChainSelector: newRandomChainSelector(),
			Metadata:      nil,
		}
		err := store.ChainMetadata().Add(t.Context(), chainMetadata)
		require.NoError(t, err)

		key := datastore.NewChainMetadataKey(chainMetadata.ChainSelector)
		result, err := store.ChainMetadata().Get(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, key.ChainSelector(), result.ChainSelector)
		require.Nil(t, result.Metadata)
	})
}

func TestCatalogChainMetadataStore_Add(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(store *memoryCatalogDataStore) datastore.ChainMetadata
		expectError bool
		errorCheck  func(error) bool
	}{
		{
			name: "success",
			setup: func(store *memoryCatalogDataStore) datastore.ChainMetadata {
				return newRandomChainMetadata()
			},
			expectError: false,
		},
		{
			name: "success with complex metadata",
			setup: func(store *memoryCatalogDataStore) datastore.ChainMetadata {
				return datastore.ChainMetadata{
					ChainSelector: newRandomChainSelector(),
					Metadata: map[string]any{
						"name":     "Ethereum Mainnet",
						"chainId":  float64(1),                                                     // JSON unmarshals numbers as float64
						"rpcUrls":  []any{"https://mainnet.infura.io", "https://eth.llamarpc.com"}, // JSON unmarshals arrays as []any
						"features": map[string]any{"eip1559": true, "london": true},                // JSON unmarshals objects as map[string]any
					},
				}
			},
			expectError: false,
		},
		{
			name: "duplicate_error",
			setup: func(store *memoryCatalogDataStore) datastore.ChainMetadata {
				// Create and add a record first
				metadata := newRandomChainMetadata()
				err := store.ChainMetadata().Add(t.Context(), metadata)
				require.NoError(t, err)
				// Return the same record to test duplicate
				return metadata
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := setupChainMetadataTestStore(t)

			chainMetadata := tt.setup(store)

			// Execute
			err := store.ChainMetadata().Add(t.Context(), chainMetadata)

			// Verify
			if tt.expectError {
				require.Error(t, err)
				if tt.errorCheck != nil {
					require.True(t, tt.errorCheck(err))
				}
			} else {
				require.NoError(t, err)

				// Verify the record was added correctly
				key := datastore.NewChainMetadataKey(chainMetadata.ChainSelector)
				result, getErr := store.ChainMetadata().Get(t.Context(), key)
				require.NoError(t, getErr)
				require.Equal(t, chainMetadata.ChainSelector, result.ChainSelector)
				require.Equal(t, chainMetadata.Metadata, result.Metadata)
			}
		})
	}
}

func TestCatalogChainMetadataStore_Update(t *testing.T) {
	t.Parallel()

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		store := setupChainMetadataTestStore(t)
		key := datastore.NewChainMetadataKey(99999999)
		err := store.ChainMetadata().Update(t.Context(), key, map[string]string{"test": "value"})
		require.Error(t, err)
		require.ErrorIs(t, err, datastore.ErrChainMetadataNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		store := setupChainMetadataTestStore(t)
		// Add initial record
		chainMetadata := newRandomChainMetadata()
		err := store.ChainMetadata().Add(t.Context(), chainMetadata)
		require.NoError(t, err)

		// Update with new metadata
		key := datastore.NewChainMetadataKey(chainMetadata.ChainSelector)
		newMetadata := map[string]any{
			"updated": true,
			"version": float64(2), // JSON unmarshals numbers as float64
		}
		err = store.ChainMetadata().Update(t.Context(), key, newMetadata)
		require.NoError(t, err)

		// Verify the update
		result, err := store.ChainMetadata().Get(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, chainMetadata.ChainSelector, result.ChainSelector)
		require.Equal(t, newMetadata, result.Metadata)
	})

	t.Run("success with custom updater", func(t *testing.T) {
		t.Parallel()
		store := setupChainMetadataTestStore(t)
		// Add initial record with map metadata
		initialMetadata := map[string]any{
			"name":    "Test Chain",
			"version": float64(1), // JSON unmarshals numbers as float64
		}
		chainMetadata := datastore.ChainMetadata{
			ChainSelector: newRandomChainSelector(),
			Metadata:      initialMetadata,
		}
		err := store.ChainMetadata().Add(t.Context(), chainMetadata)
		require.NoError(t, err)

		// Update with custom merger that combines maps
		key := datastore.NewChainMetadataKey(chainMetadata.ChainSelector)
		updateMetadata := map[string]any{
			"version":  float64(2), // JSON unmarshals numbers as float64
			"newField": "newValue",
		}

		customUpdater := func(latest any, incoming any) (any, error) {
			latestMap, ok1 := latest.(map[string]any)
			incomingMap, ok2 := incoming.(map[string]any)
			if !ok1 || !ok2 {
				return incoming, nil // fallback to replacement
			}

			// Merge maps
			result := make(map[string]any)
			for k, v := range latestMap {
				result[k] = v
			}
			for k, v := range incomingMap {
				result[k] = v
			}

			return result, nil
		}

		err = store.ChainMetadata().Update(t.Context(), key, updateMetadata, datastore.WithUpdater(customUpdater))
		require.NoError(t, err)

		// Verify the merge
		result, err := store.ChainMetadata().Get(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, chainMetadata.ChainSelector, result.ChainSelector)

		resultMap, ok := result.Metadata.(map[string]any)
		require.True(t, ok)
		require.Equal(t, "Test Chain", resultMap["name"])            // from original
		require.InEpsilon(t, float64(2), resultMap["version"], 0.01) // updated (JSON numbers are float64)
		require.Equal(t, "newValue", resultMap["newField"])          // added
	})
}

func TestCatalogChainMetadataStore_Upsert(t *testing.T) {
	t.Parallel()

	t.Run("insert new record", func(t *testing.T) {
		t.Parallel()
		store := setupChainMetadataTestStore(t)
		key := datastore.NewChainMetadataKey(newRandomChainSelector())
		metadata := map[string]any{
			"name": "New Chain",
			"id":   float64(123), // JSON unmarshals numbers as float64
		}

		err := store.ChainMetadata().Upsert(t.Context(), key, metadata)
		require.NoError(t, err)

		// Verify the record was created
		result, err := store.ChainMetadata().Get(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, key.ChainSelector(), result.ChainSelector)
		require.Equal(t, metadata, result.Metadata)
	})

	t.Run("update existing record", func(t *testing.T) {
		t.Parallel()
		store := setupChainMetadataTestStore(t)
		// Add initial record
		chainMetadata := newRandomChainMetadata()
		err := store.ChainMetadata().Add(t.Context(), chainMetadata)
		require.NoError(t, err)

		// Upsert with new metadata
		key := datastore.NewChainMetadataKey(chainMetadata.ChainSelector)
		newMetadata := map[string]any{
			"updated": true,
			"version": float64(2), // JSON unmarshals numbers as float64
		}
		err = store.ChainMetadata().Upsert(t.Context(), key, newMetadata)
		require.NoError(t, err)

		// Verify the update
		result, err := store.ChainMetadata().Get(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, chainMetadata.ChainSelector, result.ChainSelector)
		require.Equal(t, newMetadata, result.Metadata)
	})
}

func TestCatalogChainMetadataStore_Delete(t *testing.T) {
	t.Parallel()
	store := setupChainMetadataTestStore(t)

	key := datastore.NewChainMetadataKey(12345)

	// Execute
	err := store.ChainMetadata().Delete(t.Context(), key)

	// Verify
	require.Error(t, err)
	require.Contains(t, err.Error(), "delete operation not supported")
}

func TestCatalogChainMetadataStore_Fetch(t *testing.T) {
	t.Parallel()

	t.Run("empty store", func(t *testing.T) {
		t.Parallel()
		store := setupChainMetadataTestStore(t)
		results, err := store.ChainMetadata().Fetch(t.Context())
		require.NoError(t, err)
		require.Empty(t, results)
	})

	t.Run("multiple records", func(t *testing.T) {
		t.Parallel()
		store := setupChainMetadataTestStore(t)
		// Add multiple records
		records := []datastore.ChainMetadata{
			newRandomChainMetadata(),
			newRandomChainMetadata(),
			newRandomChainMetadata(),
		}

		for _, record := range records {
			err := store.ChainMetadata().Add(t.Context(), record)
			require.NoError(t, err)
		}

		// Fetch all
		results, err := store.ChainMetadata().Fetch(t.Context())
		require.NoError(t, err)
		require.Len(t, results, len(records))

		// Verify all records are present (order may vary)
		resultMap := make(map[uint64]datastore.ChainMetadata)
		for _, result := range results {
			resultMap[result.ChainSelector] = result
		}

		for _, expected := range records {
			result, found := resultMap[expected.ChainSelector]
			require.True(t, found)
			require.Equal(t, expected.ChainSelector, result.ChainSelector)
			require.Equal(t, expected.Metadata, result.Metadata)
		}
	})
}

func TestCatalogChainMetadataStore_Filter(t *testing.T) {
	t.Parallel()
	store := setupChainMetadataTestStore(t)

	// Add test records
	records := []datastore.ChainMetadata{
		{
			ChainSelector: 1,
			Metadata:      map[string]any{"name": "Ethereum", "type": "mainnet"},
		},
		{
			ChainSelector: 137,
			Metadata:      map[string]any{"name": "Polygon", "type": "mainnet"},
		},
		{
			ChainSelector: 80001,
			Metadata:      map[string]any{"name": "Mumbai", "type": "testnet"},
		},
	}

	for _, record := range records {
		err := store.ChainMetadata().Add(t.Context(), record)
		require.NoError(t, err)
	}

	t.Run("no filters", func(t *testing.T) {
		t.Parallel()
		results, err := store.ChainMetadata().Filter(t.Context())
		require.NoError(t, err)
		require.Len(t, results, len(records))
	})

	t.Run("filter by chain selector", func(t *testing.T) {
		t.Parallel()
		filter := func(records []datastore.ChainMetadata) []datastore.ChainMetadata {
			var filtered []datastore.ChainMetadata
			for _, record := range records {
				if record.ChainSelector == 137 {
					filtered = append(filtered, record)
				}
			}

			return filtered
		}

		results, err := store.ChainMetadata().Filter(t.Context(), filter)
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, uint64(137), results[0].ChainSelector)
	})

	t.Run("filter by metadata field", func(t *testing.T) {
		t.Parallel()
		filter := func(records []datastore.ChainMetadata) []datastore.ChainMetadata {
			var filtered []datastore.ChainMetadata
			for _, record := range records {
				if metadata, ok := record.Metadata.(map[string]any); ok {
					if networkType, exists := metadata["type"]; exists && networkType == "mainnet" {
						filtered = append(filtered, record)
					}
				}
			}

			return filtered
		}

		results, err := store.ChainMetadata().Filter(t.Context(), filter)
		require.NoError(t, err)
		require.Len(t, results, 2) // Ethereum and Polygon

		chainSelectors := make(map[uint64]bool)
		for _, result := range results {
			chainSelectors[result.ChainSelector] = true
		}
		require.True(t, chainSelectors[1])      // Ethereum
		require.True(t, chainSelectors[137])    // Polygon
		require.False(t, chainSelectors[80001]) // Mumbai should be filtered out
	})

	t.Run("multiple filters", func(t *testing.T) {
		t.Parallel()
		// First filter: only mainnet
		mainnetFilter := func(records []datastore.ChainMetadata) []datastore.ChainMetadata {
			var filtered []datastore.ChainMetadata
			for _, record := range records {
				if metadata, ok := record.Metadata.(map[string]any); ok {
					if networkType, exists := metadata["type"]; exists && networkType == "mainnet" {
						filtered = append(filtered, record)
					}
				}
			}

			return filtered
		}

		// Second filter: only Ethereum
		ethereumFilter := func(records []datastore.ChainMetadata) []datastore.ChainMetadata {
			var filtered []datastore.ChainMetadata
			for _, record := range records {
				if record.ChainSelector == 1 {
					filtered = append(filtered, record)
				}
			}

			return filtered
		}

		results, err := store.ChainMetadata().Filter(t.Context(), mainnetFilter, ethereumFilter)
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, uint64(1), results[0].ChainSelector)
	})
}

func TestCatalogChainMetadataStore_Transactions(t *testing.T) {
	t.Parallel()

	t.Run("transaction rollback", func(t *testing.T) {
		t.Parallel()
		store := setupChainMetadataTestStore(t)
		chainMetadata := newRandomChainMetadata()

		err := store.WithTransaction(t.Context(), func(ctx context.Context, txStore datastore.BaseCatalogStore) error {
			// Add record within transaction
			addErr := txStore.ChainMetadata().Add(ctx, chainMetadata)
			require.NoError(t, addErr)

			// Verify it exists within transaction
			key := datastore.NewChainMetadataKey(chainMetadata.ChainSelector)
			_, getErr := txStore.ChainMetadata().Get(ctx, key)
			require.NoError(t, getErr)

			// Force rollback by returning error
			return errors.New("force rollback")
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "force rollback")

		// Verify record doesn't exist after rollback
		key := datastore.NewChainMetadataKey(chainMetadata.ChainSelector)
		_, err = store.ChainMetadata().Get(t.Context(), key)
		require.Error(t, err)
		require.ErrorIs(t, err, datastore.ErrChainMetadataNotFound)
	})

	t.Run("transaction commit", func(t *testing.T) {
		t.Parallel()
		store := setupChainMetadataTestStore(t)
		chainMetadata := newRandomChainMetadata()

		err := store.WithTransaction(t.Context(), func(ctx context.Context, txStore datastore.BaseCatalogStore) error {
			// Add record within transaction
			return txStore.ChainMetadata().Add(ctx, chainMetadata)
		})
		require.NoError(t, err)

		// Verify record exists after commit
		key := datastore.NewChainMetadataKey(chainMetadata.ChainSelector)
		result, err := store.ChainMetadata().Get(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, chainMetadata.ChainSelector, result.ChainSelector)
		require.Equal(t, chainMetadata.Metadata, result.Metadata)
	})

	t.Run("ignore transactions option", func(t *testing.T) {
		t.Parallel()
		store := setupChainMetadataTestStore(t)
		chainMetadata := newRandomChainMetadata()

		// Add record outside transaction
		err := store.ChainMetadata().Add(t.Context(), chainMetadata)
		require.NoError(t, err)

		err = store.WithTransaction(t.Context(), func(ctx context.Context, txStore datastore.BaseCatalogStore) error {
			key := datastore.NewChainMetadataKey(chainMetadata.ChainSelector)

			// Should be able to read with ignore transactions option
			result, getErr := txStore.ChainMetadata().Get(ctx, key, datastore.IgnoreTransactionsGetOption)
			require.NoError(t, getErr)
			require.Equal(t, chainMetadata.ChainSelector, result.ChainSelector)

			// Force rollback
			return errors.New("force rollback")
		})
		require.Error(t, err)

		// Record should still exist since it was added outside transaction
		key := datastore.NewChainMetadataKey(chainMetadata.ChainSelector)
		_, err = store.ChainMetadata().Get(t.Context(), key)
		require.NoError(t, err)
	})
}

// Helper functions

func newRandomChainSelector() uint64 {
	// Generate a random uint64 for chain selector
	maxVal := big.NewInt(0).SetUint64(^uint64(0) >> 1) // Max int64 to avoid overflow issues
	n, _ := rand.Int(rand.Reader, maxVal)

	return n.Uint64()
}

func newRandomChainMetadata() datastore.ChainMetadata {
	// Generate random hex string for metadata
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	randomData := hex.EncodeToString(bytes)

	return datastore.ChainMetadata{
		ChainSelector: newRandomChainSelector(),
		Metadata: map[string]any{
			"name":        "Test Chain " + randomData[:8],
			"description": "A test chain with random data: " + randomData,
			"version":     float64(1), // JSON unmarshals numbers as float64
			"active":      true,
		},
	}
}
