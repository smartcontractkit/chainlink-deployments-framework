package datastore

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	cfgdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/legacy/cli"
)

var (
	syncToCatalogShort = "Sync local datastore to catalog"

	syncToCatalogLong = cli.LongDesc(`
		Syncs the entire local datastore to the catalog service. This is used for initial
		migration from file-based to catalog-based datastore management.

		The environment must have catalog configured (datastore type: catalog or all).
	`)

	syncToCatalogExample = cli.Examples(`
		# Sync the entire local datastore to catalog
		ccip datastore sync-to-catalog --environment staging
	`)
)

type syncToCatalogFlags struct {
	environment string
}

// newSyncToCatalogCmd creates the "sync-to-catalog" subcommand.
func newSyncToCatalogCmd(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sync-to-catalog",
		Short:   syncToCatalogShort,
		Long:    syncToCatalogLong,
		Example: syncToCatalogExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			f := syncToCatalogFlags{
				environment: flags.MustString(cmd.Flags().GetString("environment")),
			}

			return runSyncToCatalog(cmd, cfg, f)
		},
	}

	// Shared flags
	flags.Environment(cmd)

	return cmd
}

// runSyncToCatalog executes the sync-to-catalog command logic.
func runSyncToCatalog(cmd *cobra.Command, cfg Config, f syncToCatalogFlags) error {
	ctx := cmd.Context()
	deps := cfg.deps()
	envDir := cfg.Domain.EnvDir(f.environment)

	// --- Load

	// Load config to get catalog connection details
	envCfg, err := deps.ConfigLoader(cfg.Domain, f.environment, cfg.Logger)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Verify catalog is configured
	if envCfg.DatastoreType != cfgdomain.DatastoreTypeCatalog && envCfg.DatastoreType != cfgdomain.DatastoreTypeAll {
		return fmt.Errorf("catalog is not configured for environment %s (datastore type: %s)",
			f.environment, envCfg.DatastoreType)
	}

	// --- Execute

	cmd.Printf("📡 Syncing local datastore to catalog (endpoint: %s)\n", envCfg.Env.Catalog.GRPC)

	catalog, catalogErr := deps.CatalogLoader(ctx, f.environment, envCfg, cfg.Domain)
	if catalogErr != nil {
		return fmt.Errorf("failed to load catalog: %w", catalogErr)
	}

	if err := deps.CatalogSyncer(ctx, envDir, catalog); err != nil {
		return fmt.Errorf("error syncing datastore to catalog for %s %s: %w",
			cfg.Domain, f.environment, err,
		)
	}

	cmd.Printf("✅ Successfully synced entire datastore to catalog for %s %s\n",
		cfg.Domain, f.environment,
	)

	return nil
}
