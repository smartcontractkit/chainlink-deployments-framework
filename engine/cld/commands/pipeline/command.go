package pipeline

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"

	fresolvers "github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cs "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/text"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

var (
	pipelineShort = "Durable Pipeline commands"

	pipelineLong = text.LongDesc(`
		Commands for running and managing durable pipeline changesets.

		Durable pipelines apply changesets against environments, resolve config
		via registered resolvers, and persist artifacts.
	`)
)

// Config holds the configuration for pipeline commands.
type Config struct {
	Logger                    logger.Logger
	Domain                    domain.Domain
	LoadChangesets            func(envName string) (*cs.ChangesetsRegistry, error)
	DecodeProposalCtxProvider func(env fdeployment.Environment) (analyzer.ProposalContext, error) // optional for run
	ConfigResolverManager     *fresolvers.ConfigResolverManager

	Deps Deps
}

// Validate checks that all required configuration fields are set.
func (c *Config) Validate() error {
	var missing []string

	if c.Logger == nil {
		missing = append(missing, "Logger")
	}
	if c.Domain.RootPath() == "" {
		missing = append(missing, "Domain")
	}
	if c.LoadChangesets == nil {
		missing = append(missing, "LoadChangesets")
	}
	if c.ConfigResolverManager == nil {
		missing = append(missing, "ConfigResolverManager")
	}

	if len(missing) > 0 {
		return errors.New("pipeline.Config: missing required fields: " + strings.Join(missing, ", "))
	}

	return nil
}

// deps returns Deps with defaults applied.
func (c *Config) deps() *Deps {
	c.Deps.applyDefaults()
	return &c.Deps
}

// NewCommand creates a new pipeline command with all subcommands.
func NewCommand(cfg *Config) (*cobra.Command, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cfg.deps()

	cmd := &cobra.Command{
		Use:     "pipeline",
		Aliases: []string{"durable-pipeline"},
		Short:   pipelineShort,
		Long:    pipelineLong,
	}

	cmd.AddCommand(
		newRunCmd(cfg),
		newInputGenerateCmd(cfg),
		newListCmd(cfg),
		newTemplateInputCmd(cfg),
	)

	return cmd, nil
}
