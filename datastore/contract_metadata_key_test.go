package datastore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContractMetadataKey_Equals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key1     ContractMetadataKey
		key2     ContractMetadataKey
		expected bool
	}{
		{
			name:     "Equal keys",
			key1:     NewContractMetadataKey(1, "0x1234567890abcdef"),
			key2:     NewContractMetadataKey(1, "0x1234567890abcdef"),
			expected: true,
		},
		{
			name:     "Different chain selector",
			key1:     NewContractMetadataKey(1, "0x1234567890abcdef"),
			key2:     NewContractMetadataKey(2, "0x1234567890abcdef"),
			expected: false,
		},
		{
			name:     "Different address",
			key1:     NewContractMetadataKey(1, "0x1234567890abcdef"),
			key2:     NewContractMetadataKey(1, "0xabcdef1234567890"),
			expected: false,
		},
		{
			name:     "Completely different keys",
			key1:     NewContractMetadataKey(1, "0x1234567890abcdef"),
			key2:     NewContractMetadataKey(2, "0xabcdef1234567890"),
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

func TestContractMetadataKey(t *testing.T) {
	t.Parallel()

	chainSelector := uint64(1)
	address := "0x1234567890abcdef"

	key := NewContractMetadataKey(chainSelector, address)

	require.Equal(t, chainSelector, key.ChainSelector(), "ChainSelector should return the correct chain selector")
	require.Equal(t, address, key.Address(), "Address should return the correct address")
}

func TestContractMetadataKey_String(t *testing.T) {
	t.Parallel()

	key := NewContractMetadataKey(99, "0xabc")
	expected := "99_0xabc"
	require.Equal(t, expected, key.String())
}

func TestNewContractMetadataKeyFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    string
		want    ContractMetadataKey
		wantErr string
	}{
		{
			name: "success: valid string",
			give: "42_0x1234567890abcdef",
			want: NewContractMetadataKey(42, "0x1234567890abcdef"),
		},
		{
			name:    "failure: too few parts",
			give:    "42",
			wantErr: "invalid contract metadata key",
		},
		{
			name:    "failure: too many parts",
			give:    "42_0xabc_extra",
			wantErr: "invalid contract metadata key",
		},
		{
			name:    "failure: invalid chain selector",
			give:    "notanumber_0x1234567890abcdef",
			wantErr: "failed to parse chain selector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewContractMetadataKeyFromString(tt.give)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want.ChainSelector(), got.ChainSelector())
			require.Equal(t, tt.want.Address(), got.Address())
		})
	}
}
