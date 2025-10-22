package analyzer

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const (
	// Magic number constants
	MinStructFieldsForPrettyFormat = 2
	MinDataLengthForMethodID       = 4
	DefaultAnalyzersCount          = 2
)

// Analyzer is an extension point of proposal decoding.
// You can implement your own Analyzer which returns your own Descriptor instance.
type Analyzer func(argName string, argAbi *abi.Type, argVal any, analyzers []Analyzer) Descriptor

type DecodedCall struct {
	Address string
	Method  string
	Inputs  []NamedDescriptor
	Outputs []NamedDescriptor
}

// Describe renders a human-readable representation of the decoded call in a format-neutral way.
// This method returns a simple string representation that can be used by any renderer.
func (d *DecodedCall) Describe(context *DescriptorContext) string {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("Address: %s\n", d.Address))
	result.WriteString(fmt.Sprintf("Method: %s\n", d.Method))

	if len(d.Inputs) > 0 {
		result.WriteString("Inputs:\n")
		for _, input := range d.Inputs {
			result.WriteString(fmt.Sprintf("  %s: %s\n", input.Name, input.Value.Describe(context)))
		}
	}

	if len(d.Outputs) > 0 {
		result.WriteString("Outputs:\n")
		for _, output := range d.Outputs {
			result.WriteString(fmt.Sprintf("  %s: %s\n", output.Name, output.Value.Describe(context)))
		}
	}

	return result.String()
}
