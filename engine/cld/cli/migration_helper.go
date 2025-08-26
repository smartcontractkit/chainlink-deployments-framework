package cli

import (
	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"

	"github.com/smartcontractkit/chainlink-deployments-framework/operations"

	"github.com/smartcontractkit/chainlink-deployments/pkg/environment"
	"github.com/smartcontractkit/chainlink-deployments/pkg/migrations"
)

func saveReports(
	// TODO: rename to changesetStr when we deprecate migrations
	reporter operations.Reporter, originalReportsLen int, lggr logger.Logger, artdir *domain.ArtifactsDir, migrationStr string,
) error {
	latestReports, err := reporter.GetReports()
	if err != nil {
		return err
	}
	newReports := len(latestReports) - originalReportsLen
	if newReports > 0 {
		lggr.Infof("Saving %d new operations reports...", newReports)
		if err := artdir.SaveOperationsReports(migrationStr, latestReports); err != nil {
			return err
		}
	}

	return nil
}

func configureEnvironmentOptions(migration *migrations.MigrationsRegistry, migrationStr string) ([]environment.LoadEnvironmentOption, error) {
	var envOptions []environment.LoadEnvironmentOption

	migrationOptions, err := migration.GetMigrationOptions(migrationStr)
	if err != nil {
		return nil, err
	}

	chainOverrides, err := getChainOverrides(migration, migrationStr)
	if err != nil {
		return nil, err
	}
	if chainOverrides != nil {
		envOptions = append(envOptions, environment.OnlyLoadChainsFor(migrationStr, chainOverrides))
	}

	if migrationOptions.WithoutJD {
		envOptions = append(envOptions, environment.WithoutJD())
	}
	if migrationOptions.OperationRegistry != nil {
		envOptions = append(envOptions, environment.WithOperationRegistry(migrationOptions.OperationRegistry))
	}

	return envOptions, nil
}
