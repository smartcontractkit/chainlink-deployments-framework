package analyzer

// Renderer turns a ProposalReport into a concrete output format.
// Implementations should be format-specific (Markdown/HTML/Text).
type Renderer interface {
	RenderProposal(r *ProposalReport, ctx *DescriptorContext) string
	RenderTimelockProposal(r *ProposalReport, ctx *DescriptorContext) string
	RenderDecodedCall(d *DecodedCall, ctx *DescriptorContext) string
}
