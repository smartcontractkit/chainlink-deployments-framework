package provider

import (
	"testing"
	"time"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
)

func Test_SimChainProvider_Initialize(t *testing.T) {
	t.Parallel()

	var (
		chainSelector = chain_selectors.TEST_1000.Selector
	)

	tests := []struct {
		name           string
		giveSelector   uint64
		giveConfig     SimChainProviderConfig
		wantMinedBlock bool // Indicates whether a block should be mined automatically after initialization.
		wantErr        string
	}{
		{
			name:         "valid initialization",
			giveSelector: chainSelector,
			giveConfig: SimChainProviderConfig{
				NumAdditionalAccounts: 1,
			},
		},
		{
			name:         "valid initialization with automated block mining",
			giveSelector: chainSelector,
			giveConfig: SimChainProviderConfig{
				BlockTime: 10 * time.Millisecond,
			},
			wantMinedBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewSimChainProvider(t, tt.giveSelector, tt.giveConfig)

			got, err := p.Initialize(t.Context())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, p.chain)

				gotChain, ok := got.(evm.Chain)
				require.True(t, ok, "expected got to be of type evm.Chain")

				assert.Equal(t, tt.giveSelector, gotChain.Selector)
				assert.NotNil(t, gotChain.Client)
				assert.NotNil(t, gotChain.DeployerKey)
				assert.Len(t, gotChain.Users, int(tt.giveConfig.NumAdditionalAccounts)) //nolint:gosec // G115 overflow issue will not occur here
				assert.NotNil(t, gotChain.Confirm)

				// Check for the automated block mining if configured
				if tt.wantMinedBlock {
					// Cast the client to access the BlockNumber method
					c, ok := gotChain.Client.(*SimClient)
					require.True(t, ok, "expected gotChain.Client to be of type SimClient")

					assert.Eventually(t, func() bool {
						blockNum, err := c.BlockNumber(t.Context())
						if err != nil {
							return false
						}

						return blockNum > 1 // We commit the genesis block, so we expect at least 2 blocks (genesis + 1 mined block)
					}, 1*time.Second, 10*time.Millisecond)
				}
			}
		})
	}
}

func Test_SimChainProvider_Name(t *testing.T) {
	t.Parallel()

	p := &SimChainProvider{}
	assert.Equal(t, "Simulated EVM Chain Provider", p.Name())
}

func Test_SimChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	p := &SimChainProvider{selector: chain_selectors.TEST_1000.Selector}
	assert.Equal(t, chain_selectors.TEST_1000.Selector, p.ChainSelector())
}

func Test_RPCChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	chain := &evm.Chain{}

	p := &SimChainProvider{
		chain: chain,
	}

	assert.Equal(t, *chain, p.BlockChain())
}
