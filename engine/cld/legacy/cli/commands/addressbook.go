package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/legacy/cli"
)

// NewAddressBookCmds creates a new set of commands for address book operations.
func (c Commands) NewAddressBookCmds(domain domain.Domain) *cobra.Command {
	addressBookCmd := &cobra.Command{
		Use:   "address-book",
		Short: "Address book operations",
	}

	addressBookCmd.AddCommand(c.newAddressBookMerge(domain))
	addressBookCmd.AddCommand(c.newAddressBookMigrate(domain))
	addressBookCmd.AddCommand(c.newAddressBookRemove(domain))

	addressBookCmd.PersistentFlags().StringP("environment", "e", "", "Deployment environment (required)")
	err := addressBookCmd.MarkPersistentFlagRequired("environment")
	if err != nil {
		return nil
	}

	return addressBookCmd
}

var (
	addressBookMergeLong = cli.LongDesc(`
		Merges the address book artifact of a specific changeset to the main address book within a
		given Domain Environment. This is to ensure that the address book is up-to-date with the
		latest changeset changes.
	`)

	addressBookMergeExample = cli.Examples(`
		# Merge the address book for the 0001_deploy_cap changeset in the ccip staging domain environment
		ccip address-book merge --environment staging --name 0001_deploy_cap

		# Merge with a specific durable pipeline timestamp
		ccip address-book merge --environment staging --name 0001_deploy_cap --timestamp 1234567890
	`)
)

// newAddressBookMerge creates a command to merge the address books for a changeset to
// the main address book within a given domain environment.
func (Commands) newAddressBookMerge(domain domain.Domain) *cobra.Command {
	var (
		name      string
		timestamp string
	)

	cmd := cobra.Command{
		Use:     "merge",
		Short:   "Merge the address book for a changeset to the main address book",
		Long:    addressBookMergeLong,
		Example: addressBookMergeExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			envDir := domain.EnvDir(envKey)

			if err := envDir.MergeMigrationAddressBook(name, timestamp); err != nil {
				return fmt.Errorf("error during address book merge for %s %s %s: %w",
					domain, envKey, name, err,
				)
			}

			cmd.Printf("Merged address books for %s %s %s\n",
				domain, envKey, name,
			)

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
	addressBookMigrateLong = cli.LongDesc(`
		Converts the address book artifact format to the new datastore schema within a
		given Domain Environment. This updates your on-chain address book to the latest storage format.
	`)

	addressBookMigrateExample = cli.Examples(`
		# Migrate the address book for the ccip staging domain to the new datastore format
		ccip address-book migrate --environment staging
	`)
)

// newAddressBookMigrate creates a command to convert the address book
// artifact to the new datastore format within a given domain environment.
func (Commands) newAddressBookMigrate(domain domain.Domain) *cobra.Command {
	cmd := cobra.Command{
		Use:     "migrate",
		Short:   "Migrate address book to the new datastore format",
		Long:    addressBookMigrateLong,
		Example: addressBookMigrateExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			envDir := domain.EnvDir(envKey)

			if err := envDir.MigrateAddressBook(); err != nil {
				return fmt.Errorf("error during address book conversion for %s %s: %w",
					domain, envKey, err,
				)
			}

			cmd.Printf("Address book for %s %s successfully migrated to the new datastore format\n",
				domain, envKey,
			)

			return nil
		},
	}

	return &cmd
}

var (
	addressBookRemoveLong = cli.LongDesc(`
		Removes the address book entries introduced by a specific changeset from the main
		address book within a given Domain Environment. This can be used to rollback
		address-book merge changes.
	`)

	addressBookRemoveExample = cli.Examples(`
		# Remove the address book entries for the 0001_deploy_cap changeset in the ccip staging domain
		ccip address-book remove --environment staging --name 0001_deploy_cap
	`)
)

// newAddressBookRemove creates a command to remove a changeset's
// address book entries from the main address book within a given domain environment.
func (Commands) newAddressBookRemove(domain domain.Domain) *cobra.Command {
	var (
		name      string
		timestamp string
	)

	cmd := cobra.Command{
		Use:     "remove",
		Short:   "Remove changeset address book entries",
		Long:    addressBookRemoveLong,
		Example: addressBookRemoveExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			envDir := domain.EnvDir(envKey)

			if err := envDir.RemoveMigrationAddressBook(name, timestamp); err != nil {
				return fmt.Errorf("error during address book remove for %s %s %s: %w",
					domain, envKey, name, err,
				)
			}

			cmd.Printf("Removed address books for %s %s %s\n",
				domain, envKey, name,
			)

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
