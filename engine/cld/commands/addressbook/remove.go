package addressbook

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/legacy/cli"
)

var (
	removeShort = "Remove changeset address book entries"

	removeLong = cli.LongDesc(`
		Removes the address book entries introduced by a specific changeset from the main
		address book within a given Domain Environment. This can be used to rollback
		address-book merge changes.
	`)

	removeExample = cli.Examples(`
		# Remove the address book entries for the 0001_deploy_cap changeset in the ccip staging domain
		ccip address-book remove --environment staging --name 0001_deploy_cap
	`)
)

type removeFlags struct {
	environment string
	name        string
	timestamp   string
}

// newRemoveCmd creates the "remove" subcommand for removing address book entries.
func newRemoveCmd(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   removeShort,
		Long:    removeLong,
		Example: removeExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			f := removeFlags{
				environment: flags.MustString(cmd.Flags().GetString("environment")),
				name:        flags.MustString(cmd.Flags().GetString("name")),
				timestamp:   flags.MustString(cmd.Flags().GetString("timestamp")),
			}

			return runRemove(cmd, cfg, f)
		},
	}

	// Shared flags
	flags.Environment(cmd)

	// Local flags specific to this command
	cmd.Flags().StringP("name", "n", "", "Changeset name (required)")
	cmd.Flags().StringP("timestamp", "t", "", "Durable Pipeline timestamp (optional)")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

// runRemove executes the remove command logic.
func runRemove(cmd *cobra.Command, cfg Config, f removeFlags) error {
	deps := cfg.deps()
	envDir := cfg.Domain.EnvDir(f.environment)

	if err := deps.AddressBookRemover(envDir, f.name, f.timestamp); err != nil {
		return fmt.Errorf("error during address book remove for %s %s %s: %w",
			cfg.Domain, f.environment, f.name, err,
		)
	}

	cmd.Printf("✅ Removed address book entries for %s %s %s\n",
		cfg.Domain, f.environment, f.name,
	)

	return nil
}
