package ton_test

import (
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/ton"
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
			selector:   chain_selectors.TON_MAINNET.Selector,
			wantString: "ton-mainnet (16448340667252469081)",
			wantName:   chain_selectors.TON_MAINNET.Name,
			wantFamily: chain_selectors.FamilyTon,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := ton.Chain{
				ChainMetadata: ton.ChainMetadata{Selector: tt.selector},
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

	chain := ton.Chain{ChainMetadata: ton.ChainMetadata{Selector: 7668063110026875610}}

	t.Run("valid addresses", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			address string
		}{
			{
				name:    "valid address",
				address: "EQAAAQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHx2j",
			},
			{
				name:    "valid address with zero data",
				address: "EQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAd99",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				result, err := chain.AddressToBytes(tt.address)

				require.NoError(t, err, "Should successfully parse valid TON address: %s", tt.address)
				assert.Len(t, result, 32, "TON address should produce 32 bytes")
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
				name:    "invalid hex characters",
				address: "invalid",
			},
			{
				name:    "empty address",
				address: "",
			},
			{
				name:    "too short",
				address: "123",
			},
			{
				name:    "invalid base64 format",
				address: "EQ!!!invalid!!!",
			},
			{
				name:    "wrong length after decoding",
				address: "EQD4FPq-PRDieyQKkizFTRtSDyucUIqrj0MSWb_BdJKL", // truncated
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

	t.Run("consistent results", func(t *testing.T) {
		t.Parallel()

		address := "EQAAAQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHx2j"

		result1, err1 := chain.AddressToBytes(address)
		require.NoError(t, err1)

		result2, err2 := chain.AddressToBytes(address)
		require.NoError(t, err2)

		assert.Equal(t, result1, result2, "Expected consistent results for the same address")
	})
}
