package aptos

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
			name     string
			address  string
			expected []byte
		}{
			{
				name:    "valid short address",
				address: "0x1",
				expected: func() []byte {
					expected := make([]byte, 32)
					expected[31] = 0x01

					return expected
				}(),
			},
			{
				name:    "valid full address",
				address: "0x0000000000000000000000000000000000000000000000000000000000000001",
				expected: func() []byte {
					expected := make([]byte, 32)
					expected[31] = 0x01

					return expected
				}(),
			},
			{
				name:    "valid address without 0x prefix",
				address: "1",
				expected: func() []byte {
					expected := make([]byte, 32)
					expected[31] = 0x01

					return expected
				}(),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				result, err := AddressToBytes(tt.address)

				require.NoError(t, err)
				assert.Len(t, result, 32)
				assert.Equal(t, tt.expected, result)
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
				name:    "invalid hex characters",
				address: "invalid",
			},
			{
				name:    "empty address",
				address: "",
			},
			{
				name:    "invalid hex with 0x prefix",
				address: "0xGG",
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

	t.Run("address normalization", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			address string
		}{
			{name: "short form", address: "0x1"},
			{name: "padded short form", address: "0x01"},
			{name: "medium padded form", address: "0x0001"},
			{name: "full form", address: "0x0000000000000000000000000000000000000000000000000000000000000001"},
			{name: "no prefix", address: "1"},
		}

		var results [][]byte
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // Cannot use t.Parallel() here because tests share the results slice
				result, err := AddressToBytes(tt.address)
				require.NoError(t, err, "Failed to convert address: %s", tt.address)
				results = append(results, result)
			})
		}

		// All should produce the same result
		t.Run("all forms produce same result", func(t *testing.T) { //nolint:paralleltest // Cannot use t.Parallel() here because it depends on results from previous tests
			for i := 1; i < len(results); i++ {
				assert.Equal(t, results[0], results[i], "All address formats should produce the same result")
			}
		})
	})

	t.Run("consistent results", func(t *testing.T) {
		t.Parallel()

		address := "0x1"

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
		assert.True(t, converter.Supports(chain_selectors.FamilyAptos))
		assert.False(t, converter.Supports(chain_selectors.FamilyEVM))
		assert.False(t, converter.Supports(chain_selectors.FamilySolana))
	})

	t.Run("ConvertToBytes", func(t *testing.T) {
		t.Parallel()

		address := "0x1"
		expected := make([]byte, 32)
		expected[31] = 0x01

		result, err := converter.ConvertToBytes(address)
		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})
}
