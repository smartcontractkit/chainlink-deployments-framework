// Package commands provides modular CLI command packages for domain CLIs.
//
// There are two ways to use commands from this package:
//
// 1. Via the Commands factory (recommended for most use cases):
//
//	commands := commands.New(lggr)
//	stateCmd, err := commands.State(domain, stateConfig)
//	if err != nil {
//	    return err
//	}
//	app.AddCommand(stateCmd)
//
// 2. Via direct package imports (for advanced DI/testing):
//
//	import "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/state"
//
//	cmd, err := state.NewCommand(state.Config{
//	    Logger:    lggr,
//	    Domain:    domain,
//	    ViewState: myViewState,
//	    Deps:      state.Deps{...},  // inject mocks for testing
//	})
//	if err != nil {
//	    return err
//	}
//	app.AddCommand(cmd)
package commands

import (
	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/addressbook"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/mcms"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/state"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
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
func (c *Commands) State(dom domain.Domain, cfg StateConfig) (*cobra.Command, error) {
	return state.NewCommand(state.Config{
		Logger:    c.lggr,
		Domain:    dom,
		ViewState: cfg.ViewState,
	})
}

// Datastore creates the datastore command group.
func (c *Commands) Datastore(dom domain.Domain) (*cobra.Command, error) {
	return datastore.NewCommand(datastore.Config{
		Logger: c.lggr,
		Domain: dom,
	})
}

// AddressBook creates the address-book command group.
func (c *Commands) AddressBook(dom domain.Domain) (*cobra.Command, error) {
	return addressbook.NewCommand(addressbook.Config{
		Logger: c.lggr,
		Domain: dom,
	})
}

// MCMSConfig holds configuration for MCMS commands.
type MCMSConfig struct {
	// ProposalContextProvider creates proposal context for analysis.
	// This is domain-specific and must be provided by the user.
	ProposalContextProvider analyzer.ProposalContextProvider
}

// MCMS creates the mcms command group for proposal analysis and conversion.
func (c *Commands) MCMS(dom domain.Domain, cfg MCMSConfig) (*cobra.Command, error) {
	return mcms.NewCommand(mcms.Config{
		Logger:                  c.lggr,
		Domain:                  dom,
		ProposalContextProvider: cfg.ProposalContextProvider,
	})
}
