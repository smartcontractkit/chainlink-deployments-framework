package aptos_test

import (
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
)

func TestChain_ChainInfot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		selector   uint64
		wantName   string
		wantString string
		wantFamily string
	}{
		{
			name:       "returns correct info",
			selector:   chain_selectors.APTOS_MAINNET.Selector,
			wantString: "aptos-mainnet (4741433654826277614)",
			wantName:   chain_selectors.APTOS_MAINNET.Name,
			wantFamily: chain_selectors.FamilyAptos,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := aptos.Chain{
				Selector: tt.selector,
			}
			assert.Equal(t, tt.selector, c.ChainSelector())
			assert.Equal(t, tt.wantString, c.String())
			assert.Equal(t, tt.wantName, c.Name())
			assert.Equal(t, tt.wantFamily, c.Family())
		})
	}
}

func TestChain_AddressToBytes(t *testing.T) {
	t.Parallel()

	chain := aptos.Chain{Selector: 4741444398633441}

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

				result, err := chain.AddressToBytes(tt.address)

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

				result, err := chain.AddressToBytes(tt.address)
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
				result, err := chain.AddressToBytes(tt.address)
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

		result1, err1 := chain.AddressToBytes(address)
		require.NoError(t, err1)

		result2, err2 := chain.AddressToBytes(address)
		require.NoError(t, err2)

		assert.Equal(t, result1, result2, "Expected consistent results for the same address")
	})
}
