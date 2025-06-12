package datastore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChainMetadataKey_Equals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key1     ChainMetadataKey
		key2     ChainMetadataKey
		expected bool
	}{
		{
			name:     "Equal keys",
			key1:     NewChainMetadataKey(1),
			key2:     NewChainMetadataKey(1),
			expected: true,
		},
		{
			name:     "Different keys",
			key1:     NewChainMetadataKey(1),
			key2:     NewChainMetadataKey(2),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.expected, tt.key1.Equals(tt.key2))
		})
	}
}

func TestChainMetadataKey(t *testing.T) {
	t.Parallel()

	chainSelector := uint64(1)

	key := NewChainMetadataKey(chainSelector)

	require.Equal(t, chainSelector, key.ChainSelector(), "ChainSelector should return the correct chain selector")
}

func TestChainMetadataKey_String(t *testing.T) {
	t.Parallel()

	key := NewChainMetadataKey(99)
	expected := "99"
	require.Equal(t, expected, key.String())
}
