package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldcatalog "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/catalog"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	cfgdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/legacy/cli"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	foperations "github.com/smartcontractkit/chainlink-deployments-framework/operations"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// LoadRegistryFunc is a function type that loads the migrations registry for a given environment key.
type LoadRegistryFunc func(envKey string) (*changeset.ChangesetsRegistry, error)

// DecodeProposalCtxProvider is a function type that adds decoding context based on the environment.
type DecodeProposalCtxProvider func(env fdeployment.Environment) (analyzer.ProposalContext, error)

// NewMigrationCmds creates a new set of commands for managing migrations.
func (c Commands) NewMigrationCmds(
	domain domain.Domain,
	loadFunc LoadRegistryFunc,
	decodeProposalContext DecodeProposalCtxProvider,
) *cobra.Command {
	migrationsCmd := &cobra.Command{
		Use:   "migration",
		Short: "Migration commands",
	}

	addressBookCmd := &cobra.Command{
		Use:   "address-book",
		Short: "Address book operations",
	}
	addressBookCmd.AddCommand(c.newMigrationAddressBookMerge(domain))
	addressBookCmd.AddCommand(c.newMigrationAddressBookMigrate(domain))
	addressBookCmd.AddCommand(c.newMigrationAddressBookRemove(domain))

	datastoreCmd := &cobra.Command{
		Use:   "datastore",
		Short: "Datastore operations",
	}
	datastoreCmd.AddCommand(c.newMigrationDataStoreMerge(domain))
	datastoreCmd.AddCommand(c.newMigrationDataStoreSyncToCatalog(domain))

	migrationsCmd.AddCommand(
		c.newMigrationRun(domain, loadFunc, decodeProposalContext),
		c.newMigrationList(loadFunc),
		c.newMigrationLatest(loadFunc),
		addressBookCmd,
		datastoreCmd,
	)

	migrationsCmd.PersistentFlags().StringP("environment", "e", "", "Deployment environment (required)")
	err := migrationsCmd.MarkPersistentFlagRequired("environment")
	if err != nil {
		return nil
	}

	return migrationsCmd
}

var (
	migrationRunLong = cli.LongDesc(`
		Run a specific migration for a given Domain Environment.
		This will execute the migration’s changeset, manage artifact directories,
		and optionally force-reapply or do a dry-run.
	`)

	migrationRunExample = cli.Examples(`
		# Apply migration 0001_deploy_cap in the staging environment
		ccip migration run --environment staging --name 0001_deploy_cap

		# Force reapply the same migration
		ccip migration run --environment staging --name 0001_deploy_cap --force

		# Dry-run the migration without writing artifacts
		ccip migration run -e staging -n 0001_deploy_cap --dry-run
	`)
)

