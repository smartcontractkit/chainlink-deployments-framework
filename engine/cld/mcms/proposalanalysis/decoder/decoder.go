package decoder

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/mcms"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"
	experimentalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

// ProposalDecoder decodes MCMS proposals into structured DecodedTimelockProposal
type ProposalDecoder interface {
	Decode(ctx context.Context, env deployment.Environment, proposal *mcms.TimelockProposal) (types.DecodedTimelockProposal, error)
}

// legacyDecoder adapts the legacy experimental/analyzer package to the new decoder interface
type legacyDecoder struct {
	proposalContext experimentalanalyzer.ProposalContext
}

// NewLegacyDecoder creates a decoder that wraps legacy experimental/analyzer decoding logic.
// Use functional options to configure:
//   - WithProposalContext: provide a custom ProposalContext (otherwise default is created)
func NewLegacyDecoder(opts ...DecoderOption) ProposalDecoder {
	decoder := &legacyDecoder{}

	for _, opt := range opts {
		opt(decoder)
	}

	return decoder
}

// DecoderOption is a functional option for configuring the decoder
type DecoderOption func(*legacyDecoder)

// WithProposalContext injects a custom ProposalContext for decoding.
// If not provided, a default context will be created during decoding.
func WithProposalContext(ctx experimentalanalyzer.ProposalContext) DecoderOption {
	return func(d *legacyDecoder) {
		d.proposalContext = ctx
	}
}

func (d *legacyDecoder) Decode(
	ctx context.Context,
	env deployment.Environment,
	proposal *mcms.TimelockProposal,
) (types.DecodedTimelockProposal, error) {
	// Create proposal context for legacy experimental analyzer
	// Use the provided context if available, otherwise create a default one
	var proposalCtx experimentalanalyzer.ProposalContext

	if d.proposalContext != nil {
		proposalCtx = d.proposalContext
	} else {
		var err error
		proposalCtx, err = experimentalanalyzer.NewDefaultProposalContext(env)
		if err != nil {
			return nil, fmt.Errorf("failed to create proposal context: %w", err)
		}
	}

	// Build the report using legacy experimental analyzer
	report, err := experimentalanalyzer.BuildTimelockReport(ctx, proposalCtx, env, proposal)
	if err != nil {
		return nil, fmt.Errorf("failed to build timelock report: %w", err)
	}

	// Convert to our DecodedTimelockProposal interface
	return &decodedTimelockProposal{
		report: report,
	}, nil
}
