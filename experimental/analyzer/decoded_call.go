package analyzer

import (
	"fmt"
	"strings"
)

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
	// For now, return a simple format. In the future, this should use a renderer.
	var result strings.Builder

	result.WriteString(fmt.Sprintf("Address: %s\n", d.Address))
	result.WriteString(fmt.Sprintf("Method: %s\n", d.Method))

	if len(d.Inputs) > 0 {
		result.WriteString("Inputs:\n")
		for _, input := range d.Inputs {
			result.WriteString(fmt.Sprintf("  %s: <field value>\n", input.Name))
		}
	}

	if len(d.Outputs) > 0 {
		result.WriteString("Outputs:\n")
		for _, output := range d.Outputs {
			result.WriteString(fmt.Sprintf("  %s: <field value>\n", output.Name))
		}
	}

	return result.String()
}
