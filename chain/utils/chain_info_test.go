package utils_test

import (
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/utils"
)

func TestChainInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		selector      uint64
		expectError   string
		validateChain func(t *testing.T, info chainsel.ChainDetails)
	}{
		{
			name:     "returns details for valid chain selector",
			selector: chainsel.ETHEREUM_MAINNET.Selector,
			validateChain: func(t *testing.T, info chainsel.ChainDetails) {
				t.Helper()
				assert.Equal(t, chainsel.ETHEREUM_MAINNET.Name, info.ChainName)
				assert.Equal(t, chainsel.ETHEREUM_MAINNET.Selector, info.ChainSelector)
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
			info, err := utils.ChainInfo(tt.selector)

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