func (c Commands) newMigrationRun(
	domain domain.Domain,
	loadMigration func(envName string) (*changeset.ChangesetsRegistry, error),
	decodeProposalContext func(env fdeployment.Environment) (analyzer.ProposalContext, error),
) *cobra.Command {
	var (
		migrationName string
		force         bool
		dryRun        bool
	)

	cmd := cobra.Command{
		Use:     "run",
		Short:   "Run a migration",
		Long:    migrationRunLong,
		Example: migrationRunExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			envDir := domain.EnvDir(envKey)
			artifactsDir := envDir.ArtifactsDir()

			// Check if migration artifacts already exist.
			exists, err := artifactsDir.MigrationDirExists(migrationName)
			if err != nil {
				return fmt.Errorf("failed to check if migration artifacts dir exists: %w", err)
			}

			// If so, return an error.
			if exists && !force {
				cmd.Printf("Migration artifacts already exist for %s in %s for %s\n If you want to reapply the migration, delete the migration artifacts directory or use the --force flag. Exiting.",
					domain, envKey, migrationName,
				)

				return fmt.Errorf("cannot apply migration %s in %s because migration artifacts already exist",
					migrationName, envKey,
				)
			}

			if force {
				cmd.Println("Force flag set, removing existing migration artifacts")

				if err = artifactsDir.RemoveMigrationDir(migrationName); err != nil {
					return fmt.Errorf("failed to remove migration artifacts: %w", err)
				}
			}

			migration, err := loadMigration(envKey)
			if err != nil {
				return err
			}

			envOptions, err := configureEnvironmentOptions(migration, migrationName, dryRun, c.lggr)
			if err != nil {
				return err
			}

			reports, err := artifactsDir.LoadOperationsReports(migrationName)
			if err != nil {
				return fmt.Errorf("failed to load operations report: %w", err)
			}
			originalReportsLen := len(reports)
			cmd.Printf("Loaded %d operations reports", originalReportsLen)
			reporter := foperations.NewMemoryReporter(foperations.WithReports(reports))

			envOptions = append(envOptions, environment.WithReporter(reporter))
			env, err := environment.Load(cmd.Context(), domain, envKey, envOptions...)
			if err != nil {
				return err
			}

			// We create the directory even before the attempt to run the migrations.
			// Some migrations may execute ChangeSet functions that only have side effects but not artifacts.
			// In that case we still want to create the directory. We include a .gitkeep file to ensure
			// the directory is not empty.
			cmd.Printf("Applying %s migration %s for environment: %s\n",
				domain, migrationName, envKey,
			)

			if err = artifactsDir.CreateMigrationDir(migrationName); err != nil {
				return fmt.Errorf("failed to create .gitkeep file %w", err)
			}

			out, err := migration.Apply(migrationName, env)
			// save reports first then handle above error
			if saveErr := saveReports(reporter, originalReportsLen, c.lggr, artifactsDir, migrationName); saveErr != nil {
				return saveErr
			}
			if err != nil {
				return fmt.Errorf("failed to run migration: %w", err)
			}

			if len(out.DescribedTimelockProposals) == 0 && decodeProposalContext != nil {
				out.DescribedTimelockProposals = make([]string, len(out.MCMSTimelockProposals))
				proposalContext, err := decodeProposalContext(env)
				if err != nil {
					return err
				}
				for idx, proposal := range out.MCMSTimelockProposals {
					describedProposal, err := analyzer.DescribeTimelockProposal(cmd.Context(), proposalContext, env, &proposal)
					if err != nil {
						cmd.PrintErrf("failed to describe time lock proposal %d: %v\n", idx, err)
						continue
					}
					out.DescribedTimelockProposals[idx] = describedProposal
				}
			}

			if err := artifactsDir.SaveChangesetOutput(migrationName, out); err != nil {
				return fmt.Errorf("failed to save migration artifacts: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force apply migration, removing existing artifacts")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Use a read-only JD backend. WARNING: still uses real chain clients as of now!")

	cmd.Flags().StringVarP(&migrationName, "changeset", "c", "", "changeset (deprecated, use \"name\")")
	cmd.Flags().StringVarP(&migrationName, "name", "n", "", "name (required)")
	cmd.MarkFlagsMutuallyExclusive("changeset", "name")
	cmd.MarkFlagsOneRequired("changeset", "name")

	return &cmd
}

// getChainOverrides retrieves the chain overrides for a given migration.
// It first checks for migration options, and if not found, it retrieves input chain overrides.
func getChainOverrides(changeset *changeset.ChangesetsRegistry, migrationStr string) ([]uint64, error) {
	migrationOptions, err := changeset.GetChangesetOptions(migrationStr)
	if err != nil {
		return nil, err
	}

	if migrationOptions.ChainsToLoad != nil {
		return migrationOptions.ChainsToLoad, nil
	}

	// this is only applicable to durable pipelines
	configs, err := changeset.GetConfigurations(migrationStr)
	if err != nil {
		return nil, err
	}

	return configs.InputChainOverrides, nil
}

var (
	migrationListLong = cli.LongDesc(`
		Lists the migration keys that have been registered on the MigrationsRegistry for a given
		Domain Environment.
	`)

	migrationListExample = cli.Examples(`
		# List all migration keys for the ccip staging domain
		ccip migration list --environment staging

		# List all migration keys for the keystone production domain
		keystone migration list --environment production
	`)
)

// newMigrationList creates a command to list migration keys for a given domain environment.
func (Commands) newMigrationList(loadFunc LoadRegistryFunc) *cobra.Command {
	cmd := cobra.Command{
		Use:     "list",
		Short:   "Lists migration keys for a domain environment",
		Long:    migrationListLong,
		Example: migrationListExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")

			registry, err := loadFunc(envKey)
			if err != nil {
				return fmt.Errorf("failed to load registry: %w", err)
			}

			for _, k := range registry.ListKeys() {
				cmd.Println(k)
			}

			return nil
		},
	}

	return &cmd
}

var (
	migrationLatestLong = cli.LongDesc(`
		Gets the latest migration key registered on the MigrationsRegistry for a given Domain Environment.
	`)

	migrationLatestExample = cli.Examples(`
		# Get the latest migration key for the ccip staging domain
		ccip migration latest --environment staging

		# Get the latest migration key for the keystone production domain
		keystone migration latest --e production
	`)
)

// newMigrationLatest creates a command to get the latest migration key for a given environment.
func (Commands) newMigrationLatest(loadFunc LoadRegistryFunc) *cobra.Command {
	cmd := cobra.Command{
		Use:     "latest",
		Short:   "Get latest migration key for a domain environment",
		Long:    migrationLatestLong,
		Example: migrationLatestExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")

			registry, err := loadFunc(envKey)
			if err != nil {
				return fmt.Errorf("failed to load registry: %w", err)
			}

			key, err := registry.LatestKey()
			if err != nil {
				return fmt.Errorf("failed to get latest migration key: %w", err)
			}

			// Caught by CI script, must be fmt instead of cmd.Printf to be properly captured
			fmt.Println(key)

			return nil
		},
	}

	return &cmd
}

