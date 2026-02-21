package internal_test

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer/internal"
)

// ExampleRenderer demonstrates how to use the Renderer to display an AnalyzedProposal in text format.
func ExampleRenderer() {
	// Create a new renderer with default text format
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

// ExampleRenderer_html demonstrates rendering in HTML format.
func ExampleRenderer_html() {
	// Create a renderer with HTML format
	renderer, err := internal.NewRendererWithFormat(internal.FormatHTML)
	if err != nil {
		panic(err)
	}

	proposal := createExampleProposal()

	// Render to HTML
	output, err := renderer.Render(proposal)
	if err != nil {
		panic(err)
	}

	// Write to file
	err = renderer.RenderToFile("proposal.html", proposal)
	if err != nil {
		panic(err)
	}

	fmt.Println("HTML output generated:", len(output), "bytes")
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
		"call":      `{{define "call"}}Call: {{.Name}}{{end}}`,
		"parameter": `{{define "parameter"}}{{.Name}}: {{.Value}}{{end}}`,
	}

	// Create renderer with custom templates
	renderer, err := internal.NewRendererWithTemplates(internal.FormatText, customTemplates)
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

// ExampleRenderer_annotationDriven demonstrates annotation-driven rendering.
func ExampleRenderer_annotationDriven() {
	// This example shows how annotations control rendering behavior
	// Note: In real usage, analyzers would add these annotations

	// The proposal would have calls with important annotations
	// that cause the renderer to highlight them

	renderer, err := internal.NewRenderer()
	if err != nil {
		panic(err)
	}

	// In an actual analyzed proposal:
	// - Parameters with "render.formatter=ethereum.address" would be formatted as 0x... addresses
	// - Calls with "render.important=true" would be marked with ⭐
	// - Parameters with "render.emoji=💰" would display the emoji
	// - Values with "cld.severity=warning" would show ⚠ symbol
	// - Values with "cld.risk=high" would show 🔴 symbol

	proposal := createExampleProposal()
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
