package renderer

import (
	"io"

	"github.com/smartcontractkit/mcms"
)

// RenderRequest encapsulates the domain, environment, and optional source proposal.
type RenderRequest struct {
	Domain          string
	EnvironmentName string
	// TimelockProposal is the original MCMS timelock proposal used during analysis.
	// It may be nil in tests or when callers omit proposal metadata.
	// Callers should pass the same instance supplied to AnalyzerEngine.Run.
	TimelockProposal *mcms.TimelockProposal
}

// Renderer transforms an AnalyzedProposal into a specific output format
type Renderer interface {
	ID() string
	RenderTo(w io.Writer, req RenderRequest, proposal AnalyzedProposal) error
}
