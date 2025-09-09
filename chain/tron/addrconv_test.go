package tron

import (
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddressToBytes(t *testing.T) {
	t.Parallel()

	t.Run("valid addresses", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name        string
			address     string
			description string
		}{
			{
				name:        "valid TRON address",
				address:     "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH",
				description: "Standard TRON address in base58 format",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				result, err := AddressToBytes(tt.address)

				require.NoError(t, err, "Should successfully parse valid TRON address: %s (%s)", tt.address, tt.description)
				assert.Len(t, result, 21, "TRON address should produce 21 bytes")
				assert.NotNil(t, result)
			})
		}
	})

	t.Run("invalid addresses", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			address string
		}{
			{
				name:    "invalid - non-base58 string",
				address: "invalid",
			},
			{
				name:    "invalid - empty string",
				address: "",
			},
			{
				name:    "invalid - wrong format",
				address: "0x742d35Cc6634C0532925a3b8D4c8C1B8c4c8C1B8",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				result, err := AddressToBytes(tt.address)
				require.Error(t, err, "Expected error for address: %s", tt.address)
				assert.Nil(t, result)
			})
		}
	})

	t.Run("consistent results", func(t *testing.T) {
		t.Parallel()

		address := "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"

		result1, err1 := AddressToBytes(address)
		require.NoError(t, err1)

		result2, err2 := AddressToBytes(address)
		require.NoError(t, err2)

		assert.Equal(t, result1, result2, "Expected consistent results for the same address")
	})
}

func TestAddressConverter(t *testing.T) {
	t.Parallel()

	converter := AddressConverter{}

	t.Run("Supports", func(t *testing.T) {
		t.Parallel()
		assert.True(t, converter.Supports(chain_selectors.FamilyTron))
		assert.False(t, converter.Supports(chain_selectors.FamilyEVM))
		assert.False(t, converter.Supports(chain_selectors.FamilySolana))
	})

	t.Run("ConvertToBytes", func(t *testing.T) {
		t.Parallel()

		address := "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"

		result, err := converter.ConvertToBytes(address)
		require.NoError(t, err)
		assert.Len(t, result, 21)
	})
}
