package sui_test

import (
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/sui"
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
			selector:   chain_selectors.SUI_MAINNET.Selector,
			wantString: "sui-mainnet (17529533435026248318)",
			wantName:   chain_selectors.SUI_MAINNET.Name,
			wantFamily: chain_selectors.FamilySui,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := sui.Chain{
				ChainMetadata: sui.ChainMetadata{Selector: tt.selector},
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

	chain := sui.Chain{ChainMetadata: sui.ChainMetadata{Selector: 13264668187771770619}}

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

				result, err := chain.AddressToBytes(tt.address)

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
				name:    "invalid hex characters",
				address: "invalid",
			},
			{
				name:    "empty address",
				address: "",
			},
			{
				name:    "address too short",
				address: "0xa402ce953053607dffcdfec89406c579c8d8ddb9c90e01b7aa28f5f1538ac2",
			},
			{
				name:    "address too long",
				address: "0xa402ce953053607dffcdfec89406c579c8d8ddb9c90e01b7aa28f5f1538ac289a",
			},
			{
				name:    "invalid hex with 0x prefix",
				address: "0xGGGGce953053607dffcdfec89406c579c8d8ddb9c90e01b7aa28f5f1538ac289",
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
			{name: "with 0x prefix", address: "0xa402ce953053607dffcdfec89406c579c8d8ddb9c90e01b7aa28f5f1538ac289"},
			{name: "without 0x prefix", address: "a402ce953053607dffcdfec89406c579c8d8ddb9c90e01b7aa28f5f1538ac289"},
		}

		var results [][]byte
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // Cannot use t.Parallel() here because tests share the results slice
				result, err := chain.AddressToBytes(tt.address)
				require.NoError(t, err, "Failed to convert address: %s", tt.address)
				results = append(results, result)
			})
		}

		// Both formats should produce the same result
		t.Run("both formats produce same result", func(t *testing.T) { //nolint:paralleltest // Cannot use t.Parallel() here because it depends on results from previous tests
			if len(results) >= 2 {
				assert.Equal(t, results[0], results[1], "Sui addresses with and without 0x should produce same result")
			}
		})
	})

	t.Run("consistent results", func(t *testing.T) {
		t.Parallel()

		address := "0xa402ce953053607dffcdfec89406c579c8d8ddb9c90e01b7aa28f5f1538ac289"

		result1, err1 := chain.AddressToBytes(address)
		require.NoError(t, err1)

		result2, err2 := chain.AddressToBytes(address)
		require.NoError(t, err2)

		assert.Equal(t, result1, result2, "Expected consistent results for the same address")
	})
}
