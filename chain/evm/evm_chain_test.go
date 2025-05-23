package evm_test

import (
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
)

func TestChain_ChainInfot(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := evm.Chain{
				Selector: tt.selector,
			}
			assert.Equal(t, tt.selector, provider.ChainSelector())
			assert.Equal(t, tt.wantString, provider.String())
			assert.Equal(t, tt.wantName, provider.Name())
		})
	}
}
