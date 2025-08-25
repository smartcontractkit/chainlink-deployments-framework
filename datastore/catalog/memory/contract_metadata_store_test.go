package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

// setupContractMetadataTestStore creates a new memory datastore for testing contract metadata
func setupContractMetadataTestStore(t *testing.T) (*memoryDataStore, func()) {
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

//nolint:paralleltest // Subtests share database instance, cannot run in parallel
func TestCatalogContractMetadataStore_Get(t *testing.T) {
	store, closer := setupContractMetadataTestStore(t)
	defer closer()

	t.Run("not found", func(t *testing.T) {
		key := datastore.NewContractMetadataKey(99999999, "0x1234567890123456789012345678901234567890")
		_, err := store.ContractMetadata().Get(t.Context(), key)
		require.Error(t, err)
		require.ErrorIs(t, err, datastore.ErrContractMetadataNotFound)
	})

	t.Run("success", func(t *testing.T) {
		contractMetadata := newRandomContractMetadata()
		err := store.ContractMetadata().Add(t.Context(), contractMetadata)
		require.NoError(t, err)

		key := datastore.NewContractMetadataKey(contractMetadata.ChainSelector, contractMetadata.Address)
		result, err := store.ContractMetadata().Get(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, key.ChainSelector(), result.ChainSelector)
		require.Equal(t, key.Address(), result.Address)
		require.Equal(t, contractMetadata.Metadata, result.Metadata)
	})

	t.Run("success with nil metadata", func(t *testing.T) {
		contractMetadata := datastore.ContractMetadata{
			ChainSelector: newRandomChainSelector(),
			Address:       newRandomAddress(),
			Metadata:      nil,
		}
		err := store.ContractMetadata().Add(t.Context(), contractMetadata)
		require.NoError(t, err)

		key := datastore.NewContractMetadataKey(contractMetadata.ChainSelector, contractMetadata.Address)
		result, err := store.ContractMetadata().Get(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, key.ChainSelector(), result.ChainSelector)
		require.Equal(t, key.Address(), result.Address)
		require.Nil(t, result.Metadata)
	})
}

//nolint:paralleltest // Subtests share database instance, cannot run in parallel
func TestCatalogContractMetadataStore_Add(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(store *memoryDataStore) datastore.ContractMetadata
		expectError bool
		errorCheck  func(error) bool
	}{
		{
			name: "success",
			setup: func(store *memoryDataStore) datastore.ContractMetadata {
				return newRandomContractMetadata()
			},
			expectError: false,
		},
		{
			name: "success with complex metadata",
			setup: func(store *memoryDataStore) datastore.ContractMetadata {
				return datastore.ContractMetadata{
					ChainSelector: newRandomChainSelector(),
					Address:       newRandomAddress(),
					Metadata: map[string]any{
						"name":        "USDC Token",
						"symbol":      "USDC",
						"decimals":    float64(6), // JSON unmarshals numbers as float64
						"totalSupply": float64(1000000000),
						"features":    []any{"mintable", "burnable", "pausable"},                              // JSON unmarshals arrays as []any
						"config":      map[string]any{"upgradeEnabled": true, "maxSupply": float64(21000000)}, // JSON unmarshals objects as map[string]any
					},
				}
			},
			expectError: false,
		},
		{
			name: "duplicate_error",
			setup: func(store *memoryDataStore) datastore.ContractMetadata {
				// Create and add a record first
				metadata := newRandomContractMetadata()
				err := store.ContractMetadata().Add(t.Context(), metadata)
				require.NoError(t, err)
				// Return the same record to test duplicate
				return metadata
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh store for each test case to avoid concurrency issues
			store, closer := setupContractMetadataTestStore(t)
			defer closer()

			contractMetadata := tt.setup(store)

			// Execute
			err := store.ContractMetadata().Add(t.Context(), contractMetadata)

			// Verify
			if tt.expectError {
				require.Error(t, err)
				if tt.errorCheck != nil {
					require.True(t, tt.errorCheck(err))
				}
			} else {
				require.NoError(t, err)

				// Verify the record was added correctly
				key := datastore.NewContractMetadataKey(contractMetadata.ChainSelector, contractMetadata.Address)
				result, getErr := store.ContractMetadata().Get(t.Context(), key)
				require.NoError(t, getErr)
				require.Equal(t, contractMetadata.ChainSelector, result.ChainSelector)
				require.Equal(t, contractMetadata.Address, result.Address)
				require.Equal(t, contractMetadata.Metadata, result.Metadata)
			}
		})
	}
}

