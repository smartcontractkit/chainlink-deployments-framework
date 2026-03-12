package mcms

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/chainwrappers"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	cldfmcmsadapters "github.com/smartcontractkit/chainlink-deployments-framework/chain/mcms/adapters"
)

// getInspectorFromChainSelector returns an inspector for the given chain selector.
func getInspectorFromChainSelector(cfg *forkConfig) (sdk.Inspector, error) {
	chainSelector := types.ChainSelector(cfg.chainSelector)
	chainMetadata, ok := cfg.proposal.ChainMetadata[chainSelector]
	if !ok {
		return nil, fmt.Errorf("failed to get chain metadata from timelock proposal for chain selector %v", cfg.chainSelector)
	}

	chainAccessor := cldfmcmsadapters.Wrap(cfg.blockchains)

	return chainwrappers.BuildInspector(&chainAccessor, chainSelector, cfg.timelockProposal.Action, chainMetadata)
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

	chainAccessor := cldfmcmsadapters.Wrap(cfg.blockchains)

	return chainwrappers.BuildExecutor(&chainAccessor, chainSelector, encoder, cfg.timelockProposal.Action, chainMetadata)
}

// getTimelockExecutorWithChainOverride returns a timelock executor for the given chain selector.
func getTimelockExecutorWithChainOverride(cfg *forkConfig, chainSelector types.ChainSelector) (sdk.TimelockExecutor, error) {
	chainMetadata, ok := cfg.timelockProposal.ChainMetadata[chainSelector]
	if !ok {
		return nil, fmt.Errorf("failed to get chain metadata from timelock proposal for chain selector %v", chainSelector)
	}

	chainAccessor := cldfmcmsadapters.Wrap(cfg.blockchains)

	return chainwrappers.BuildTimelockExecutor(&chainAccessor, chainSelector, cfg.timelockProposal.Action, chainMetadata)
}
