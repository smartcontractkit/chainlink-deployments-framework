package decoder

import (
	"context"
	"errors"
	"fmt"

	"github.com/smartcontractkit/mcms"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	experimentalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

type reportBuilderFunc func(
	ctx context.Context,
	env deployment.Environment,
	proposal *mcms.TimelockProposal,
) (*experimentalanalyzer.ProposalReport, error)

type ExperimentalDecoder struct {
	config      Config
	buildReport reportBuilderFunc
}

var _ ProposalDecoder = (*ExperimentalDecoder)(nil)

// NewExperimentalDecoder creates a ProposalDecoder backed by the experimental
// analyzer.
func NewExperimentalDecoder(cfg Config) *ExperimentalDecoder {
	return &ExperimentalDecoder{
		config:      cfg,
		buildReport: newDefaultReportBuilder(cfg),
	}
}

// newDefaultReportBuilder creates the report builder that
// creates a ProposalContext from the config.
func newDefaultReportBuilder(cfg Config) reportBuilderFunc {
	return func(
		ctx context.Context,
		env deployment.Environment,
		proposal *mcms.TimelockProposal,
	) (*experimentalanalyzer.ProposalReport, error) {
		proposalCtx, err := experimentalanalyzer.NewDefaultProposalContext(env,
			experimentalanalyzer.WithEVMABIMappings(cfg.EVMABIMappings),
			experimentalanalyzer.WithSolanaDecoders(cfg.SolanaDecoders),
		)
		if err != nil {
			return nil, fmt.Errorf("creating proposal context: %w", err)
		}

		return experimentalanalyzer.BuildTimelockReport(ctx, proposalCtx, env, proposal)
	}
}

func (d *ExperimentalDecoder) Decode(
	ctx context.Context,
	env deployment.Environment,
	proposal *mcms.TimelockProposal,
) (DecodedTimelockProposal, error) {
	if proposal == nil {
		return nil, errors.New("proposal cannot be nil")
	}

	report, err := d.buildReport(ctx, env, proposal)
	if err != nil {
		return nil, fmt.Errorf("building timelock report: %w", err)
	}
	if report == nil {
		return nil, errors.New("report builder returned a nil report")
	}

	return adaptTimelockProposal(report, proposal), nil
}
