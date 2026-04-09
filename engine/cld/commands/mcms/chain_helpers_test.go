package mcms

import (
	"context"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
)

// mcmsTestProposalValidUntil is a far-future MCMS validUntil (uint32 max ≈ year 2106) for test fixtures.
const mcmsTestProposalValidUntil uint32 = 0xffffffff

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
				ValidUntil:    mcmsTestProposalValidUntil,
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
				ValidUntil: mcmsTestProposalValidUntil,
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
				ValidUntil: mcmsTestProposalValidUntil,
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
				ValidUntil: mcmsTestProposalValidUntil,
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
			ValidUntil: mcmsTestProposalValidUntil,
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
			ValidUntil: mcmsTestProposalValidUntil,
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