var (
	migrationMergeAddressBookLong = cli.LongDesc(`
		Merges the address book artifact of a specific migration to the main address book within a
		given Domain Environment. This is to ensure that the address book is up-to-date with the
		latest migration changes.
	`)

	migrationMergeAddressBookExample = cli.Examples(`
		# Merge the address book for the 0001_deploy_cap migration in the ccip staging domain environment
		ccip migration address-book merge --environment staging --name 0001_deploy_cap
	`)
)

// newMigrationAddressBookMerge creates a command to merge the address books for a migration to
// the main address book within a given domain environment.
func (Commands) newMigrationAddressBookMerge(domain domain.Domain) *cobra.Command {
	var (
		migrationName string
		timestamp     string
	)

	cmd := cobra.Command{
		Use:     "merge",
		Short:   "Merge the address book for a migration to the main address book",
		Long:    migrationMergeAddressBookLong,
		Example: migrationMergeAddressBookExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			envDir := domain.EnvDir(envKey)

			if err := envDir.MergeMigrationAddressBook(migrationName, timestamp); err != nil {
				return fmt.Errorf("error during address book merge for %s %s %s: %w",
					domain, envKey, migrationName, err,
				)
			}

			cmd.Printf("Merged address books for %s %s %s",
				domain, envKey, migrationName,
			)

			return nil
		},
	}

	cmd.Flags().StringVarP(&migrationName, "name", "n", "", "name (required)")
	cmd.Flags().StringVarP(&timestamp, "timestamp", "t", "", "Durable Pipeline timestamp (optional)")

	err := cmd.MarkFlagRequired("name")
	if err != nil {
		return nil
	}

	return &cmd
}

var (
	migrationMigrateAddressBookLong = cli.LongDesc(`
		Converts the address book artifact format to the new datastore schema within a
		given Domain Environment. This updates your on-chain address book to the latest storage format.
	`)

	migrationMigrateAddressBookExample = cli.Examples(`
		# Migrate the address book for the ccip staging domain to the new datastore format
		ccip migration address-book migrate --environment staging
	`)
)

// newMigrationsAddressBookMigrate creates a command to convert the address book
// artifact to the new datastore format within a given domain environment.
func (Commands) newMigrationAddressBookMigrate(domain domain.Domain) *cobra.Command {
	cmd := cobra.Command{
		Use:     "migrate",
		Short:   "Migrate address book to the new datastore format",
		Long:    migrationMigrateAddressBookLong,
		Example: migrationMigrateAddressBookExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			envDir := domain.EnvDir(envKey)

			if err := envDir.MigrateAddressBook(); err != nil {
				return fmt.Errorf("error during address book conversion for %s %s: %w",
					domain, envKey, err,
				)
			}

			cmd.Printf("Address book for %s %s successfully migrated to the new datastore format",
				domain, envKey,
			)

			return nil
		},
	}

	return &cmd
}

var (
	migrationAddressBookRemoveLong = cli.LongDesc(`
		Removes the address book entries introduced by a specific migration from the main
		address book within a given Domain Environment. This can be used to rollback
		address-book merge changes.
	`)

	migrationAddressBookRemove = cli.Examples(`
		# Remove the address book entries for the 0001_deploy_cap migration in the ccip staging domain
		ccip migration address-book remove --environment staging --name 0001_deploy_cap
	`)
)

// newMigrationsAddressBookRemove creates a command to remove a migration's
// address book entries from the main address book within a given domain environment.
func (Commands) newMigrationAddressBookRemove(domain domain.Domain) *cobra.Command {
	var (
		migrationName string
		timestamp     string
	)

	cmd := cobra.Command{
		Use:     "remove",
		Short:   "Remove migration address book",
		Long:    migrationAddressBookRemoveLong,
		Example: migrationAddressBookRemove,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			envDir := domain.EnvDir(envKey)

			if err := envDir.RemoveMigrationAddressBook(migrationName, timestamp); err != nil {
				return fmt.Errorf("error during address book remove for %s %s %s: %w",
					domain, envKey, migrationName, err,
				)
			}

			cmd.Printf("Removed address books for %s %s %s",
				domain, envKey, migrationName,
			)

			return nil
		},
	}

	cmd.Flags().StringVarP(&migrationName, "changeset", "c", "", "changeset (deprecated, use \"name\")")
	cmd.Flags().StringVarP(&migrationName, "name", "n", "", "name(required)")
	cmd.Flags().StringVarP(&timestamp, "timestamp", "t", "", "Durable Pipeline timestamp (optional)")
	cmd.MarkFlagsMutuallyExclusive("changeset", "name")
	cmd.MarkFlagsOneRequired("changeset", "name")

	return &cmd
}

