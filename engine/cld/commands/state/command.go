package state

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/text"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

var (
	stateShort = "State commands for managing environment state"

	stateLong = text.LongDesc(`
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

// Validate checks that all required configuration fields are set.
// Returns an error describing which fields are missing.
func (c Config) Validate() error {
	var missing []string

	if c.Logger == nil {
		missing = append(missing, "Logger")
	}
	if c.Domain.RootPath() == "" {
		missing = append(missing, "Domain")
	}
	if c.ViewState == nil {
		missing = append(missing, "ViewState")
	}

	if len(missing) > 0 {
		return errors.New("state.Config: missing required fields: " + strings.Join(missing, ", "))
	}

	return nil
}

// deps returns the Deps with defaults applied.
func (c *Config) deps() *Deps {
	c.Deps.applyDefaults()

	return &c.Deps
}

// NewCommand creates a new state command with all subcommands.
// Returns an error if required configuration is missing.
//
// Usage:
//
//	cmd, err := state.NewCommand(state.Config{
//	    Logger:    lggr,
//	    Domain:    myDomain,
//	    ViewState: myViewStateFunc,
//	})
//	if err != nil {
//	    return err
//	}
//	rootCmd.AddCommand(cmd)
func NewCommand(cfg Config) (*cobra.Command, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cfg.deps()

	cmd := &cobra.Command{
		Use:   "state",
		Short: stateShort,
		Long:  stateLong,
	}

	cmd.AddCommand(newGenerateCmd(cfg))

	return cmd, nil
}
