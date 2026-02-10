package commands

import (
	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// NewDatastoreCmds creates a new set of commands for datastore operations.
// This method delegates to the modular datastore package for backward compatibility.
//
// Deprecated: Use the modular commands package for new integrations:
//
//	import "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands"
//
//	cmds := commands.New(lggr)
//	datastoreCmd, err := cmds.Datastore(myDomain)
//	if err != nil {
//	    return err
//	}
//	rootCmd.AddCommand(datastoreCmd)
func (c Commands) NewDatastoreCmds(dom domain.Domain) *cobra.Command {
	cmd, err := datastore.NewCommand(datastore.Config{
		Logger: c.lggr,
		Domain: dom,
	})
	if err != nil {
		panic(err)
	}

	return cmd
}
