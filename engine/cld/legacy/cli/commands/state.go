package commands

import (
	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/commands/state"
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
//	import "github.com/smartcontractkit/chainlink-deployments-framework/pkg/commands"
//
//	cmds := commands.New(lggr)
//	rootCmd.AddCommand(cmds.State(myDomain, commands.StateConfig{
//	    ViewState: myViewStateFunc,
//	}))
func (c Commands) NewStateCmds(dom domain.Domain, config StateConfig) *cobra.Command {
	return state.NewCommand(state.Config{
		Logger:    c.lggr,
		Domain:    dom,
		ViewState: config.ViewState,
	})
}
