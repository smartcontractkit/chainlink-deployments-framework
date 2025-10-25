package analyzer

// Renderer turns a ProposalReport into a concrete output format.
// Implementations should be format-specific (Markdown/HTML/Text).
type Renderer interface {
	RenderProposal(r *ProposalReport, ctx *FieldContext) string
	RenderTimelockProposal(r *ProposalReport, ctx *FieldContext) string
	RenderDecodedCall(d *DecodedCall, ctx *FieldContext) string
	RenderField(field NamedField, ctx *FieldContext) string
}
