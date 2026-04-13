package proposalutils

import (
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	cldfevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func TestWithTimelockAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		action mcmstypes.TimelockAction
	}{
		{
			name:   "sets schedule action",
			action: mcmstypes.TimelockActionSchedule,
		},
		{
			name:   "sets cancel action",
			action: mcmstypes.TimelockActionCancel,
		},
		{
			name:   "sets bypass action",
			action: mcmstypes.TimelockActionBypass,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var opts mcmsInspectorOptions
			WithTimelockAction(tt.action)(&opts)
			assert.Equal(t, tt.action, opts.TimelockAction)
		})
	}
}

func TestMcmsInspectorForChain(t *testing.T) {
	t.Parallel()

	evmSelector := chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector

	tests := []struct {
		name    string
		chains  map[uint64]chain.BlockChain
		chain   uint64
		opts    []MCMSInspectorOption
		wantErr string
	}{
		{
			name: "success with default action",
			chains: map[uint64]chain.BlockChain{
				evmSelector: cldfevm.Chain{Selector: evmSelector},
			},
			chain: evmSelector,
		},
		{
			name: "success with custom action",
			chains: map[uint64]chain.BlockChain{
				evmSelector: cldfevm.Chain{Selector: evmSelector},
			},
			chain: evmSelector,
			opts:  []MCMSInspectorOption{WithTimelockAction(mcmstypes.TimelockActionBypass)},
		},
		{
			name:    "error when chain not in environment",
			chains:  nil,
			chain:   evmSelector,
			wantErr: "missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := cldf.Environment{
				BlockChains: chain.NewBlockChains(tt.chains),
			}

			inspector, err := McmsInspectorForChain(env, tt.chain, tt.opts...)

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

func TestMcmsInspectors(t *testing.T) {
	t.Parallel()

	evmSelector1 := chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector
	evmSelector2 := chainsel.ETHEREUM_MAINNET.Selector

	tests := []struct {
		name          string
		chains        map[uint64]chain.BlockChain
		wantLen       int
		wantSelectors []uint64
	}{
		{
			name:          "empty blockchains returns empty map",
			chains:        nil,
			wantLen:       0,
			wantSelectors: nil,
		},
		{
			name: "single chain returns single inspector with uint64 key",
			chains: map[uint64]chain.BlockChain{
				evmSelector1: cldfevm.Chain{Selector: evmSelector1},
			},
			wantLen:       1,
			wantSelectors: []uint64{evmSelector1},
		},
		{
			name: "multiple chains returns inspector per chain",
			chains: map[uint64]chain.BlockChain{
				evmSelector1: cldfevm.Chain{Selector: evmSelector1},
				evmSelector2: cldfevm.Chain{Selector: evmSelector2},
			},
			wantLen:       2,
			wantSelectors: []uint64{evmSelector1, evmSelector2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := cldf.Environment{
				BlockChains: chain.NewBlockChains(tt.chains),
			}

			inspectors, err := McmsInspectors(env)
			require.NoError(t, err)
			require.Len(t, inspectors, tt.wantLen)

			for _, sel := range tt.wantSelectors {
				assert.NotNil(t, inspectors[sel], "expected inspector for selector %d", sel)
			}
		})
	}
}
