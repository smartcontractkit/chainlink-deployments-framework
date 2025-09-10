package sui

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
			name    string
			address string
		}{
			{
				name:    "valid address with 0x prefix",
				address: "0xa402ce953053607dffcdfec89406c579c8d8ddb9c90e01b7aa28f5f1538ac289",
			},
			{
				name:    "valid address without 0x prefix",
				address: "a402ce953053607dffcdfec89406c579c8d8ddb9c90e01b7aa28f5f1538ac289",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				result, err := AddressToBytes(tt.address)

				require.NoError(t, err)
				assert.Len(t, result, 32)
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
				name:    "invalid - too short",
				address: "0x123",
			},
			{
				name:    "invalid - too long",
				address: "0xa402ce953053607dffcdfec89406c579c8d8ddb9c90e01b7aa28f5f1538ac289aa",
			},
			{
				name:    "invalid - non-hex characters",
				address: "invalid",
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

		address := "0xa402ce953053607dffcdfec89406c579c8d8ddb9c90e01b7aa28f5f1538ac289"

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
		assert.True(t, converter.Supports(chain_selectors.FamilySui))
		assert.False(t, converter.Supports(chain_selectors.FamilyEVM))
		assert.False(t, converter.Supports(chain_selectors.FamilySolana))
	})

	t.Run("ConvertToBytes", func(t *testing.T) {
		t.Parallel()

		address := "0xa402ce953053607dffcdfec89406c579c8d8ddb9c90e01b7aa28f5f1538ac289"

		result, err := converter.ConvertToBytes(address)
		require.NoError(t, err)
		assert.Len(t, result, 32)
	})
}
