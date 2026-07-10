package renderer

import (
	"io"

	"github.com/smartcontractkit/mcms"
)

// RenderRequest encapsulates the domain, environment, and optional source proposal.
type RenderRequest struct {
	Domain          string
	EnvironmentName string
	// TimelockProposal carries MCMS timelock metadata for rendering.
	// AnalyzerEngine.RenderTo clones this value before invoking renderers.
	// It may be nil in tests or when callers omit proposal metadata.
	TimelockProposal *mcms.TimelockProposal
}

// Renderer transforms an AnalyzedProposal into a specific output format
type Renderer interface {
	ID() string
	RenderTo(w io.Writer, req RenderRequest, proposal AnalyzedProposal) error
}
