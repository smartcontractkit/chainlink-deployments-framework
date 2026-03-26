package common_test

import (
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

func TestChainMetadata(t *testing.T) {
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
			selector:   chainsel.ETHEREUM_MAINNET.Selector,
			wantString: "ethereum-mainnet (5009297550715157269)",
			wantName:   chainsel.ETHEREUM_MAINNET.Name,
			wantFamily: chainsel.FamilyEVM,
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

			c := common.ChainMetadata{
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

	c := common.ChainMetadata{Selector: chainsel.ETHEREUM_MAINNET.Selector}
	got, err := c.NetworkType()
	require.NoError(t, err)
	assert.Equal(t, chainsel.NetworkTypeMainnet, got)

	c = common.ChainMetadata{Selector: 0}
	_, err = c.NetworkType()
	require.Error(t, err)
}

func TestChainMetadata_IsNetworkType(t *testing.T) {
	t.Parallel()

	c := common.ChainMetadata{Selector: chainsel.ETHEREUM_MAINNET.Selector}

	assert.True(t, c.IsNetworkType(chainsel.NetworkTypeMainnet))
	assert.False(t, c.IsNetworkType(chainsel.NetworkTypeTestnet))
}
