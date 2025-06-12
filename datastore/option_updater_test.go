package datastore

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func Test_UpdateV2(t *testing.T) {
	// Example usage of the updater with a custom update function
	store := NewMemoryContractMetadataStore()

	// Create a new contract metadata record
	record := ContractMetadata{
		ChainSelector: 11111,
		Address:       "0x1234567890abcdef1234567890abcdef12345678",
		Metadata: SimpleContract{
			DeployedAt:   time.Now(),
			TxHash:       common.HexToHash("0xabcdefabcdefabcdefabcdefabcdefabcdef"),
			BlockNumber:  100,
			LastUpdateAt: time.Now(),
		},
	}

	// Add the record to the store
	store.Records = append(store.Records, record)

	// The record can be updated using a custom updater function
	err := store.UpdateV2(record, WithUpdater(SimpleContractUpdater))
	if err != nil {
		require.NoError(t, err, "UpdateV2 should not return an error with custom updater")
	}

	// The record can also be applied without a custom updater, this will use the identity updater
	// which replaces the record entirely
	err = store.UpdateV2(record)
	if err != nil {
		require.NoError(t, err, "UpdateV2 should not return an error with identity updater")
	}
}

func Test_UpsertV2(t *testing.T) {
	// Example usage of the upsert with a custom update function
	store := NewMemoryContractMetadataStore()

	// Create a new contract metadata record
	record := ContractMetadata{
		ChainSelector: 22222,
		Address:       "0xabcdefabcdefabcdefabcdefabcdefabcdef12345678",
		Metadata: SimpleContract{
			DeployedAt:   time.Now(),
			TxHash:       common.HexToHash("0x1234567890abcdef1234567890abcdef12345678"),
			BlockNumber:  200,
			LastUpdateAt: time.Now(),
		},
	}

	// Upsert the record into the store
	// - if the record does not exist, it will be added
	// - if the record exists, it will be updated using the provided updater function
	err := store.UpsertV2(record, WithUpdater(SimpleContractUpdater))
	if err != nil {
		require.NoError(t, err, "UpsertV2 should not return an error with custom updater")
	}

	// Upsert the record without a custom updater
	// - if the record does not exist, it will be simply added
	// - if the record exists, it will be updated using the identity updater and therefore replaced entirely
	err = store.UpsertV2(record)
	if err != nil {
		require.NoError(t, err, "UpsertV2 should not return an error with identity updater")
	}
}
