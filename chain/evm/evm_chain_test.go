package evm_test

import (
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
)

func TestChain_ChainInfo(t *testing.T) {
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
			selector:   chain_selectors.ETHEREUM_MAINNET.Selector,
			wantString: "ethereum-mainnet (5009297550715157269)",
			wantName:   chain_selectors.ETHEREUM_MAINNET.Name,
			wantFamily: chain_selectors.FamilyEVM,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := evm.Chain{
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

	chain := evm.Chain{Selector: 1}

	tests := []struct {
		name          string
		address       string
		expectedLen   int
		expectedBytes []byte
		shouldSucceed bool
		description   string
	}{
		{
			name:          "valid address with 0x prefix",
			address:       "0x742d35Cc6634C0532925a3b8D4c8C1B8c4c8C1B8",
			expectedLen:   20,
			expectedBytes: []byte{0x74, 0x2d, 0x35, 0xcc, 0x66, 0x34, 0xc0, 0x53, 0x29, 0x25, 0xa3, 0xb8, 0xd4, 0xc8, 0xc1, 0xb8, 0xc4, 0xc8, 0xc1, 0xb8},
			shouldSucceed: true,
			description:   "should convert valid hex address with 0x prefix to bytes",
		},
		{
			name:          "valid address without 0x prefix",
			address:       "742d35Cc6634C0532925a3b8D4c8C1B8c4c8C1B8",
			expectedLen:   20,
			shouldSucceed: true,
			description:   "should convert valid hex address without 0x prefix to bytes",
		},
		{
			name:          "zero address",
			address:       "0x0000000000000000000000000000000000000000",
			expectedLen:   20,
			expectedBytes: make([]byte, 20), // All zeros
			shouldSucceed: true,
			description:   "should convert zero address to 20 zero bytes",
		},
		{
			name:          "invalid - too short",
			address:       "0x123",
			shouldSucceed: false,
			description:   "should reject address that is too short",
		},
		{
			name:          "invalid - invalid hex characters",
			address:       "742d35Cc6634C0532925a3b8D4c8C1B8c4c8C1B8XX",
			shouldSucceed: false,
			description:   "should reject address with invalid hex characters",
		},
		{
			name:          "invalid - empty string",
			address:       "",
			shouldSucceed: false,
			description:   "should reject empty address string",
		},
		{
			name:          "invalid - non-hex string",
			address:       "invalid",
			shouldSucceed: false,
			description:   "should reject non-hex address string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := chain.AddressToBytes(tt.address)

			if tt.shouldSucceed {
				require.NoError(t, err, tt.description)
				assert.Len(t, result, tt.expectedLen, "Expected length %d for address %s", tt.expectedLen, tt.address)

				if tt.expectedBytes != nil {
					assert.Equal(t, tt.expectedBytes, result, "Expected specific bytes for address %s", tt.address)
				}
			} else {
				require.Error(t, err, tt.description)
				assert.Nil(t, result, "Expected nil result for invalid address %s", tt.address)
			}
		})
	}

	t.Run("case sensitivity", func(t *testing.T) {
		t.Parallel()

		// EVM addresses should be case-insensitive
		addr1 := "0x742d35Cc6634C0532925a3b8D4c8C1B8c4c8C1B8"
		addr2 := "0x742D35CC6634C0532925A3B8D4C8C1B8C4C8C1B8"

		result1, err1 := chain.AddressToBytes(addr1)
		result2, err2 := chain.AddressToBytes(addr2)

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, result1, result2, "EVM addresses should be case-insensitive")
	})

	t.Run("consistent results", func(t *testing.T) {
		t.Parallel()

		address := "0x742d35Cc6634C0532925a3b8D4c8C1B8c4c8C1B8"

		result1, err1 := chain.AddressToBytes(address)
		require.NoError(t, err1)

		result2, err2 := chain.AddressToBytes(address)
		require.NoError(t, err2)

		assert.Equal(t, result1, result2, "Expected consistent results for the same address")
	})
}
