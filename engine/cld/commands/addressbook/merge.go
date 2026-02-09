package addressbook

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/legacy/cli"
)

var (
	mergeShort = "Merge the address book for a changeset to the main address book"

	mergeLong = cli.LongDesc(`
		Merges the address book artifact of a specific changeset to the main address book within a
		given Domain Environment. This is to ensure that the address book is up-to-date with the
		latest changeset changes.
	`)

	mergeExample = cli.Examples(`
		# Merge the address book for the 0001_deploy_cap changeset in the ccip staging domain environment
		ccip address-book merge --environment staging --name 0001_deploy_cap

		# Merge with a specific durable pipeline timestamp
		ccip address-book merge --environment staging --name 0001_deploy_cap --timestamp 1234567890
	`)
)

type mergeFlags struct {
	environment string
	name        string
	timestamp   string
}

// newMergeCmd creates the "merge" subcommand for merging address book artifacts.
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
	cmd.Flags().StringP("timestamp", "t", "", "Durable Pipeline timestamp (optional)")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

// runMerge executes the merge command logic.
func runMerge(cmd *cobra.Command, cfg Config, f mergeFlags) error {
	deps := cfg.deps()
	envDir := cfg.Domain.EnvDir(f.environment)

	if err := deps.AddressBookMerger(envDir, f.name, f.timestamp); err != nil {
		return fmt.Errorf("error during address book merge for %s %s %s: %w",
			cfg.Domain, f.environment, f.name, err,
		)
	}

	cmd.Printf("✅ Merged address book for %s %s %s\n",
		cfg.Domain, f.environment, f.name,
	)

	return nil
}
