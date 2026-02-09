package datastore

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/text"
	cfgdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/domain"
)

var (
	mergeShort = "Merge datastore artifacts"

	mergeLong = text.LongDesc(`
		Merges the datastore artifact of a specific changeset to the main datastore within a
		given Domain Environment. The merge destination depends on the datastore configuration:
		- file: merges to local JSON files
		- catalog: merges to the remote catalog service
		- all: merges to both local files and catalog
	`)

	mergeExample = text.Examples(`
		# Merge the datastore for the 0001_deploy_cap changeset in the ccip staging domain
		ccip datastore merge --environment staging --name 0001_deploy_cap

		# Merge with a specific durable pipeline timestamp
		ccip datastore merge --environment staging --name 0001_deploy_cap --timestamp 1234567890
	`)
)

type mergeFlags struct {
	environment string
	name        string
	timestamp   string
}

// newMergeCmd creates the "merge" subcommand for merging datastore artifacts.
func newMergeCmd(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "merge",
		Short:   mergeShort,
		Long:    mergeLong,
		Example: mergeExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			f := mergeFlags{
				environment: flags.MustString(cmd.Flags().GetString("environment")),
				name:        flags.MustString(cmd.Flags().GetString("name")),
				timestamp:   flags.MustString(cmd.Flags().GetString("timestamp")),
			}

			return runMerge(cmd, cfg, f)
		},
	}

	// Shared flags
	flags.Environment(cmd)

	// Local flags specific to this command
	cmd.Flags().StringP("name", "n", "", "Changeset name (required)")
	cmd.Flags().StringP("timestamp", "t", "", "Pipeline timestamp (optional)")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

// runMerge executes the merge command logic.
func runMerge(cmd *cobra.Command, cfg Config, f mergeFlags) error {
	ctx := cmd.Context()
	deps := cfg.deps()
	envDir := cfg.Domain.EnvDir(f.environment)

	// --- Load

	// Load config to check datastore type
	envCfg, err := deps.ConfigLoader(cfg.Domain, f.environment, cfg.Logger)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// --- Execute

	switch envCfg.DatastoreType {
	case cfgdomain.DatastoreTypeCatalog:
		cmd.Printf("📡 Using catalog datastore mode (endpoint: %s)\n", envCfg.Env.Catalog.GRPC)

		catalog, catalogErr := deps.CatalogLoader(ctx, f.environment, envCfg, cfg.Domain)
		if catalogErr != nil {
			return fmt.Errorf("failed to load catalog: %w", catalogErr)
		}

		if err := deps.CatalogMerger(ctx, envDir, f.name, f.timestamp, catalog); err != nil {
			return fmt.Errorf("error during datastore merge to catalog for %s %s %s: %w",
				cfg.Domain, f.environment, f.name, err,
			)
		}

		cmd.Printf("✅ Merged datastore to catalog for %s %s %s\n",
			cfg.Domain, f.environment, f.name,
		)

	case cfgdomain.DatastoreTypeFile:
		cmd.Printf("📁 Using file-based datastore mode\n")

		if err := deps.FileMerger(envDir, f.name, f.timestamp); err != nil {
			return fmt.Errorf("error during datastore merge to file for %s %s %s: %w",
				cfg.Domain, f.environment, f.name, err,
			)
		}

		cmd.Printf("✅ Merged datastore to local files for %s %s %s\n",
			cfg.Domain, f.environment, f.name,
		)

	case cfgdomain.DatastoreTypeAll:
		cmd.Printf("📡 Using all datastore mode (catalog: %s, file: %s)\n",
			envCfg.Env.Catalog.GRPC, envDir.DataStoreDirPath())

		catalog, catalogErr := deps.CatalogLoader(ctx, f.environment, envCfg, cfg.Domain)
		if catalogErr != nil {
			return fmt.Errorf("failed to load catalog: %w", catalogErr)
		}

		if err := deps.CatalogMerger(ctx, envDir, f.name, f.timestamp, catalog); err != nil {
			return fmt.Errorf("error during datastore merge to catalog for %s %s %s: %w",
				cfg.Domain, f.environment, f.name, err,
			)
		}

		if err := deps.FileMerger(envDir, f.name, f.timestamp); err != nil {
			return fmt.Errorf("error during datastore merge to file for %s %s %s: %w",
				cfg.Domain, f.environment, f.name, err,
			)
		}

		cmd.Printf("✅ Merged datastore to both catalog and local files for %s %s %s\n",
			cfg.Domain, f.environment, f.name,
		)

	default:
		return fmt.Errorf("invalid datastore type: %s", envCfg.DatastoreType)
	}

	return nil
}
