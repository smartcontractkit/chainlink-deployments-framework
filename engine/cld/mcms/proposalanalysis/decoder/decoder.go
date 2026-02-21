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
	evmABIMappings map[string]string
	solanaDecoders map[string]experimentalanalyzer.DecodeInstructionFn
}

// NewLegacyDecoder creates a decoder that wraps legacy experimental/analyzer decoding logic.
// Use functional options to configure:
//   - WithEVMABIMappings: override proposal context EVM ABI mappings
//   - WithSolanaDecoders: override proposal context Solana decoder mappings
func NewLegacyDecoder(opts ...DecoderOption) ProposalDecoder {
	decoder := &legacyDecoder{}

	for _, opt := range opts {
		opt(decoder)
	}

	return decoder
}

// DecoderOption is a functional option for configuring the decoder
type DecoderOption func(*legacyDecoder)

// WithEVMABIMappings overrides the proposal context EVM ABI mappings used during decoding.
func WithEVMABIMappings(mappings map[string]string) DecoderOption {
	return func(d *legacyDecoder) {
		d.evmABIMappings = mappings
	}
}

// WithSolanaDecoders overrides the proposal context Solana decoder mappings used during decoding.
func WithSolanaDecoders(decoders map[string]experimentalanalyzer.DecodeInstructionFn) DecoderOption {
	return func(d *legacyDecoder) {
		d.solanaDecoders = decoders
	}
}

func (d *legacyDecoder) Decode(
	ctx context.Context,
	env deployment.Environment,
	proposal *mcms.TimelockProposal,
) (types.DecodedTimelockProposal, error) {
	proposalCtx, err := experimentalanalyzer.NewDefaultProposalContext(env,
		experimentalanalyzer.WithEVMABIMappings(d.evmABIMappings),
		experimentalanalyzer.WithSolanaDecoders(d.solanaDecoders),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create proposal context: %w", err)
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
