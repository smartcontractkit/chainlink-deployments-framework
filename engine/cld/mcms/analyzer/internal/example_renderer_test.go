package internal_test

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer/internal"
)

// ExampleRenderer demonstrates how to use the Renderer to display an AnalyzedProposal.
func ExampleRenderer() {
	// Create a new renderer with default templates
	renderer, err := internal.NewRenderer()
	if err != nil {
		panic(err)
	}

	// Create an analyzed proposal (simplified for example)
	// In practice, this would come from the analyzer engine
	proposal := createExampleProposal()

	// Render the proposal to a string
	output, err := renderer.Render(proposal)
	if err != nil {
		panic(err)
	}

	// Display or write the output
	fmt.Println(output)
}

// ExampleRenderer_customTemplates demonstrates how to use custom templates.
func ExampleRenderer_customTemplates() {
	// Define custom templates
	customTemplates := map[string]string{
		"proposal": `{{define "proposal"}}
=== CUSTOM PROPOSAL REPORT ===
Batch Operations: {{len .BatchOperations}}
{{range .BatchOperations}}{{template "batchOperation" .}}{{end}}
{{end}}`,
		"batchOperation": `{{define "batchOperation"}}
Batch Operation - Calls: {{len .Calls}}
{{end}}`,
		"call":        `{{define "call"}}Call: {{.Name}}{{end}}`,
		"parameter":   `{{define "parameter"}}{{.Name}}: {{.Value}}{{end}}`,
		"annotations": `{{define "annotations"}}{{end}}`,
	}

	// Create renderer with custom templates
	renderer, err := internal.NewRendererWithTemplates(customTemplates)
	if err != nil {
		panic(err)
	}

	proposal := createExampleProposal()

	// Render with custom templates
	output, err := renderer.Render(proposal)
	if err != nil {
		panic(err)
	}

	fmt.Println(output)
}

// createExampleProposal creates a sample analyzed proposal for examples.
// This is a placeholder - real implementations would use actual data.
func createExampleProposal() analyzer.AnalyzedProposal {
	// Note: This is simplified for the example
	// In real usage, this would be created by the analyzer engine
	return nil
}
