package proposalanalysis

import (
	"context"
	"io"

	"github.com/smartcontractkit/mcms"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/renderer"
	experimentalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

type DecodeInstructionFn = experimentalanalyzer.DecodeInstructionFn

// DecoderConfig configures the proposal decoder used by the analyzer engine.
// The decoder is expected to support multi-chain proposal decoding using the
// provided chain-specific mappings.
type DecoderConfig struct {
	EVMABIMappings map[string]string
	SolanaDecoders map[string]DecodeInstructionFn
}

type AnalyzerEngine interface {
	Run(ctx context.Context, domain cldfdomain.Domain, env deployment.Environment, proposal *mcms.TimelockProposal) (analyzer.AnalyzedProposal, error)

	RegisterAnalyzer(analyzer analyzer.BaseAnalyzer) error

	RegisterRenderer(renderer renderer.Renderer) error

	RenderTo(w io.Writer, rendererID string, proposal analyzer.AnalyzedProposal) error
}
