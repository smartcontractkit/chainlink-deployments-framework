package run

import (
	"context"

	cs "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// ConfigureEnvironmentOptions builds the environment loading options for a changeset execution.
func ConfigureEnvironmentOptions(
	ctx context.Context,
	registry *cs.ChangesetsRegistry,
	changesetStr string,
	dryRun bool,
	lggr logger.Logger,
) ([]environment.LoadEnvironmentOption, error) {
	var envOptions []environment.LoadEnvironmentOption

	envOptions = append(envOptions, environment.WithLogger(lggr))

	changesetOptions, err := registry.GetChangesetOptions(changesetStr)
	if err != nil {
		return nil, err
	}

	chainOverrides, err := GetChainOverrides(ctx, registry, changesetStr)
	if err != nil {
		return nil, err
	}
	if chainOverrides != nil {
		envOptions = append(envOptions, environment.OnlyLoadChainsFor(chainOverrides))
	}

	if changesetOptions.WithoutJD {
		envOptions = append(envOptions, environment.WithoutJD())
	}

	if changesetOptions.OperationRegistry != nil {
		envOptions = append(envOptions, environment.WithOperationRegistry(changesetOptions.OperationRegistry))
	}

	if dryRun {
		envOptions = append(envOptions, environment.WithDryRunJobDistributor())
	}

	return envOptions, nil
}

// GetChainOverrides retrieves the chain overrides for a given changeset.
// It fails fast with a clear error if any of the resolved chain selectors have
// been marked as decommissioned/deprecated, preventing misleading RPC errors
// (e.g. "insufficient funds", "underpriced transaction") that occur when
// interacting with a sunset chain.
func GetChainOverrides(ctx context.Context, registry *cs.ChangesetsRegistry, changesetStr string) ([]uint64, error) {
	changesetOptions, err := registry.GetChangesetOptions(changesetStr)
	if err != nil {
		return nil, err
	}

	var chainOverrides []uint64
	if changesetOptions.ChainsToLoad != nil {
		chainOverrides = changesetOptions.ChainsToLoad
	} else {
		configs, err := registry.GetConfigurations(changesetStr)
		if err != nil {
			return nil, err
		}
		chainOverrides = configs.InputChainOverrides
	}

	// Validate that no requested chain has been decommissioned. Only check
	// when overrides are explicitly provided (non-nil); a nil value means
	// "load all chains" and is handled by LoadChains downstream.
	if chainOverrides != nil {
		if err := checkDecommissionedChains(ctx, defaultChainDetailsChecker{}, chainOverrides); err != nil {
			return nil, err
		}
	}

	return chainOverrides, nil
}

// SaveReports saves any new operations reports generated during changeset execution.
func SaveReports(
	reporter operations.Reporter,
	originalReportsLen int,
	lggr logger.Logger,
	artdir *domain.ArtifactsDir,
	changesetStr string,
) error {
	latestReports, err := reporter.GetReports()
	if err != nil {
		return err
	}
	newReports := len(latestReports) - originalReportsLen
	if newReports > 0 {
		lggr.Infof("Saving %d new operations reports...", newReports)
		if err := artdir.SaveOperationsReports(changesetStr, latestReports); err != nil {
			return err
		}
	}

	return nil
}
