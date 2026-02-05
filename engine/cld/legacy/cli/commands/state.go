package commands

import (
	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/state"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// StateConfig holds configuration for state commands.
// Deprecated: Use commands.StateConfig directly for new integrations.
type StateConfig struct {
	ViewState deployment.ViewStateV2
}

// NewStateCmds creates a new set of commands for state environment.
// This method delegates to the modular state package for backward compatibility.
//
// Deprecated: Use the modular commands package for new integrations:
//
//	import "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands"
//
//	cmds := commands.New(lggr)
//	stateCmd, err := cmds.State(myDomain, commands.StateConfig{
//	    ViewState: myViewStateFunc,
//	})
//	if err != nil {
//	    return err
//	}
//	rootCmd.AddCommand(stateCmd)
func (c Commands) NewStateCmds(dom domain.Domain, config StateConfig) *cobra.Command {
	cmd, err := state.NewCommand(state.Config{
		Logger:    c.lggr,
		Domain:    dom,
		ViewState: config.ViewState,
	})
	if err != nil {
		// Return a command that errors on execution to maintain backward compatibility.
		// The new API (state.NewCommand) returns error directly for proper handling.
		return &cobra.Command{
			Use:   "state",
			Short: "State commands (misconfigured)",
			RunE: func(_ *cobra.Command, _ []string) error {
				return err
			},
		}
	}

	return cmd
}
