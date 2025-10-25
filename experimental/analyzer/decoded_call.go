package analyzer

const (
	// Magic number constants
	MinStructFieldsForPrettyFormat = 2
	MinDataLengthForMethodID       = 4
	DefaultAnalyzersCount          = 2
)

type DecodedCall struct {
	Address string
	Method  string
	Inputs  []NamedField
	Outputs []NamedField
}

// String renders a human-readable representation of the decoded call using the default text renderer.
// This method is kept for backwards compatibility but rendering should be done through renderers.
func (d *DecodedCall) String(context *FieldContext) string {
	// Use the text renderer to provide proper formatting
	renderer := NewTextRenderer()
	return renderer.RenderDecodedCall(d, context)
}
