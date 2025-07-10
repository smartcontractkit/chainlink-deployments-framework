package tron_test

import (
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
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
