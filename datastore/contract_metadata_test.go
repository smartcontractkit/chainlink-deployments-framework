package datastore

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestContractMetadata_Clone(t *testing.T) {
	t.Parallel()

	original := ContractMetadata{
		ChainSelector: 1,
		Address:       "0x123",
		Metadata: TestContractMetadata{
			DeployedAt:  time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
			TxHash:      common.HexToHash("0xabc"),
			BlockNumber: 42,
		},
	}

	cloned, err := original.Clone()
	require.NoError(t, err)

	require.Equal(t, original.ChainSelector, cloned.ChainSelector)
	require.Equal(t, original.Address, cloned.Address)

	typedMeta, err := As[TestContractMetadata](cloned.Metadata)
	require.NoError(t, err)
	require.Equal(t, original.Metadata, typedMeta)

	// Modify the original and ensure the cloned remains unchanged
	original.ChainSelector = 2
	original.Address = "0x456"
	original.Metadata = TestContractMetadata{
		DeployedAt:  time.Date(2025, 2, 3, 4, 5, 6, 0, time.UTC),
		TxHash:      common.HexToHash("0xdef"),
		BlockNumber: 99,
	}

	require.NotEqual(t, original.ChainSelector, cloned.ChainSelector)
	require.NotEqual(t, original.Address, cloned.Address)
	require.NotEqual(t, original.Metadata, cloned.Metadata)
}

func TestContractMetadata_Key(t *testing.T) {
	t.Parallel()

	metadata := ContractMetadata{
		ChainSelector: 1,
		Address:       "0x123",
		Metadata: TestContractMetadata{
			DeployedAt:  time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
			TxHash:      common.HexToHash("0xabc"),
			BlockNumber: 42,
		},
	}

	key := metadata.Key()
	expectedKey := NewContractMetadataKey(1, "0x123")

	require.Equal(t, expectedKey, key)
}
