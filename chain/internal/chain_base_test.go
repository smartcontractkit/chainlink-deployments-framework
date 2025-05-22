package internal_test

import (
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal"
)

func TestChainInfoProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		selector   uint64
		wantName   string
		wantString string
	}{
		{
			name:       "returns correct info",
			selector:   chain_selectors.ETHEREUM_MAINNET.Selector,
			wantString: "ethereum-mainnet (5009297550715157269)",
			wantName:   chain_selectors.ETHEREUM_MAINNET.Name,
		},
		{
			name:       "returns empty for unknown chain",
			selector:   0,
			wantString: "",
			wantName:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := internal.ChainBase{
				Selector: tt.selector,
			}
			assert.Equal(t, tt.selector, provider.ChainSelector())
			assert.Equal(t, tt.wantString, provider.String())
			assert.Equal(t, tt.wantName, provider.Name())
		})
	}
}

func TestChainInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		selector      uint64
		expectError   string
		validateChain func(t *testing.T, info chain_selectors.ChainDetails)
	}{
		{
			name:     "returns details for valid chain selector",
			selector: chain_selectors.ETHEREUM_MAINNET.Selector,
			validateChain: func(t *testing.T, info chain_selectors.ChainDetails) {
				t.Helper()
				assert.Equal(t, chain_selectors.ETHEREUM_MAINNET.Name, info.ChainName)
				assert.Equal(t, chain_selectors.ETHEREUM_MAINNET.Selector, info.ChainSelector)
			},
		},
		{
			name:        "returns error for invalid chain selector",
			selector:    0, // Invalid selector
			expectError: "unknown chain selector 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			info, err := internal.ChainInfo(tt.selector)

			if len(tt.expectError) > 0 {
				assert.ErrorContains(t, err, tt.expectError)
				return
			}

			require.NoError(t, err)
			if tt.validateChain != nil {
				tt.validateChain(t, info)
			}
		})
	}
}
