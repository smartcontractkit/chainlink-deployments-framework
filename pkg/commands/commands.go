// Package commands provides modular CLI command packages for domain CLIs.
//
// There are two ways to use commands from this package:
//
// 1. Via the Commands factory (recommended for most use cases):
//
//	commands := commands.New(lggr)
//	app.AddCommand(
//	    commands.State(domain, stateConfig),
//	    commands.EVM(domain),
//	    commands.JD(domain),
//	)
//
// 2. Via direct package imports (for advanced DI/testing):
//
//	import "github.com/smartcontractkit/chainlink-deployments-framework/pkg/commands/state"
//
//	app.AddCommand(state.NewCommand(state.Config{
//	    Logger:    lggr,
//	    Domain:    domain,
//	    ViewState: myViewState,
//	    Deps:      &state.Deps{...},  // inject mocks for testing
//	}))
package commands

import (
	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/commands/state"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// Commands provides a factory for creating CLI commands with shared configuration.
// This allows setting the logger once and reusing it across all commands.
type Commands struct {
	lggr logger.Logger
}

// New creates a new Commands factory with the given logger.
// The logger will be shared across all commands created by this factory.
func New(lggr logger.Logger) *Commands {
	return &Commands{lggr: lggr}
}

// StateConfig holds configuration for state commands.
type StateConfig struct {
	// ViewState is the function that generates state from an environment.
	// This is domain-specific and must be provided by the user.
	ViewState state.ViewStateFunc
}

// State creates the state command group for managing environment state.
//
// Usage:
//
//	cmds := commands.New(lggr)
//	rootCmd.AddCommand(cmds.State(domain, commands.StateConfig{
//	    ViewState: myViewStateFunc,
//	}))
func (c *Commands) State(dom domain.Domain, cfg StateConfig) *cobra.Command {
	return state.NewCommand(state.Config{
		Logger:    c.lggr,
		Domain:    dom,
		ViewState: cfg.ViewState,
	})
}
