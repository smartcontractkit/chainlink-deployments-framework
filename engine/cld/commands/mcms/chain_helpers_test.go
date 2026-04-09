package mcms

import (
	"context"
	"math"
	"testing"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
)

// uint32Unix converts tm.Unix() to uint32 after a range check (gosec G115).
func uint32Unix(t *testing.T, tm time.Time) uint32 {
	t.Helper()
	sec := tm.Unix()
	if sec < 0 || sec > int64(math.MaxUint32) {
		t.Fatalf("unix timestamp %d out of uint32 range", sec)
	}

	return uint32(sec)
}

func TestSelectorFamily_EVM(t *testing.T) {
	t.Parallel()

	sel := types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)
	fam, err := selectorFamily(sel)
	require.NoError(t, err)
	require.Equal(t, chainsel.FamilyEVM, fam)
}

func TestNewChainAccessor(t *testing.T) {
	t.Parallel()

	cfg := &forkConfig{blockchains: chain.NewBlockChains(nil)}
	acc := newChainAccessor(cfg)
	require.NotNil(t, acc)
	require.Empty(t, acc.Selectors())
}

func TestGetInspectorFromChainSelector_MissingProposalChainMetadata(t *testing.T) {
	t.Parallel()

	sel := chainsel.ETHEREUM_MAINNET.Selector
	cfg := &forkConfig{
		chainSelector: sel,
		proposal: mcms.Proposal{
			BaseProposal: mcms.BaseProposal{
				Version:       "v1",
				Kind:          types.KindProposal,
				ValidUntil:    uint32Unix(t, time.Now().Add(24*time.Hour)),
				ChainMetadata: map[types.ChainSelector]types.ChainMetadata{},
			},
		},
		timelockProposal: &mcms.TimelockProposal{},
		blockchains:      chain.NewBlockChains(nil),
	}

	_, err := getInspectorFromChainSelector(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get chain metadata")
}

func TestGetInspectorFromChainSelector_AptosUnknownAction(t *testing.T) {
	t.Parallel()

	sel := types.ChainSelector(chainsel.APTOS_TESTNET.Selector)
	cfg := &forkConfig{
		chainSelector: uint64(sel),
		proposal: mcms.Proposal{
			BaseProposal: mcms.BaseProposal{
				Version:    "v1",
				Kind:       types.KindProposal,
				ValidUntil: uint32Unix(t, time.Now().Add(24*time.Hour)),
				ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
					sel: {MCMAddress: "0x1", StartingOpCount: 0},
				},
			},
		},
		timelockProposal: &mcms.TimelockProposal{
			Action: types.TimelockAction("execute"),
		},
		blockchains: chain.NewBlockChains(nil),
	}

	_, err := getInspectorFromChainSelector(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown timelock action")
}

func TestGetInspectorFromChainSelector_SuiInvalidMetadata(t *testing.T) {
	t.Parallel()

	sel := types.ChainSelector(chainsel.SUI_TESTNET.Selector)
	cfg := &forkConfig{
		chainSelector: uint64(sel),
		proposal: mcms.Proposal{
			BaseProposal: mcms.BaseProposal{
				Version:    "v1",
				Kind:       types.KindProposal,
				ValidUntil: uint32Unix(t, time.Now().Add(24*time.Hour)),
				ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
					sel: {
						MCMAddress:       "0x1",
						StartingOpCount:  0,
						AdditionalFields: []byte(`{"role":0}`),
					},
				},
			},
		},
		timelockProposal: &mcms.TimelockProposal{
			Action: types.TimelockActionSchedule,
		},
		blockchains: chain.NewBlockChains(nil),
	}

	_, err := getInspectorFromChainSelector(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "sui chain metadata")
}

func TestGetInspectorFromChainSelector_DefaultUnsupportedFamily(t *testing.T) {
	t.Parallel()

	sel := types.ChainSelector(chainsel.STELLAR_TESTNET.Selector)
	cfg := &forkConfig{
		chainSelector: uint64(sel),
		proposal: mcms.Proposal{
			BaseProposal: mcms.BaseProposal{
				Version:    "v1",
				Kind:       types.KindProposal,
				ValidUntil: uint32Unix(t, time.Now().Add(24*time.Hour)),
				ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
					sel: {MCMAddress: "0x1", StartingOpCount: 0},
				},
			},
		},
		timelockProposal: &mcms.TimelockProposal{
			Action: types.TimelockActionSchedule,
		},
		blockchains: chain.NewBlockChains(nil),
	}

	_, err := getInspectorFromChainSelector(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported chain family")
}

func TestGetExecutorWithChainOverride_MissingTimelockChainMetadata(t *testing.T) {
	t.Parallel()

	sel := types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)
	proposal := mcms.Proposal{
		BaseProposal: mcms.BaseProposal{
			Version:    "v1",
			Kind:       types.KindProposal,
			ValidUntil: uint32Unix(t, time.Now().Add(24*time.Hour)),
			ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
				sel: {MCMAddress: "0x1", StartingOpCount: 0},
			},
		},
	}
	cfg := &forkConfig{
		proposal: proposal,
		timelockProposal: &mcms.TimelockProposal{
			BaseProposal: mcms.BaseProposal{
				ChainMetadata: map[types.ChainSelector]types.ChainMetadata{},
			},
		},
		blockchains: chain.NewBlockChains(nil),
	}

	_, err := getExecutorWithChainOverride(cfg, sel)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get chain metadata from timelock proposal")
}

func TestGetTimelockExecutorWithChainOverride_MissingMetadata(t *testing.T) {
	t.Parallel()

	sel := types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)
	cfg := &forkConfig{
		timelockProposal: &mcms.TimelockProposal{
			BaseProposal: mcms.BaseProposal{
				ChainMetadata: map[types.ChainSelector]types.ChainMetadata{},
			},
		},
		blockchains: chain.NewBlockChains(nil),
	}

	_, err := getTimelockExecutorWithChainOverride(cfg, sel)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get chain metadata from timelock proposal")
}

func TestCreateExecutable_PropagatesExecutorError(t *testing.T) {
	t.Parallel()

	sel := types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)
	proposal := mcms.Proposal{
		BaseProposal: mcms.BaseProposal{
			Version:    "v1",
			Kind:       types.KindProposal,
			ValidUntil: uint32Unix(t, time.Now().Add(24*time.Hour)),
			ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
				sel: {MCMAddress: "0x1", StartingOpCount: 0},
			},
		},
	}
	cfg := &forkConfig{
		chainSelector: 0,
		proposal:      proposal,
		timelockProposal: &mcms.TimelockProposal{
			BaseProposal: mcms.BaseProposal{
				ChainMetadata: map[types.ChainSelector]types.ChainMetadata{},
			},
		},
		blockchains: chain.NewBlockChains(nil),
	}

	_, err := createExecutable(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to get executor with chain override")
}

func TestCreateTimelockExecutable_PropagatesExecutorError(t *testing.T) {
	t.Parallel()

	sel := types.ChainSelector(chainsel.APTOS_TESTNET.Selector)
	cfg := &forkConfig{
		chainSelector: 0,
		timelockProposal: &mcms.TimelockProposal{
			BaseProposal: mcms.BaseProposal{
				ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
					sel: {MCMAddress: "0x1", StartingOpCount: 0},
				},
			},
			Action: types.TimelockAction("invalid"),
		},
		blockchains: chain.NewBlockChains(nil),
	}

	_, err := createTimelockExecutable(context.Background(), cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown timelock action")
}
