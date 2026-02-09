package commands

import (
	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/addressbook"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// NewAddressBookCmds creates a new set of commands for address book operations.
// This method delegates to the modular addressbook package for backward compatibility.
//
// Deprecated: Use the modular commands package for new integrations:
//
//	import "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands"
//
//	cmds := commands.New(lggr)
//	addressBookCmd, err := cmds.AddressBook(myDomain)
//	if err != nil {
//	    return err
//	}
//	rootCmd.AddCommand(addressBookCmd)
func (c Commands) NewAddressBookCmds(dom domain.Domain) *cobra.Command {
	cmd, err := addressbook.NewCommand(addressbook.Config{
		Logger: c.lggr,
		Domain: dom,
	})
	if err != nil {
		// Return an error command that surfaces the configuration error on any invocation.
		// PersistentPreRunE ensures subcommands also return the real error.
		// RunE handles direct invocation of the root command.
		errCmd := &cobra.Command{
			Use:   "address-book",
			Short: "Address book operations (misconfigured)",
			RunE: func(_ *cobra.Command, _ []string) error {
				return err
			},
		}
		errCmd.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
			return err
		}

		return errCmd
	}

	return cmd
}
