package mcms

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/text"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

var (
	mcmsShort = "MCMS proposal commands"

	mcmsLong = text.LongDesc(`
		Commands for managing MCMS proposals.

		These commands provide functionality for analyzing proposals, decoding errors,
		converting to UPF format, and executing proposals on forked environments.
	`)
)

// Config holds the configuration for mcms commands.
type Config struct {
	// Logger is the logger to use for command output. Required.
	Logger logger.Logger

	// Domain is the domain context for the commands. Required.
	Domain domain.Domain

	// ProposalContextProvider creates proposal context for analysis. Required.
	ProposalContextProvider analyzer.ProposalContextProvider

	// Deps holds optional dependencies that can be overridden.
	// If fields are nil, production defaults are used.
	Deps Deps
}

// Validate checks that all required configuration fields are set.
func (c Config) Validate() error {
	var missing []string

	if c.Logger == nil {
		missing = append(missing, "Logger")
	}
	if c.Domain.RootPath() == "" {
		missing = append(missing, "Domain")
	}
	if c.ProposalContextProvider == nil {
		missing = append(missing, "ProposalContextProvider")
	}

	if len(missing) > 0 {
		return errors.New("mcms.Config: missing required fields: " + strings.Join(missing, ", "))
	}

	return nil
}

// deps returns the Deps with defaults applied.
func (c *Config) deps() *Deps {
	c.Deps.applyDefaults()

	return &c.Deps
}

// NewCommand creates a new mcms command with all subcommands.
func NewCommand(cfg Config) (*cobra.Command, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cfg.deps()

	cmd := &cobra.Command{
		Use:   "mcms",
		Short: mcmsShort,
		Long:  mcmsLong,
	}

	cmd.AddCommand(newErrorDecodeCmd(cfg))
	cmd.AddCommand(newAnalyzeProposalCmd(cfg))
	cmd.AddCommand(newConvertUpfCmd(cfg))
	cmd.AddCommand(newExecuteForkCmd(cfg))

	return cmd, nil
}
