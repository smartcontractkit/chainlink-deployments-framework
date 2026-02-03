package state

import (
	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/legacy/cli"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

var (
	stateShort = "State commands for managing environment state"

	stateLong = cli.LongDesc(`
		Commands for generating and managing deployment state.

		State represents a snapshot of all deployed contracts and their configurations
		for a given environment. Use these commands to generate fresh state from
		on-chain data or manage existing state files.
	`)
)

// Config holds the configuration for state commands.
type Config struct {
	// Logger is the logger to use for command output. Required.
	Logger logger.Logger

	// Domain is the domain context for the commands. Required.
	Domain domain.Domain

	// ViewState is the function that generates state from an environment.
	// This is domain-specific and must be provided by the user.
	ViewState ViewStateFunc

	// Deps holds optional dependencies that can be overridden.
	// If fields are nil, production defaults are used.
	Deps Deps
}

// deps returns the Deps with defaults applied.
func (c *Config) deps() *Deps {
	c.Deps.applyDefaults()

	return &c.Deps
}

// NewCommand creates a new state command with all subcommands.
//
// Usage:
//
//	rootCmd.AddCommand(state.NewCommand(state.Config{
//	    Logger:    lggr,
//	    Domain:    myDomain,
//	    ViewState: myViewStateFunc,
//	}))
func NewCommand(cfg Config) *cobra.Command {
	cfg.deps()

	cmd := &cobra.Command{
		Use:   "state",
		Short: stateShort,
		Long:  stateLong,
	}

	cmd.AddCommand(newGenerateCmd(cfg))

	return cmd
}
