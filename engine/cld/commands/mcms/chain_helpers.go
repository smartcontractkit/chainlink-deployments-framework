package mcms

import (
	"context"
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/chainwrappers"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	cldfmcmsadapters "github.com/smartcontractkit/chainlink-deployments-framework/chain/mcms/adapters"
)

func newChainAccessor(cfg *forkConfig) *cldfmcmsadapters.ChainAccessAdapter {
	a := cldfmcmsadapters.Wrap(cfg.blockchains)
	return &a
}

func selectorFamily(sel types.ChainSelector) (string, error) {
	return chainsel.GetSelectorFamily(uint64(sel))
}

// getInspectorFromChainSelector returns an inspector for the given chain selector.
func getInspectorFromChainSelector(cfg *forkConfig) (sdk.Inspector, error) {
	chainSelector := types.ChainSelector(cfg.chainSelector)
	chainMetadata, ok := cfg.proposal.ChainMetadata[chainSelector]
	if !ok {
		return nil, fmt.Errorf("failed to get chain metadata from timelock proposal for chain selector %v", cfg.chainSelector)
	}

	family, err := selectorFamily(chainSelector)
	if err != nil {
		return nil, fmt.Errorf("chain selector family: %w", err)
	}

	acc := newChainAccessor(cfg)
	action := cfg.timelockProposal.Action

	switch family {
	case chainsel.FamilyEVM:
		return buildEVMInspector(acc, action, chainSelector, chainMetadata)
	case chainsel.FamilySolana:
		return buildSolanaInspector(acc, action, chainSelector, chainMetadata)
	case chainsel.FamilyAptos:
		return buildAptosInspector(acc, action, chainSelector, chainMetadata)
	case chainsel.FamilySui:
		return buildSuiInspector(acc, action, chainSelector, chainMetadata)
	case chainsel.FamilyTon:
		return buildTonInspector(acc, action, chainSelector, chainMetadata)
	default:
		return chainwrappers.BuildInspector(acc, chainSelector, action, chainMetadata)
	}
}

// createExecutable creates an MCMS executable for the proposal.
func createExecutable(cfg *forkConfig) (*mcms.Executable, error) {
	executors := make(map[types.ChainSelector]sdk.Executor, len(cfg.proposal.ChainMetadata))
	for chainSelector := range cfg.proposal.ChainMetadata {
		if cfg.chainSelector == 0 || cfg.chainSelector == uint64(chainSelector) {
			executor, err := getExecutorWithChainOverride(cfg, chainSelector)
			if err != nil {
				return &mcms.Executable{}, fmt.Errorf("unable to get executor with chain override: %w", err)
			}
			executors[chainSelector] = executor
		}
	}

	return mcms.NewExecutable(&cfg.proposal, executors)
}

// createTimelockExecutable creates a timelock executable for the proposal.
func createTimelockExecutable(ctx context.Context, cfg *forkConfig) (*mcms.TimelockExecutable, error) {
	executors := make(map[types.ChainSelector]sdk.TimelockExecutor, len(cfg.timelockProposal.ChainMetadata))
	for chainSelector := range cfg.timelockProposal.ChainMetadata {
		if cfg.chainSelector != 0 && cfg.chainSelector != uint64(chainSelector) {
			continue
		}
		executor, err := getTimelockExecutorWithChainOverride(cfg, chainSelector)
		if err != nil {
			return &mcms.TimelockExecutable{}, err
		}
		executors[chainSelector] = executor
	}

	return mcms.NewTimelockExecutable(ctx, cfg.timelockProposal, executors)
}

// getExecutorWithChainOverride returns an executor for the given chain selector.
func getExecutorWithChainOverride(cfg *forkConfig, chainSelector types.ChainSelector) (sdk.Executor, error) {
	encoders, err := cfg.proposal.GetEncoders()
	if err != nil {
		return nil, fmt.Errorf("error getting encoders: %w", err)
	}
	encoder, ok := encoders[chainSelector]
	if !ok {
		return nil, fmt.Errorf("failed to get encoder from proposal for chain selector %v", chainSelector)
	}
	chainMetadata, ok := cfg.timelockProposal.ChainMetadata[chainSelector]
	if !ok {
		return nil, fmt.Errorf("failed to get chain metadata from timelock proposal for chain selector %v", chainSelector)
	}

	family, err := selectorFamily(chainSelector)
	if err != nil {
		return nil, fmt.Errorf("chain selector family: %w", err)
	}

	acc := newChainAccessor(cfg)
	action := cfg.timelockProposal.Action

	switch family {
	case chainsel.FamilyEVM:
		return buildEVMExecutor(acc, action, chainSelector, encoder, chainMetadata)
	case chainsel.FamilySolana:
		return buildSolanaExecutor(acc, action, chainSelector, encoder, chainMetadata)
	case chainsel.FamilyAptos:
		return buildAptosExecutor(acc, action, chainSelector, encoder, chainMetadata)
	case chainsel.FamilySui:
		return buildSuiExecutor(acc, action, chainSelector, encoder, chainMetadata)
	case chainsel.FamilyTon:
		return buildTonExecutor(acc, action, chainSelector, encoder, chainMetadata)
	default:
		return chainwrappers.BuildExecutor(acc, chainSelector, encoder, action, chainMetadata)
	}
}

// getTimelockExecutorWithChainOverride returns a timelock executor for the given chain selector.
func getTimelockExecutorWithChainOverride(cfg *forkConfig, chainSelector types.ChainSelector) (sdk.TimelockExecutor, error) {
	chainMetadata, ok := cfg.timelockProposal.ChainMetadata[chainSelector]
	if !ok {
		return nil, fmt.Errorf("failed to get chain metadata from timelock proposal for chain selector %v", chainSelector)
	}

	family, err := selectorFamily(chainSelector)
	if err != nil {
		return nil, fmt.Errorf("chain selector family: %w", err)
	}

	acc := newChainAccessor(cfg)
	action := cfg.timelockProposal.Action

	switch family {
	case chainsel.FamilyEVM:
		return buildEVMTimelockExecutor(acc, action, chainSelector, chainMetadata)
	case chainsel.FamilySolana:
		return buildSolanaTimelockExecutor(acc, action, chainSelector, chainMetadata)
	case chainsel.FamilyAptos:
		return buildAptosTimelockExecutor(acc, action, chainSelector, chainMetadata)
	case chainsel.FamilySui:
		return buildSuiTimelockExecutor(acc, action, chainSelector, chainMetadata)
	case chainsel.FamilyTon:
		return buildTonTimelockExecutor(acc, action, chainSelector, chainMetadata)
	default:
		return chainwrappers.BuildTimelockExecutor(acc, chainSelector, action, chainMetadata)
	}
}
