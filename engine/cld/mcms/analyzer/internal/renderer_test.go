package internal

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer"
)

// Mock implementations for testing
type mockDecodedParameter struct {
	name  string
	ptype string
	value any
}

func (m mockDecodedParameter) Name() string { return m.name }
func (m mockDecodedParameter) Type() string { return m.ptype }
func (m mockDecodedParameter) Value() any   { return m.value }

type mockDecodedCall struct {
	name    string
	inputs  analyzer.DecodedParameters
	outputs analyzer.DecodedParameters
}

func (m mockDecodedCall) Name() string                        { return m.name }
func (m mockDecodedCall) ContractType() string                { return "" }
func (m mockDecodedCall) ContractVersion() string             { return "" }
func (m mockDecodedCall) To() string                          { return "" }
func (m mockDecodedCall) Inputs() analyzer.DecodedParameters  { return m.inputs }
func (m mockDecodedCall) Outputs() analyzer.DecodedParameters { return m.outputs }
func (m mockDecodedCall) Data() []byte                        { return nil }
func (m mockDecodedCall) AdditionalFields() json.RawMessage   { return nil }

type mockDecodedBatchOperation struct {
	calls analyzer.DecodedCalls
}

func (m mockDecodedBatchOperation) ChainSelector() uint64        { return 0 }
func (m mockDecodedBatchOperation) Calls() analyzer.DecodedCalls { return m.calls }

type mockDecodedTimelockProposal struct {
	batchOps analyzer.DecodedBatchOperations
}

func (m mockDecodedTimelockProposal) BatchOperations() analyzer.DecodedBatchOperations {
	return m.batchOps
}

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
	renderer, err := NewRenderer()
	require.NoError(t, err)

	proposal := &analyzedProposal{
		annotated: &annotated{
			annotations: []analyzer.Annotation{
				NewAnnotation("test.annotation", "string", "test value"),
				SeverityAnnotation("warning"),
				RiskAnnotation("medium"),
			},
		},
		decodedProposal: mockDecodedTimelockProposal{},
		batchOperations: nil,
	}

	output, err := renderer.Render(proposal)
	require.NoError(t, err)
	assert.Contains(t, output, "ANALYZED PROPOSAL")
	assert.Contains(t, output, "Annotations:")
	assert.Contains(t, output, "test.annotation")
	assert.Contains(t, output, "test value")
	assert.Contains(t, output, "cld.severity")
	assert.Contains(t, output, "warning")
	assert.Contains(t, output, "cld.risk")
	assert.Contains(t, output, "medium")
}

func TestRenderer_Render_CompleteProposal(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	// Create a complete analyzed proposal
	param1 := &analyzedParameter{
		annotated: &annotated{
			annotations: []analyzer.Annotation{
				NewAnnotation("param.note", "string", "important parameter"),
			},
		},
		decodedParameter: mockDecodedParameter{
			name:  "recipient",
			ptype: "address",
			value: "0x1234567890abcdef",
		},
	}

	param2 := &analyzedParameter{
		annotated: &annotated{},
		decodedParameter: mockDecodedParameter{
			name:  "amount",
			ptype: "uint256",
			value: "1000000000000000000",
		},
	}

	outputParam := &analyzedParameter{
		annotated: &annotated{},
		decodedParameter: mockDecodedParameter{
			name:  "success",
			ptype: "bool",
			value: true,
		},
	}

	call1 := &analyzedCall{
		annotated: &annotated{
			annotations: []analyzer.Annotation{
				SeverityAnnotation("info"),
			},
		},
		decodedCall: mockDecodedCall{
			name: "transfer",
		},
		inputs:  []analyzer.AnalyzedParameter{param1, param2},
		outputs: []analyzer.AnalyzedParameter{outputParam},
	}

	batchOp := &analyzedBatchOperation{
		annotated: &annotated{
			annotations: []analyzer.Annotation{
				RiskAnnotation("low"),
			},
		},
		decodedBatchOperation: mockDecodedBatchOperation{},
		calls:                 []analyzer.AnalyzedCall{call1},
	}

	proposal := &analyzedProposal{
		annotated: &annotated{
			annotations: []analyzer.Annotation{
				NewAnnotation("proposal.id", "string", "PROP-001"),
			},
		},
		decodedProposal: mockDecodedTimelockProposal{},
		batchOperations: []analyzer.AnalyzedBatchOperation{batchOp},
	}

	output, err := renderer.Render(proposal)
	require.NoError(t, err)

	// Verify proposal level
	assert.Contains(t, output, "ANALYZED PROPOSAL")
	assert.Contains(t, output, "proposal.id")
	assert.Contains(t, output, "PROP-001")
	assert.Contains(t, output, "Batch Operations: 1")

	// Verify batch operation level
	assert.Contains(t, output, "BATCH OPERATION")
	assert.Contains(t, output, "cld.risk")
	assert.Contains(t, output, "low")
	assert.Contains(t, output, "Calls: 1")

	// Verify call level
	assert.Contains(t, output, "CALL: transfer")
	assert.Contains(t, output, "cld.severity")
	assert.Contains(t, output, "info")
	assert.Contains(t, output, "Inputs (2)")
	assert.Contains(t, output, "Outputs (1)")

	// Verify parameter level
	assert.Contains(t, output, "recipient (address): 0x1234567890abcdef")
	assert.Contains(t, output, "amount (uint256): 1000000000000000000")
	assert.Contains(t, output, "success (bool): true")
	assert.Contains(t, output, "param.note")
	assert.Contains(t, output, "important parameter")
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
		"annotations":    `{{define "annotations"}}ANNOTATIONS{{end}}`,
	}

	renderer, err := NewRendererWithTemplates(customTemplates)
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
