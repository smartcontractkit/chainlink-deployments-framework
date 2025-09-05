package solana_test

import (
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
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
			selector:   chain_selectors.SOLANA_MAINNET.Selector,
			wantString: "solana-mainnet (124615329519749607)",
			wantName:   chain_selectors.SOLANA_MAINNET.Name,
			wantFamily: chain_selectors.FamilySolana,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := solana.Chain{
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

	chain := solana.Chain{Selector: 1151111081099710}

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

			result, err := chain.AddressToBytes(tt.address)

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

		result1, err1 := chain.AddressToBytes(address)
		require.NoError(t, err1)

		result2, err2 := chain.AddressToBytes(address)
		require.NoError(t, err2)

		assert.Equal(t, result1, result2, "Expected consistent results for the same address")
	})
}
