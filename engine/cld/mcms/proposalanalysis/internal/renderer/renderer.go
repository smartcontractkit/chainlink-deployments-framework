package renderer

import (
	"io"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer"
)

// RenderRequest encapsulates the domain and environment name
type RenderRequest struct {
	Domain          string
	EnvironmentName string
}

// Renderer transforms an AnalyzedProposal into a specific output format
type Renderer interface {
	ID() string
	RenderTo(w io.Writer, req RenderRequest, proposal analyzer.AnalyzedProposal) error
}
