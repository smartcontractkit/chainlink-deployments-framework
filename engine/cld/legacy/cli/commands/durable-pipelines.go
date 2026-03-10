package commands

import (
	"github.com/spf13/cobra"

	fresolvers "github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/pipeline"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

// NewDurablePipelineCmds delegates to the modular pipeline command package.
// Preserved for backward compatibility.
func (c Commands) NewDurablePipelineCmds(
	domain domain.Domain,
	loadChangesets func(envName string) (*changeset.ChangesetsRegistry, error),
	decodeProposalCtxProvider func(env fdeployment.Environment) (analyzer.ProposalContext, error),
	loadConfigResolvers *fresolvers.ConfigResolverManager) *cobra.Command {
	cmd, err := pipeline.NewCommand(&pipeline.Config{
		Logger:                    c.lggr,
		Domain:                    domain,
		LoadChangesets:            loadChangesets,
		DecodeProposalCtxProvider: decodeProposalCtxProvider,
		ConfigResolverManager:     loadConfigResolvers,
	})
	if err != nil {
		panic(err)
	}

	return cmd
}