//nolint:paralleltest // Subtests share database instance, cannot run in parallel
func TestCatalogContractMetadataStore_Update(t *testing.T) {
	store, closer := setupContractMetadataTestStore(t)
	defer closer()

	t.Run("not found", func(t *testing.T) {
		key := datastore.NewContractMetadataKey(99999999, "0x1234567890123456789012345678901234567890")
		err := store.ContractMetadata().Update(t.Context(), key, map[string]string{"test": "value"})
		require.Error(t, err)
		require.ErrorIs(t, err, datastore.ErrContractMetadataNotFound)
	})

	t.Run("success", func(t *testing.T) {
		// Add initial record
		contractMetadata := newRandomContractMetadata()
		err := store.ContractMetadata().Add(t.Context(), contractMetadata)
		require.NoError(t, err)

		// Update with new metadata
		key := datastore.NewContractMetadataKey(contractMetadata.ChainSelector, contractMetadata.Address)
		newMetadata := map[string]any{
			"updated": true,
			"version": float64(2), // JSON unmarshals numbers as float64
		}
		err = store.ContractMetadata().Update(t.Context(), key, newMetadata)
		require.NoError(t, err)

		// Verify the update
		result, err := store.ContractMetadata().Get(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, contractMetadata.ChainSelector, result.ChainSelector)
		require.Equal(t, contractMetadata.Address, result.Address)
		require.Equal(t, newMetadata, result.Metadata)
	})

	t.Run("success with custom updater", func(t *testing.T) {
		// Add initial record with map metadata
		initialMetadata := map[string]any{
			"name":    "Test Contract",
			"version": float64(1), // JSON unmarshals numbers as float64
		}
		contractMetadata := datastore.ContractMetadata{
			ChainSelector: newRandomChainSelector(),
			Address:       newRandomAddress(),
			Metadata:      initialMetadata,
		}
		err := store.ContractMetadata().Add(t.Context(), contractMetadata)
		require.NoError(t, err)

		// Update with custom merger that combines maps
		key := datastore.NewContractMetadataKey(contractMetadata.ChainSelector, contractMetadata.Address)
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

		err = store.ContractMetadata().Update(t.Context(), key, updateMetadata, datastore.WithUpdater(customUpdater))
		require.NoError(t, err)

		// Verify the merge
		result, err := store.ContractMetadata().Get(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, contractMetadata.ChainSelector, result.ChainSelector)
		require.Equal(t, contractMetadata.Address, result.Address)

		resultMap, ok := result.Metadata.(map[string]any)
		require.True(t, ok)
		require.Equal(t, "Test Contract", resultMap["name"])         // from original
		require.InEpsilon(t, float64(2), resultMap["version"], 0.01) // updated (JSON numbers are float64)
		require.Equal(t, "newValue", resultMap["newField"])          // added
	})
}

//nolint:paralleltest // Subtests share database instance, cannot run in parallel
func TestCatalogContractMetadataStore_Upsert(t *testing.T) {
	store, closer := setupContractMetadataTestStore(t)
	defer closer()

	t.Run("insert new record", func(t *testing.T) {
		key := datastore.NewContractMetadataKey(newRandomChainSelector(), newRandomAddress())
		metadata := map[string]any{
			"name":     "New Contract",
			"decimals": float64(18), // JSON unmarshals numbers as float64
		}

		err := store.ContractMetadata().Upsert(t.Context(), key, metadata)
		require.NoError(t, err)

		// Verify the record was created
		result, err := store.ContractMetadata().Get(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, key.ChainSelector(), result.ChainSelector)
		require.Equal(t, key.Address(), result.Address)
		require.Equal(t, metadata, result.Metadata)
	})

	t.Run("update existing record", func(t *testing.T) {
		// Add initial record
		contractMetadata := newRandomContractMetadata()
		err := store.ContractMetadata().Add(t.Context(), contractMetadata)
		require.NoError(t, err)

		// Upsert with new metadata
		key := datastore.NewContractMetadataKey(contractMetadata.ChainSelector, contractMetadata.Address)
		newMetadata := map[string]any{
			"updated": true,
			"version": float64(2), // JSON unmarshals numbers as float64
		}
		err = store.ContractMetadata().Upsert(t.Context(), key, newMetadata)
		require.NoError(t, err)

		// Verify the update
		result, err := store.ContractMetadata().Get(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, contractMetadata.ChainSelector, result.ChainSelector)
		require.Equal(t, contractMetadata.Address, result.Address)
		require.Equal(t, newMetadata, result.Metadata)
	})
}