var (
	migrationDataStoreMergeExample = cli.Examples(`
		# Merge the data store for the 0001_deploy_cap migration in the ccip staging domain
		ccip migration datastore merge --environment staging --name 0001_deploy_cap
	`)
)

// newMigrationsDataStoreMerge creates a command to merge the data store for a migration
func (Commands) newMigrationDataStoreMerge(domain domain.Domain) *cobra.Command {
	var (
		migrationName string
		timestamp     string
	)

	cmd := cobra.Command{
		Use:     "merge",
		Short:   "Merge data stores",
		Long:    "Merge the data store for a migration to the main data store",
		Example: migrationDataStoreMergeExample,
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

				if err := envDir.MergeMigrationDataStoreCatalog(cmd.Context(), migrationName, timestamp, catalog); err != nil {
					return fmt.Errorf("error during data store merge to catalog for %s %s %s: %w",
						domain, envKey, migrationName, err,
					)
				}

				cmd.Printf("✅ Merged data stores to catalog for %s %s %s\n",
					domain, envKey, migrationName,
				)
			case cfgdomain.DatastoreTypeFile:
				// File mode - merge to local files
				cmd.Printf("📁 Using file-based datastore mode\n")

				if err := envDir.MergeMigrationDataStore(migrationName, timestamp); err != nil {
					return fmt.Errorf("error during data store merge to file for %s %s %s: %w",
						domain, envKey, migrationName, err,
					)
				}

				cmd.Printf("✅ Merged data stores to local files for %s %s %s\n",
					domain, envKey, migrationName,
				)
			case cfgdomain.DatastoreTypeAll:
				// All mode - merge to both catalog and local files
				cmd.Printf("📡 Using all datastore mode (catalog: %s, file: %s)\n", cfg.Env.Catalog.GRPC, envDir.DataStoreDirPath())

				catalog, catalogErr := cldcatalog.LoadCatalog(cmd.Context(), envKey, cfg, domain)
				if catalogErr != nil {
					return fmt.Errorf("failed to load catalog: %w", catalogErr)
				}

				if err := envDir.MergeMigrationDataStoreCatalog(cmd.Context(), migrationName, timestamp, catalog); err != nil {
					return fmt.Errorf("error during data store merge to catalog for %s %s %s: %w",
						domain, envKey, migrationName, err,
					)
				}

				if err := envDir.MergeMigrationDataStore(migrationName, timestamp); err != nil {
					return fmt.Errorf("error during data store merge to file for %s %s %s: %w",
						domain, envKey, migrationName, err,
					)
				}

				cmd.Printf("✅ Merged data stores to both catalog and local files for %s %s %s\n",
					domain, envKey, migrationName,
				)
			default:
				return fmt.Errorf("invalid datastore type: %s", cfg.DatastoreType)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&migrationName, "changeset", "c", "", "changeset (deprecated, use \"name\")")
	cmd.Flags().StringVarP(&migrationName, "name", "n", "", "name (required)")
	cmd.Flags().StringVarP(&timestamp, "timestamp", "t", "", "Durable Pipeline timestamp (optional)")
	cmd.MarkFlagsMutuallyExclusive("changeset", "name")
	cmd.MarkFlagsOneRequired("changeset", "name")

	return &cmd
}

var (
	migrationDataStoreSyncToCatalogExample = cli.Examples(`
		# Sync the entire local datastore to catalog for initial migration
		ccip migration datastore sync-to-catalog --environment staging
	`)
)

// newMigrationDataStoreSyncToCatalog creates a command to sync the entire local datastore to catalog
func (Commands) newMigrationDataStoreSyncToCatalog(domain domain.Domain) *cobra.Command {
	cmd := cobra.Command{
		Use:     "sync-to-catalog",
		Short:   "Sync local datastore to catalog",
		Long:    "Sync the entire local datastore to the catalog service. This is used for initial migration from file-based to catalog-based datastore.",
		Example: migrationDataStoreSyncToCatalogExample,
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
