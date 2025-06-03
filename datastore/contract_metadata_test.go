package datastore

import (
	"testing"

	"github.com/stretchr/testify/require"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
)

func TestContractMetadata_Clone(t *testing.T) {
	t.Parallel()

	original := ContractMetadata{
		ChainSelector: 1,
		Address:       "0x123",
		Metadata: CustomMetadata{
			Field:         "test field",
			ChainSelector: chain_selectors.APTOS_MAINNET.Selector,
		},
	}

	cloned, err := original.Clone()
	require.NoError(t, err, "Clone should not return an error")

	require.Equal(t, original.ChainSelector, cloned.ChainSelector)
	require.Equal(t, original.Address, cloned.Address)

	concrete, err := As[CustomMetadata](cloned.Metadata)
	require.NoError(t, err, "As should not return an error for CustomMetadata")
	require.Equal(t, original.Metadata, concrete)

	// Modify the original and ensure the cloned remains unchanged
	original.ChainSelector = 2
	original.Address = "0x456"
	original.Metadata = CustomMetadata{
		Field:         "updated field",
		ChainSelector: chain_selectors.APTOS_MAINNET.Selector,
	}

	require.NotEqual(t, original.ChainSelector, cloned.ChainSelector)
	require.NotEqual(t, original.Address, cloned.Address)

	concrete, err = As[CustomMetadata](cloned.Metadata)
	require.NoError(t, err, "As should not return an error for CustomMetadata")
	require.NotEqual(t, original.Metadata, concrete, "Cloned metadata should not be equal to modified original")
}

func TestContractMetadata_Key(t *testing.T) {
	t.Parallel()

	metadata := ContractMetadata{
		ChainSelector: 1,
		Address:       "0x123",
		Metadata:      DefaultMetadata{Data: "test data"},
	}

	key := metadata.Key()
	expectedKey := NewContractMetadataKey(1, "0x123")

	require.Equal(t, expectedKey, key)
}