//nolint:paralleltest // Subtests share database instance, cannot run in parallel
func TestCatalogContractMetadataStore_Delete(t *testing.T) {
	store, closer := setupContractMetadataTestStore(t)
	defer closer()

	t.Run("not found", func(t *testing.T) {
		key := datastore.NewContractMetadataKey(99999999, "0x1234567890123456789012345678901234567890")
		err := store.ContractMetadata().Delete(t.Context(), key)
		require.Error(t, err)
		require.ErrorIs(t, err, datastore.ErrContractMetadataNotFound)
	})

	t.Run("success", func(t *testing.T) {
		// Add a record first
		contractMetadata := newRandomContractMetadata()
		err := store.ContractMetadata().Add(t.Context(), contractMetadata)
		require.NoError(t, err)

		// Delete it
		key := datastore.NewContractMetadataKey(contractMetadata.ChainSelector, contractMetadata.Address)
		err = store.ContractMetadata().Delete(t.Context(), key)
		require.NoError(t, err)

		// Verify it's gone
		_, err = store.ContractMetadata().Get(t.Context(), key)
		require.Error(t, err)
		require.ErrorIs(t, err, datastore.ErrContractMetadataNotFound)
	})
}

//nolint:paralleltest // Subtests share database instance, cannot run in parallel
func TestCatalogContractMetadataStore_Fetch(t *testing.T) {
	store, closer := setupContractMetadataTestStore(t)
	defer closer()

	t.Run("empty store", func(t *testing.T) {
		results, err := store.ContractMetadata().Fetch(t.Context())
		require.NoError(t, err)
		require.Empty(t, results)
	})

	t.Run("multiple records", func(t *testing.T) {
		// Add multiple records
		records := []datastore.ContractMetadata{
			newRandomContractMetadata(),
			newRandomContractMetadata(),
			newRandomContractMetadata(),
		}

		for _, record := range records {
			err := store.ContractMetadata().Add(t.Context(), record)
			require.NoError(t, err)
		}

		// Fetch all
		results, err := store.ContractMetadata().Fetch(t.Context())
		require.NoError(t, err)
		require.Len(t, results, len(records))

		// Verify all records are present (order may vary)
		resultMap := make(map[string]datastore.ContractMetadata)
		for _, result := range results {
			key := fmt.Sprintf("%d_%s", result.ChainSelector, result.Address)
			resultMap[key] = result
		}

		for _, expected := range records {
			key := fmt.Sprintf("%d_%s", expected.ChainSelector, expected.Address)
			result, found := resultMap[key]
			require.True(t, found)
			require.Equal(t, expected.ChainSelector, result.ChainSelector)
			require.Equal(t, expected.Address, result.Address)
			require.Equal(t, expected.Metadata, result.Metadata)
		}
	})
}

