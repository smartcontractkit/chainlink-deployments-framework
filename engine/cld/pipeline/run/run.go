package run

import (
	cs "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// ConfigureEnvironmentOptions builds the environment loading options for a changeset execution.
func ConfigureEnvironmentOptions(
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

	chainOverrides, err := GetChainOverrides(registry, changesetStr)
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
func GetChainOverrides(registry *cs.ChangesetsRegistry, changesetStr string) ([]uint64, error) {
	changesetOptions, err := registry.GetChangesetOptions(changesetStr)
	if err != nil {
		return nil, err
	}

	if changesetOptions.ChainsToLoad != nil {
		return changesetOptions.ChainsToLoad, nil
	}

	configs, err := registry.GetConfigurations(changesetStr)
	if err != nil {
		return nil, err
	}

	return configs.InputChainOverrides, nil
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
