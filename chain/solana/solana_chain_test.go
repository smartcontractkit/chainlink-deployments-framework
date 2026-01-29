package solana_test

import (
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
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
			selector:   chainsel.SOLANA_MAINNET.Selector,
			wantString: "solana-mainnet (124615329519749607)",
			wantName:   chainsel.SOLANA_MAINNET.Name,
			wantFamily: chainsel.FamilySolana,
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

func TestChainMetadata_NetworkType(t *testing.T) {
	t.Parallel()

	c := solana.Chain{Selector: chainsel.SOLANA_MAINNET.Selector}
	got, err := c.NetworkType()
	require.NoError(t, err)
	assert.Equal(t, chainsel.NetworkTypeMainnet, got)

	c = solana.Chain{Selector: 0}
	_, err = c.NetworkType()
	require.Error(t, err)
}

func TestChainMetadata_IsNetworkType(t *testing.T) {
	t.Parallel()

	c := solana.Chain{Selector: chainsel.SOLANA_MAINNET.Selector}

	assert.True(t, c.IsNetworkType(chainsel.NetworkTypeMainnet))
	assert.False(t, c.IsNetworkType(chainsel.NetworkTypeTestnet))
}