//nolint:paralleltest // Subtests share database instance, cannot run in parallel
func TestCatalogContractMetadataStore_Filter(t *testing.T) {
	store, closer := setupContractMetadataTestStore(t)
	defer closer()

	// Add test records
	usdcAddress := "0xA0b86a33E6441b8Fa0eBe6C8D96fc6d3E0B74Acb"
	daiAddress := "0x6B175474E89094C44Da98b954EedeAC495271d0F"
	wethAddress := "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"

	records := []datastore.ContractMetadata{
		{
			ChainSelector: 1, // Ethereum
			Address:       usdcAddress,
			Metadata:      map[string]any{"name": "USD Coin", "type": "stablecoin", "decimals": float64(6)},
		},
		{
			ChainSelector: 1, // Ethereum
			Address:       daiAddress,
			Metadata:      map[string]any{"name": "Dai Stablecoin", "type": "stablecoin", "decimals": float64(18)},
		},
		{
			ChainSelector: 137,         // Polygon
			Address:       usdcAddress, // Same address on different chain
			Metadata:      map[string]any{"name": "USD Coin (PoS)", "type": "stablecoin", "decimals": float64(6)},
		},
		{
			ChainSelector: 1, // Ethereum
			Address:       wethAddress,
			Metadata:      map[string]any{"name": "Wrapped Ether", "type": "wrapped", "decimals": float64(18)},
		},
	}

	for _, record := range records {
		err := store.ContractMetadata().Add(t.Context(), record)
		require.NoError(t, err)
	}

	t.Run("no filters", func(t *testing.T) {
		results, err := store.ContractMetadata().Filter(t.Context())
		require.NoError(t, err)
		require.Len(t, results, len(records))
	})

	t.Run("filter by chain selector", func(t *testing.T) {
		filter := func(records []datastore.ContractMetadata) []datastore.ContractMetadata {
			var filtered []datastore.ContractMetadata
			for _, record := range records {
				if record.ChainSelector == 1 { // Ethereum only
					filtered = append(filtered, record)
				}
			}

			return filtered
		}

		results, err := store.ContractMetadata().Filter(t.Context(), filter)
		require.NoError(t, err)
		require.Len(t, results, 3) // USDC, DAI, WETH on Ethereum

		// Verify all results are from Ethereum
		for _, result := range results {
			require.Equal(t, uint64(1), result.ChainSelector)
		}
	})

	t.Run("filter by contract address", func(t *testing.T) {
		filter := func(records []datastore.ContractMetadata) []datastore.ContractMetadata {
			var filtered []datastore.ContractMetadata
			for _, record := range records {
				if record.Address == usdcAddress {
					filtered = append(filtered, record)
				}
			}

			return filtered
		}

		results, err := store.ContractMetadata().Filter(t.Context(), filter)
		require.NoError(t, err)
		require.Len(t, results, 2) // USDC on Ethereum and Polygon

		// Verify all results have the same address
		for _, result := range results {
			require.Equal(t, usdcAddress, result.Address)
		}
	})

	t.Run("filter by metadata field", func(t *testing.T) {
		filter := func(records []datastore.ContractMetadata) []datastore.ContractMetadata {
			var filtered []datastore.ContractMetadata
			for _, record := range records {
				if metadata, ok := record.Metadata.(map[string]any); ok {
					if tokenType, exists := metadata["type"]; exists && tokenType == "stablecoin" {
						filtered = append(filtered, record)
					}
				}
			}

			return filtered
		}

		results, err := store.ContractMetadata().Filter(t.Context(), filter)
		require.NoError(t, err)
		require.Len(t, results, 3) // USDC (2x) and DAI

		// Verify all results are stablecoins
		for _, result := range results {
			metadata, ok := result.Metadata.(map[string]any)
			require.True(t, ok)
			require.Equal(t, "stablecoin", metadata["type"])
		}
	})

	t.Run("multiple filters", func(t *testing.T) {
		// First filter: only Ethereum
		ethereumFilter := func(records []datastore.ContractMetadata) []datastore.ContractMetadata {
			var filtered []datastore.ContractMetadata
			for _, record := range records {
				if record.ChainSelector == 1 {
					filtered = append(filtered, record)
				}
			}

			return filtered
		}

		// Second filter: only stablecoins
		stablecoinFilter := func(records []datastore.ContractMetadata) []datastore.ContractMetadata {
			var filtered []datastore.ContractMetadata
			for _, record := range records {
				if metadata, ok := record.Metadata.(map[string]any); ok {
					if tokenType, exists := metadata["type"]; exists && tokenType == "stablecoin" {
						filtered = append(filtered, record)
					}
				}
			}

			return filtered
		}

		results, err := store.ContractMetadata().Filter(t.Context(), ethereumFilter, stablecoinFilter)
		require.NoError(t, err)
		require.Len(t, results, 2) // USDC and DAI on Ethereum

		// Verify all results are stablecoins on Ethereum
		for _, result := range results {
			require.Equal(t, uint64(1), result.ChainSelector)
			metadata, ok := result.Metadata.(map[string]any)
			require.True(t, ok)
			require.Equal(t, "stablecoin", metadata["type"])
		}
	})
}

