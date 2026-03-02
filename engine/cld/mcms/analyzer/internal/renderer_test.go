package internal

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer"
)

func TestNewRenderer(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)
	require.NotNil(t, renderer)
	require.NotNil(t, renderer.tmpl)
}

func TestRenderer_Render_EmptyProposal(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	proposal := &analyzedProposal{
		annotated:       &annotated{},
		decodedProposal: mockDecodedTimelockProposal{},
		batchOperations: nil,
	}

	output, err := renderer.Render(proposal)
	require.NoError(t, err)
	assert.Contains(t, output, "ANALYZED PROPOSAL")
	assert.Contains(t, output, "No batch operations found")
}

func TestRenderer_Render_ProposalWithAnnotations(t *testing.T) {
	t.Skip("Skipping - annotations now drive rendering behavior, not displayed separately")
	// Note: For annotation-driven rendering tests, see renderer_enhanced_test.go
}

func TestRenderer_Render_CompleteProposal(t *testing.T) {
	t.Skip("Skipping - annotations now drive rendering behavior, not displayed separately")
	// Note: For annotation-driven rendering tests, see renderer_enhanced_test.go
}

func TestRenderer_Render_MultipleBatchOperations(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	call1 := &analyzedCall{
		annotated:   &annotated{},
		decodedCall: mockDecodedCall{name: "setConfig"},
		inputs:      nil,
		outputs:     nil,
	}

	call2 := &analyzedCall{
		annotated:   &annotated{},
		decodedCall: mockDecodedCall{name: "unpause"},
		inputs:      nil,
		outputs:     nil,
	}

	batchOp1 := &analyzedBatchOperation{
		annotated:             &annotated{},
		decodedBatchOperation: mockDecodedBatchOperation{},
		calls:                 []analyzer.AnalyzedCall{call1},
	}

	batchOp2 := &analyzedBatchOperation{
		annotated:             &annotated{},
		decodedBatchOperation: mockDecodedBatchOperation{},
		calls:                 []analyzer.AnalyzedCall{call2},
	}

	proposal := &analyzedProposal{
		annotated:       &annotated{},
		decodedProposal: mockDecodedTimelockProposal{},
		batchOperations: []analyzer.AnalyzedBatchOperation{batchOp1, batchOp2},
	}

	output, err := renderer.Render(proposal)
	require.NoError(t, err)

	assert.Contains(t, output, "Batch Operations: 2")
	assert.Contains(t, output, "CALL: setConfig")
	assert.Contains(t, output, "CALL: unpause")

	// Verify both batch operations are rendered
	batchOpCount := strings.Count(output, "BATCH OPERATION")
	assert.Equal(t, 2, batchOpCount)
}

func TestNewRendererWithTemplates_CustomTemplate(t *testing.T) {
	customTemplates := map[string]string{
		"proposal":       `{{define "proposal"}}CUSTOM PROPOSAL: {{len .BatchOperations}} batch ops{{end}}`,
		"batchOperation": `{{define "batchOperation"}}CUSTOM BATCH OP{{end}}`,
		"call":           `{{define "call"}}CUSTOM CALL: {{.Name}}{{end}}`,
		"parameter":      `{{define "parameter"}}{{.Name}}={{.Value}}{{end}}`,
	}

	renderer, err := NewRendererWithTemplates(FormatText, customTemplates)
	require.NoError(t, err)

	proposal := &analyzedProposal{
		annotated:       &annotated{},
		decodedProposal: mockDecodedTimelockProposal{},
		batchOperations: nil,
	}

	output, err := renderer.Render(proposal)
	require.NoError(t, err)
	assert.Contains(t, output, "CUSTOM PROPOSAL: 0 batch ops")
}

func TestTemplateFuncs_Indent(t *testing.T) {
	funcs := templateFuncs()
	indentFunc := funcs["indent"].(func(int, string) string)

	input := "line1\nline2\nline3"
	expected := "  line1\n  line2\n  line3"
	result := indentFunc(2, input)
	assert.Equal(t, expected, result)
}

func TestTemplateFuncs_HasAnnotations(t *testing.T) {
	funcs := templateFuncs()
	hasAnnotationsFunc := funcs["hasAnnotations"].(func(analyzer.Annotated) bool)

	// Test with nil
	assert.False(t, hasAnnotationsFunc(nil))

	// Test with no annotations
	annotated1 := &annotated{annotations: nil}
	assert.False(t, hasAnnotationsFunc(annotated1))

	// Test with annotations
	annotated2 := &annotated{
		annotations: []analyzer.Annotation{
			NewAnnotation("test", "string", "value"),
		},
	}
	assert.True(t, hasAnnotationsFunc(annotated2))
}

func TestTemplateFuncs_SeverityAndRiskSymbols(t *testing.T) {
	funcs := templateFuncs()
	severitySymbol := funcs["severitySymbol"].(func(string) string)
	riskSymbol := funcs["riskSymbol"].(func(string) string)

	// Test severity symbols
	assert.Equal(t, "✗", severitySymbol("error"))
	assert.Equal(t, "⚠", severitySymbol("warning"))
	assert.Equal(t, "ℹ", severitySymbol("info"))
	assert.Equal(t, "⚙", severitySymbol("debug"))
	assert.Equal(t, "?", severitySymbol("unknown"))
	assert.Equal(t, "?", severitySymbol("invalid"))

	// Test risk symbols
	assert.Equal(t, "🔴", riskSymbol("high"))
	assert.Equal(t, "🟡", riskSymbol("medium"))
	assert.Equal(t, "🟢", riskSymbol("low"))
	assert.Equal(t, "⚪", riskSymbol("unknown"))
	assert.Equal(t, "⚪", riskSymbol("invalid"))
}
