package proposalutils

import (
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	cldfevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
)

func TestMcmsTimelockInspectorForChain(t *testing.T) {
	t.Parallel()

	evmSelector := chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector

	tests := []struct {
		name    string
		chains  map[uint64]chain.BlockChain
		chain   uint64
		wantErr string
	}{
		{
			name: "success for loaded evm chain",
			chains: map[uint64]chain.BlockChain{
				evmSelector: cldfevm.Chain{Selector: evmSelector},
			},
			chain: evmSelector,
		},
		{
			name:    "error when chain not loaded",
			chains:  nil,
			chain:   evmSelector,
			wantErr: "missing EVM chain client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inspector, err := McmsTimelockInspectorForChain(
				chain.NewBlockChains(tt.chains),
				tt.chain,
				mcmstypes.ChainMetadata{},
			)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, inspector)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, inspector)
		})
	}
}
