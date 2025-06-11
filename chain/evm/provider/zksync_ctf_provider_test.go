package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/testutils"
)

func Test_ZkSyncCTFChainProviderConfig_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  ZkSyncCTFChainProviderConfig
		wantErr string
	}{
		{
			name: "valid config",
			config: ZkSyncCTFChainProviderConfig{
				Once: testutils.DefaultNetworkOnce,
			},
			wantErr: "",
		},
		{
			name: "missing sync.Once instance",
			config: ZkSyncCTFChainProviderConfig{
				Once: nil,
			},
			wantErr: "sync.Once instance is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.validate()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_CTFChainProvider_Initialize(t *testing.T) {
	t.Parallel()

	var chainSelector = chain_selectors.TEST_1000.Selector

	tests := []struct {
		name         string
		giveSelector uint64
		giveConfig   ZkSyncCTFChainProviderConfig
		wantErr      string
	}{
		{
			name:         "valid initialization",
			giveSelector: chainSelector,
			giveConfig: ZkSyncCTFChainProviderConfig{
				Once: testutils.DefaultNetworkOnce,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewZkSyncCTFChainProvider(t, tt.giveSelector, tt.giveConfig)

			got, err := p.Initialize(t.Context())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, p.chain)

				// Check that the chain is of type *aptos.Chain and has the expected fields
				gotChain, ok := got.(evm.Chain)
				require.True(t, ok, "expected got to be of type aptos.Chain")
				assert.Equal(t, tt.giveSelector, gotChain.Selector)
				assert.NotEmpty(t, gotChain.Client)
				assert.NotEmpty(t, gotChain.DeployerKey)
				assert.NotEmpty(t, gotChain.Users)
				assert.NotNil(t, gotChain.Confirm)
				assert.True(t, gotChain.IsZkSyncVM)
				assert.NotNil(t, gotChain.ClientZkSyncVM)
				assert.NotNil(t, gotChain.DeployerKeyZkSyncVM)
			}
		})
	}
}

func Test_ZkSyncCTFChainProvider_Name(t *testing.T) {
	t.Parallel()

	p := &ZkSyncCTFChainProvider{}
	assert.Equal(t, "ZkSync EVM CTF Chain Provider", p.Name())
}

func Test_ZkSyncCTFChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	p := &ZkSyncCTFChainProvider{selector: chain_selectors.TEST_1000.Selector}
	assert.Equal(t, chain_selectors.TEST_1000.Selector, p.ChainSelector())
}

func Test_ZkSyncCTFChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	chain := &evm.Chain{}

	p := &ZkSyncCTFChainProvider{
		chain: chain,
	}

	assert.Equal(t, *chain, p.BlockChain())
}
