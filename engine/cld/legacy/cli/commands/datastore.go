package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	cldcatalog "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/catalog"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	cfgdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/legacy/cli"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// NewDatastoreCmds creates a new set of commands for datastore operations.
func (c Commands) NewDatastoreCmds(domain domain.Domain) *cobra.Command {
	datastoreCmd := &cobra.Command{
		Use:   "datastore",
		Short: "Datastore operations",
	}

	datastoreCmd.AddCommand(c.newDatastoreMerge(domain))
	datastoreCmd.AddCommand(c.newDatastoreSyncToCatalog(domain))

	datastoreCmd.PersistentFlags().StringP("environment", "e", "", "Deployment environment (required)")
	err := datastoreCmd.MarkPersistentFlagRequired("environment")
	if err != nil {
		return nil
	}

	return datastoreCmd
}

var (
	datastoreMergeLong = cli.LongDesc(`
		Merges the datastore artifact of a specific changeset to the main datastore within a
		given Domain Environment. The merge destination depends on the datastore configuration:
		- file: merges to local JSON files
		- catalog: merges to the remote catalog service
		- all: merges to both local files and catalog
	`)

	datastoreMergeExample = cli.Examples(`
		# Merge the datastore for the 0001_deploy_cap changeset in the ccip staging domain
		ccip datastore merge --environment staging --name 0001_deploy_cap

		# Merge with a specific durable pipeline timestamp
		ccip datastore merge --environment staging --name 0001_deploy_cap --timestamp 1234567890
	`)
)

// newDatastoreMerge creates a command to merge the datastore for a changeset
func (Commands) newDatastoreMerge(domain domain.Domain) *cobra.Command {
	var (
		name      string
		timestamp string
	)

	cmd := cobra.Command{
		Use:     "merge",
		Short:   "Merge datastore artifacts",
		Long:    datastoreMergeLong,
		Example: datastoreMergeExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			envDir := domain.EnvDir(envKey)

			// Load config to check datastore type
			cfg, err := config.Load(domain, envKey, logger.Nop())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Determine which merge method to use based on datastore configuration
			switch cfg.DatastoreType {
			case cfgdomain.DatastoreTypeCatalog:
				// Catalog mode - merge to catalog service
				cmd.Printf("📡 Using catalog datastore mode (endpoint: %s)\n", cfg.Env.Catalog.GRPC)

				catalog, catalogErr := cldcatalog.LoadCatalog(cmd.Context(), envKey, cfg, domain)
				if catalogErr != nil {
					return fmt.Errorf("failed to load catalog: %w", catalogErr)
				}

				if err := envDir.MergeMigrationDataStoreCatalog(cmd.Context(), name, timestamp, catalog); err != nil {
					return fmt.Errorf("error during datastore merge to catalog for %s %s %s: %w",
						domain, envKey, name, err,
					)
				}

				cmd.Printf("✅ Merged datastore to catalog for %s %s %s\n",
					domain, envKey, name,
				)
			case cfgdomain.DatastoreTypeFile:
				// File mode - merge to local files
				cmd.Printf("📁 Using file-based datastore mode\n")

				if err := envDir.MergeMigrationDataStore(name, timestamp); err != nil {
					return fmt.Errorf("error during datastore merge to file for %s %s %s: %w",
						domain, envKey, name, err,
					)
				}

				cmd.Printf("✅ Merged datastore to local files for %s %s %s\n",
					domain, envKey, name,
				)
			case cfgdomain.DatastoreTypeAll:
				// All mode - merge to both catalog and local files
				cmd.Printf("📡 Using all datastore mode (catalog: %s, file: %s)\n", cfg.Env.Catalog.GRPC, envDir.DataStoreDirPath())

				catalog, catalogErr := cldcatalog.LoadCatalog(cmd.Context(), envKey, cfg, domain)
				if catalogErr != nil {
					return fmt.Errorf("failed to load catalog: %w", catalogErr)
				}

				if err := envDir.MergeMigrationDataStoreCatalog(cmd.Context(), name, timestamp, catalog); err != nil {
					return fmt.Errorf("error during datastore merge to catalog for %s %s %s: %w",
						domain, envKey, name, err,
					)
				}

				if err := envDir.MergeMigrationDataStore(name, timestamp); err != nil {
					return fmt.Errorf("error during datastore merge to file for %s %s %s: %w",
						domain, envKey, name, err,
					)
				}

				cmd.Printf("✅ Merged datastore to both catalog and local files for %s %s %s\n",
					domain, envKey, name,
				)
			default:
				return fmt.Errorf("invalid datastore type: %s", cfg.DatastoreType)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "name (required)")
	cmd.Flags().StringVarP(&timestamp, "timestamp", "t", "", "Durable Pipeline timestamp (optional)")

	err := cmd.MarkFlagRequired("name")
	if err != nil {
		return nil
	}

	return &cmd
}

var (
	datastoreSyncToCatalogLong = cli.LongDesc(`
		Syncs the entire local datastore to the catalog service. This is used for initial
		migration from file-based to catalog-based datastore management.

		The environment must have catalog configured (datastore type: catalog or all).
	`)

	datastoreSyncToCatalogExample = cli.Examples(`
		# Sync the entire local datastore to catalog
		ccip datastore sync-to-catalog --environment staging
	`)
)

// newDatastoreSyncToCatalog creates a command to sync the entire local datastore to catalog
func (Commands) newDatastoreSyncToCatalog(domain domain.Domain) *cobra.Command {
	cmd := cobra.Command{
		Use:     "sync-to-catalog",
		Short:   "Sync local datastore to catalog",
		Long:    datastoreSyncToCatalogLong,
		Example: datastoreSyncToCatalogExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			envKey, _ := cmd.Flags().GetString("environment")
			envDir := domain.EnvDir(envKey)

			// Load config to get catalog connection details
			cfg, err := config.Load(domain, envKey, logger.Nop())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Verify catalog is configured
			if cfg.DatastoreType != cfgdomain.DatastoreTypeCatalog && cfg.DatastoreType != cfgdomain.DatastoreTypeAll {
				return fmt.Errorf("catalog is not configured for environment %s (datastore type: %s)", envKey, cfg.DatastoreType)
			}

			cmd.Printf("📡 Syncing local datastore to catalog (endpoint: %s)\n", cfg.Env.Catalog.GRPC)

			catalog, catalogErr := cldcatalog.LoadCatalog(ctx, envKey, cfg, domain)
			if catalogErr != nil {
				return fmt.Errorf("failed to load catalog: %w", catalogErr)
			}

			if err := envDir.SyncDataStoreToCatalog(ctx, catalog); err != nil {
				return fmt.Errorf("error syncing datastore to catalog for %s %s: %w",
					domain, envKey, err,
				)
			}

			cmd.Printf("✅ Successfully synced entire datastore to catalog for %s %s\n",
				domain, envKey,
			)

			return nil
		},
	}

	return &cmd
}
