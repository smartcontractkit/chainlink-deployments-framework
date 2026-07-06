package addressbook

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/text"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
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

		# Add missing address book entries without removing existing address refs
		ccip address-book migrate --environment testnet --preserve-existing

		# Migrate only Sui testnet addresses
		ccip address-book migrate --environment testnet --selector 13264668187771770619 --preserve-existing
	`)
)

type migrateFlags struct {
	environment      string
	preserveExisting bool
	chainSelector    uint64
}

// newMigrateCmd creates the "migrate" subcommand for migrating address book to datastore format.
func newMigrateCmd(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "migrate",
		Short:   migrateShort,
		Long:    migrateLong,
		Example: migrateExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			preserveExisting, err := cmd.Flags().GetBool("preserve-existing")
			if err != nil {
				return err
			}

			f := migrateFlags{
				environment:      flags.MustString(cmd.Flags().GetString("environment")),
				preserveExisting: preserveExisting,
				chainSelector:    flags.MustUint64(cmd.Flags().GetUint64("selector")),
			}

			return runMigrate(cmd, cfg, f)
		},
	}

	// Shared flags
	flags.Environment(cmd)
	flags.ChainSelector(cmd, false)
	cmd.Flags().Bool(
		"preserve-existing",
		false,
		"Keep existing address refs and only add address book entries that are not already present",
	)

	return cmd
}

// runMigrate executes the migrate command logic.
func runMigrate(cmd *cobra.Command, cfg Config, f migrateFlags) error {
	deps := cfg.deps()
	envDir := cfg.Domain.EnvDir(f.environment)

	if err := deps.AddressBookMigrator(envDir, domain.MigrateAddressBookOptions{
		PreserveExisting: f.preserveExisting,
		ChainSelector:    f.chainSelector,
	}); err != nil {
		return fmt.Errorf("error during address book migration for %s %s: %w",
			cfg.Domain, f.environment, err,
		)
	}

	cmd.Printf("%s\n", migrateSuccessMessage(cfg.Domain.String(), f.environment, f))

	return nil
}

func migrateSuccessMessage(domain, environment string, f migrateFlags) string {
	switch {
	case f.preserveExisting && f.chainSelector != 0:
		return fmt.Sprintf(
			"✅ Added missing address book entries for chain selector %d to address refs in %s %s without removing existing entries",
			f.chainSelector, domain, environment,
		)
	case f.preserveExisting:
		return fmt.Sprintf(
			"✅ Added missing address book entries to address refs in %s %s without removing existing entries",
			domain, environment,
		)
	case f.chainSelector != 0:
		return fmt.Sprintf(
			"✅ Replaced address refs for chain selector %d in %s %s from the address book",
			f.chainSelector, domain, environment,
		)
	default:
		return fmt.Sprintf(
			"✅ Address book for %s %s successfully migrated to the new datastore format",
			domain, environment,
		)
	}
}
