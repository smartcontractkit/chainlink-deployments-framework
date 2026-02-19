package proposalanalysis

import (
	"context"
	"io"

	"github.com/smartcontractkit/mcms"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/renderer"
)

type AnalyzerEngine interface {
	Run(ctx context.Context, domain cldfdomain.Domain, env deployment.Environment, proposal *mcms.TimelockProposal) (analyzer.AnalyzedProposal, error)

	RegisterAnalyzer(analyzer analyzer.BaseAnalyzer) error

	RegisterRenderer(renderer renderer.Renderer) error

	RenderTo(w io.Writer, rendererID string, proposal analyzer.AnalyzedProposal) error
}
