package datastore

import (
	"testing"

	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"
)

func TestContractMetadata_Clone(t *testing.T) {
	t.Parallel()

	original := ContractMetadata{
		ChainSelector: 1,
		Address:       "0x123",
		Metadata: testMetadata{
			Field:         "test field",
			ChainSelector: chainsel.APTOS_MAINNET.Selector,
		},
	}

	cloned, err := original.Clone()
	require.NoError(t, err, "Clone should not return an error")

	require.Equal(t, original.ChainSelector, cloned.ChainSelector)
	require.Equal(t, original.Address, cloned.Address)

	concrete, err := As[testMetadata](cloned.Metadata)
	require.NoError(t, err, "As should not return an error for CustomMetadata")
	require.Equal(t, original.Metadata, concrete)

	// Modify the original and ensure the cloned remains unchanged
	original.ChainSelector = 2
	original.Address = "0x456"
	original.Metadata = testMetadata{
		Field:         "updated field",
		ChainSelector: chainsel.APTOS_MAINNET.Selector,
	}

	require.NotEqual(t, original.ChainSelector, cloned.ChainSelector)
	require.NotEqual(t, original.Address, cloned.Address)

	concrete, err = As[testMetadata](cloned.Metadata)
	require.NoError(t, err, "As should not return an error for CustomMetadata")
	require.NotEqual(t, original.Metadata, concrete, "Cloned metadata should not be equal to modified original")
}

func TestContractMetadata_Key(t *testing.T) {
	t.Parallel()

	metadata := ContractMetadata{
		ChainSelector: 1,
		Address:       "0x123",
		Metadata:      testMetadata{Field: "test data", ChainSelector: 0},
	}

	key := metadata.Key()
	expectedKey := NewContractMetadataKey(1, "0x123")

	require.Equal(t, expectedKey, key)
}