//nolint:paralleltest // Subtests share database instance, cannot run in parallel
func TestCatalogContractMetadataStore_Transactions(t *testing.T) {
	store, closer := setupContractMetadataTestStore(t)
	defer closer()

	t.Run("transaction rollback", func(t *testing.T) {
		contractMetadata := newRandomContractMetadata()

		err := store.WithTransaction(t.Context(), func(ctx context.Context, txStore datastore.BaseCatalogStore) error {
			// Add record within transaction
			addErr := txStore.ContractMetadata().Add(ctx, contractMetadata)
			require.NoError(t, addErr)

			// Verify it exists within transaction
			key := datastore.NewContractMetadataKey(contractMetadata.ChainSelector, contractMetadata.Address)
			_, getErr := txStore.ContractMetadata().Get(ctx, key)
			require.NoError(t, getErr)

			// Force rollback by returning error
			return errors.New("force rollback")
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "force rollback")

		// Verify record doesn't exist after rollback
		key := datastore.NewContractMetadataKey(contractMetadata.ChainSelector, contractMetadata.Address)
		_, err = store.ContractMetadata().Get(t.Context(), key)
		require.Error(t, err)
		require.ErrorIs(t, err, datastore.ErrContractMetadataNotFound)
	})

	t.Run("transaction commit", func(t *testing.T) {
		contractMetadata := newRandomContractMetadata()

		err := store.WithTransaction(t.Context(), func(ctx context.Context, txStore datastore.BaseCatalogStore) error {
			// Add record within transaction
			return txStore.ContractMetadata().Add(ctx, contractMetadata)
		})
		require.NoError(t, err)

		// Verify record exists after commit
		key := datastore.NewContractMetadataKey(contractMetadata.ChainSelector, contractMetadata.Address)
		result, err := store.ContractMetadata().Get(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, contractMetadata.ChainSelector, result.ChainSelector)
		require.Equal(t, contractMetadata.Address, result.Address)
		require.Equal(t, contractMetadata.Metadata, result.Metadata)
	})

	t.Run("ignore transactions option", func(t *testing.T) {
		contractMetadata := newRandomContractMetadata()

		// Add record outside transaction
		err := store.ContractMetadata().Add(t.Context(), contractMetadata)
		require.NoError(t, err)

		err = store.WithTransaction(t.Context(), func(ctx context.Context, txStore datastore.BaseCatalogStore) error {
			key := datastore.NewContractMetadataKey(contractMetadata.ChainSelector, contractMetadata.Address)

			// Should be able to read with ignore transactions option
			result, getErr := txStore.ContractMetadata().Get(ctx, key, datastore.IgnoreTransactionsGetOption)
			require.NoError(t, getErr)
			require.Equal(t, contractMetadata.ChainSelector, result.ChainSelector)
			require.Equal(t, contractMetadata.Address, result.Address)

			// Force rollback
			return errors.New("force rollback")
		})
		require.Error(t, err)

		// Record should still exist since it was added outside transaction
		key := datastore.NewContractMetadataKey(contractMetadata.ChainSelector, contractMetadata.Address)
		_, err = store.ContractMetadata().Get(t.Context(), key)
		require.NoError(t, err)
	})
}

// Helper functions

func newRandomAddress() string {
	// Generate a random hex address (20 bytes = 40 hex characters)
	bytes := make([]byte, 20)
	_, _ = rand.Read(bytes)

	return "0x" + hex.EncodeToString(bytes)
}

func newRandomContractMetadata() datastore.ContractMetadata {
	// Generate random hex string for metadata
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	randomData := hex.EncodeToString(bytes)

	return datastore.ContractMetadata{
		ChainSelector: newRandomChainSelector(),
		Address:       newRandomAddress(),
		Metadata: map[string]any{
			"name":        "Test Contract " + randomData[:8],
			"description": "A test contract with random data: " + randomData,
			"version":     float64(1), // JSON unmarshals numbers as float64
			"active":      true,
		},
	}
}
