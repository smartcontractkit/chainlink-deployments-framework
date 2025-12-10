package solana

import (
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddressToBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		address       string
		expectedLen   int
		shouldSucceed bool
		description   string
	}{
		{
			name:          "valid system program address",
			address:       "11111111111111111111111111111112", // System program
			expectedLen:   32,
			shouldSucceed: true,
			description:   "should convert system program address to 32 bytes",
		},
		{
			name:          "valid token program address",
			address:       "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA", // Token program
			expectedLen:   32,
			shouldSucceed: true,
			description:   "should convert token program address to 32 bytes",
		},
		{
			name:          "invalid - non-base58 string",
			address:       "invalid",
			shouldSucceed: false,
			description:   "should reject non-base58 address string",
		},
		{
			name:          "invalid - empty string",
			address:       "",
			shouldSucceed: false,
			description:   "should reject empty address string",
		},
		{
			name:          "invalid - too short",
			address:       "123",
			shouldSucceed: false,
			description:   "should reject address that is too short",
		},
		{
			name:          "invalid - invalid base58 characters",
			address:       "InvalidBase58Characters!",
			shouldSucceed: false,
			description:   "should reject address with invalid base58 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := AddressToBytes(tt.address)

			if tt.shouldSucceed {
				require.NoError(t, err, tt.description)
				assert.Len(t, result, tt.expectedLen, "Expected length %d for address %s", tt.expectedLen, tt.address)
			} else {
				require.Error(t, err, tt.description)
				assert.Nil(t, result, "Expected nil result for invalid address %s", tt.address)
			}
		})
	}

	t.Run("consistent results", func(t *testing.T) {
		t.Parallel()

		address := "11111111111111111111111111111112"

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
		assert.True(t, converter.Supports(chain_selectors.FamilySolana))
		assert.False(t, converter.Supports(chain_selectors.FamilyEVM))
		assert.False(t, converter.Supports(chain_selectors.FamilyAptos))
	})

	t.Run("ConvertToBytes", func(t *testing.T) {
		t.Parallel()

		address := "11111111111111111111111111111112"

		result, err := converter.ConvertToBytes(address)
		require.NoError(t, err)
		assert.Len(t, result, 32)
	})
}
