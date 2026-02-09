package addressbook

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/text"
)

var (
	migrateShort = "Migrate address book to the new datastore format"

	migrateLong = text.LongDesc(`
		Converts the address book artifact format to the new datastore schema within a
		given Domain Environment. This updates your on-chain address book to the latest storage format.
	`)

	migrateExample = text.Examples(`
		# Migrate the address book for the ccip staging domain to the new datastore format
		ccip address-book migrate --environment staging
	`)
)

type migrateFlags struct {
	environment string
}

// newMigrateCmd creates the "migrate" subcommand for migrating address book to datastore format.
func newMigrateCmd(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "migrate",
		Short:   migrateShort,
		Long:    migrateLong,
		Example: migrateExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			f := migrateFlags{
				environment: flags.MustString(cmd.Flags().GetString("environment")),
			}

			return runMigrate(cmd, cfg, f)
		},
	}

	// Shared flags
	flags.Environment(cmd)

	return cmd
}

// runMigrate executes the migrate command logic.
func runMigrate(cmd *cobra.Command, cfg Config, f migrateFlags) error {
	deps := cfg.deps()
	envDir := cfg.Domain.EnvDir(f.environment)

	if err := deps.AddressBookMigrator(envDir); err != nil {
		return fmt.Errorf("error during address book migration for %s %s: %w",
			cfg.Domain, f.environment, err,
		)
	}

	cmd.Printf("✅ Address book for %s %s successfully migrated to the new datastore format\n",
		cfg.Domain, f.environment,
	)

	return nil
}
