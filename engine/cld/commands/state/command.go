package state

import (
	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
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
// The command requires an environment flag (-e) which is used by all subcommands.
//
// Usage:
//
//	rootCmd.AddCommand(state.NewCommand(state.Config{
//	    Logger:    lggr,
//	    Domain:    myDomain,
//	    ViewState: myViewStateFunc,
//	}))
func NewCommand(cfg Config) *cobra.Command {
	// Apply defaults for optional dependencies
	cfg.deps()

	cmd := &cobra.Command{
		Use:   "state",
		Short: "State commands",
	}

	// Add subcommands
	cmd.AddCommand(newGenerateCmd(cfg))

	// The environment flag is persistent because all subcommands require it.
	// Currently there's only "generate", but this pattern supports future subcommands
	// that also need the environment context.
	cmd.PersistentFlags().
		StringP("environment", "e", "", "Deployment environment (required)")
	_ = cmd.MarkPersistentFlagRequired("environment")

	return cmd
}
