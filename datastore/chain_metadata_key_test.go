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

func TestChainMetadataKey_ChainSelector(t *testing.T) {
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

func TestNewChainMetadataKeyFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    string
		want    ChainMetadataKey
		wantErr string
	}{
		{
			name: "success: valid string",
			give: "5009297550715157269",
			want: NewChainMetadataKey(5009297550715157269),
		},
		{
			name:    "failure: empty string",
			give:    "",
			wantErr: "invalid chain metadata key",
		},
		{
			name:    "failure: invalid chain selector",
			give:    "notanumber",
			wantErr: "failed to parse chain selector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewChainMetadataKeyFromString(tt.give)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want.ChainSelector(), got.ChainSelector())
		})
	}
}
