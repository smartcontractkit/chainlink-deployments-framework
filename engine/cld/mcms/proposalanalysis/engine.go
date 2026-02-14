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

type AnalyzerEngine interface {
	Run(ctx context.Context, domain cldfdomain.Domain, env deployment.Environment, proposal *mcms.TimelockProposal) (analyzer.AnalyzedProposal, error)

	RegisterAnalyzer(analyzer analyzer.BaseAnalyzer) error

	RegisterRenderer(renderer renderer.Renderer) error

	RenderTo(w io.Writer, rendererID string, proposal analyzer.AnalyzedProposal) error

	RegisterEVMABIMappings(evmABIMappings map[string]string) error

	RegisterSolanaDecoders(solanaDecoders map[string]DecodeInstructionFn) error
}
