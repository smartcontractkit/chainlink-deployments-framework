package tron_test

import (
	"testing"
	"time"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
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
			selector:   chain_selectors.TRON_MAINNET.Selector,
			wantString: "tron-mainnet (1546563616611573945)",
			wantName:   chain_selectors.TRON_MAINNET.Name,
			wantFamily: chain_selectors.FamilyTron,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := tron.Chain{
				ChainMetadata: tron.ChainMetadata{Selector: tt.selector},
			}
			assert.Equal(t, tt.selector, c.ChainSelector())
			assert.Equal(t, tt.wantString, c.String())
			assert.Equal(t, tt.wantName, c.Name())
			assert.Equal(t, tt.wantFamily, c.Family())
		})
	}
}

func Test_DefaultOptions(t *testing.T) {
	t.Parallel()

	t.Run("DefaultConfirmRetryOptions", func(t *testing.T) {
		t.Parallel()
		opts := tron.DefaultConfirmRetryOptions()
		assert.Equal(t, uint(180), opts.RetryAttempts)
		assert.Equal(t, 500*time.Millisecond, opts.RetryDelay)
	})

	t.Run("DefaultDeployOptions", func(t *testing.T) {
		t.Parallel()
		opts := tron.DefaultDeployOptions()
		assert.Equal(t, 100_000_000, opts.FeeLimit)
		assert.Equal(t, 100, opts.CurPercent)
		assert.Equal(t, 50_000_000, opts.OeLimit)
		assert.Equal(t, tron.DefaultConfirmRetryOptions(), opts.ConfirmRetryOptions)
	})

	t.Run("DefaultTriggerOptions", func(t *testing.T) {
		t.Parallel()
		opts := tron.DefaultTriggerOptions()
		assert.Equal(t, int32(10_000_000), opts.FeeLimit)
		assert.Equal(t, int64(0), opts.TAmount)
		assert.Equal(t, tron.DefaultConfirmRetryOptions(), opts.ConfirmRetryOptions)
	})
}

func TestChain_AddressToBytes(t *testing.T) {
	t.Parallel()

	chain := tron.Chain{ChainMetadata: tron.ChainMetadata{Selector: 6433500567565415381}}

	t.Run("valid addresses", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name        string
			address     string
			description string
		}{
			{
				name:        "standard TRON address",
				address:     "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH",
				description: "typical TRON wallet address",
			},
			{
				name:        "USDT contract address",
				address:     "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
				description: "USDT contract on TRON network",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				result, err := chain.AddressToBytes(tt.address)

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
				name:    "invalid characters",
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
				name:    "invalid base58 characters",
				address: "InvalidBase58!",
			},
			{
				name:    "wrong prefix (Bitcoin-like)",
				address: "1LyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH",
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

		address := "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"

		result1, err1 := chain.AddressToBytes(address)
		require.NoError(t, err1)

		result2, err2 := chain.AddressToBytes(address)
		require.NoError(t, err2)

		assert.Equal(t, result1, result2, "Expected consistent results for the same address")
	})
}
