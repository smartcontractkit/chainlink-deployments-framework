package datastore

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/text"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

var (
	datastoreShort = "Datastore operations"

	datastoreLong = text.LongDesc(`
		Commands for managing datastore artifacts.

		The datastore contains contract addresses and metadata for deployed contracts.
		These commands allow merging changeset artifacts and syncing to the catalog service.
	`)
)

// Config holds the configuration for datastore commands.
type Config struct {
	// Logger is the logger to use for command output. Required.
	Logger logger.Logger

	// Domain is the domain context for the commands. Required.
	Domain domain.Domain

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

	if len(missing) > 0 {
		return errors.New("datastore.Config: missing required fields: " + strings.Join(missing, ", "))
	}

	return nil
}

// deps returns the Deps with defaults applied.
func (c *Config) deps() *Deps {
	c.Deps.applyDefaults()

	return &c.Deps
}

// NewCommand creates a new datastore command with all subcommands.
func NewCommand(cfg Config) (*cobra.Command, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cfg.deps()

	cmd := &cobra.Command{
		Use:   "datastore",
		Short: datastoreShort,
		Long:  datastoreLong,
	}

	cmd.AddCommand(newMergeCmd(cfg))
	cmd.AddCommand(newSyncToCatalogCmd(cfg))

	return cmd, nil
}
